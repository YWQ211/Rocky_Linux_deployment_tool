package core

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"rockdev/internal/i18n"
	"strings"
	"time"
)

type Mirror struct{ ID, Name, URL string }

var Mirrors = []Mirror{{"official", "Rocky 官方", "https://dl.rockylinux.org/pub/rocky"}, {"aliyun", "阿里云", "https://mirrors.aliyun.com/rockylinux"}, {"tuna", "清华 TUNA", "https://mirrors.tuna.tsinghua.edu.cn/rocky"}, {"ustc", "中国科大", "https://mirrors.ustc.edu.cn/rocky"}}

func FindMirror(id string) (Mirror, error) {
	for _, m := range Mirrors {
		if m.ID == id {
			return m, nil
		}
	}
	return Mirror{}, i18n.Errorf("未知镜像 %q，请运行 mirror list", id)
}

func Probe(ctx context.Context, client *http.Client, base string, s System) error {
	for _, repo := range []string{"BaseOS", "AppStream", "extras"} {
		url := fmt.Sprintf("%s/%s/%s/%s/os/repodata/repomd.xml", strings.TrimRight(base, "/"), s.Major, repo, s.Arch)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return err
		}
		resp, err := client.Do(req)
		if err != nil {
			return i18n.Errorf("%s: %w", url, err)
		}
		var meta struct {
			XMLName xml.Name
			Data    []struct {
				Type     string `xml:"type,attr"`
				Location struct {
					Href string `xml:"href,attr"`
				} `xml:"location"`
			} `xml:"data"`
		}
		err = xml.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(&meta)
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return i18n.Errorf("%s: HTTP %d", url, resp.StatusCode)
		}
		primary := false
		for _, d := range meta.Data {
			if d.Type == "primary" && d.Location.Href != "" {
				primary = true
			}
		}
		if err != nil || meta.XMLName.Local != "repomd" || !primary {
			return i18n.Errorf("%s: 无效的仓库元数据", url)
		}
	}
	return nil
}

var repoNames = map[string]string{"baseos": "BaseOS", "appstream": "AppStream", "extras": "extras"}

func RewriteRepo(data []byte, base string) ([]byte, bool, error) {
	lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	current := ""
	found := false
	for i, line := range lines {
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(trim, "[") && strings.HasSuffix(trim, "]") {
			current = repoNames[strings.ToLower(strings.TrimSpace(trim[1:len(trim)-1]))]
			continue
		}
		if current == "" {
			continue
		}
		k, _, ok := strings.Cut(trim, "=")
		if !ok {
			continue
		}
		key := strings.TrimSpace(k)
		if key == "mirrorlist" || key == "metalink" {
			lines[i] = "# rockdev disabled: " + trim
		}
		if key == "baseurl" {
			lines[i] = "baseurl=" + strings.TrimRight(base, "/") + "/$releasever/" + current + "/$basearch/os/"
			found = true
		}
	}
	// A stock section can contain only a mirrorlist and commented baseurl.
	for i := len(lines) - 1; i >= 0; i-- {
		trim := strings.TrimSpace(lines[i])
		if !strings.HasPrefix(trim, "[") || !strings.HasSuffix(trim, "]") {
			continue
		}
		name := repoNames[strings.ToLower(strings.TrimSpace(trim[1:len(trim)-1]))]
		if name == "" {
			continue
		}
		has := false
		for j := i + 1; j < len(lines) && !strings.HasPrefix(strings.TrimSpace(lines[j]), "["); j++ {
			if strings.HasPrefix(strings.TrimSpace(lines[j]), "baseurl=") {
				has = true
				break
			}
		}
		if !has {
			entry := "baseurl=" + strings.TrimRight(base, "/") + "/$releasever/" + name + "/$basearch/os/"
			lines = append(lines[:i+1], append([]string{entry}, lines[i+1:]...)...)
			found = true
		}
	}
	if !found {
		return data, false, nil
	}
	out := []byte(strings.Join(lines, "\n"))
	return out, string(out) != string(data), nil
}

type backupEntry struct {
	Name              string
	Mode              os.FileMode
	Original, Updated string
}
type backupManifest struct{ Entries []backupEntry }

func digest(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }

func (m Manager) SwitchMirror(ctx context.Context, base string) (string, error) {
	files, err := filepath.Glob(filepath.Join(m.Paths.RepoDir, "*.repo"))
	if err != nil {
		return "", err
	}
	manifest := backupManifest{}
	managed := false
	originals := map[string][]byte{}
	updates := map[string][]byte{}
	for _, path := range files {
		info, err := os.Lstat(path)
		if err != nil {
			return "", err
		}
		if !info.Mode().IsRegular() {
			return "", i18n.Errorf("拒绝非普通 repo 文件 %s", path)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		for _, line := range strings.Split(string(data), "\n") {
			trim := strings.TrimSpace(line)
			if strings.HasPrefix(trim, "[") && strings.HasSuffix(trim, "]") && repoNames[strings.ToLower(trim[1:len(trim)-1])] != "" {
				managed = true
			}
		}
		out, changed, err := RewriteRepo(data, base)
		if err != nil {
			return "", err
		}
		if !changed {
			continue
		}
		name := filepath.Base(path)
		manifest.Entries = append(manifest.Entries, backupEntry{Name: name, Mode: info.Mode().Perm(), Original: digest(data), Updated: digest(out)})
		originals[name] = data
		updates[name] = out
	}
	if len(manifest.Entries) == 0 {
		if managed {
			return "", nil
		}
		return "", i18n.Errorf("没有找到 BaseOS/AppStream/extras 仓库配置")
	}
	id := "backup-" + time.Now().Format("20060102-150405.000000000")
	dir := filepath.Join(m.Paths.StateDir, "backups", id)
	if err = os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	for name, data := range originals {
		if err = os.WriteFile(filepath.Join(dir, name), data, 0600); err != nil {
			return id, err
		}
	}
	raw, _ := json.MarshalIndent(manifest, "", "  ")
	if err = os.WriteFile(filepath.Join(dir, "manifest.json"), raw, 0600); err != nil {
		return id, err
	}
	rollback := func(cause error) (string, error) {
		var errs []string
		for _, e := range manifest.Entries {
			if err := AtomicWrite(filepath.Join(m.Paths.RepoDir, e.Name), originals[e.Name], e.Mode); err != nil {
				errs = append(errs, err.Error())
			}
		}
		if len(errs) > 0 {
			return id, i18n.Errorf("%w; 配置恢复失败: %s; 备份 %s", cause, strings.Join(errs, "; "), id)
		}
		return id, i18n.Errorf("%w; 原配置已恢复，备份 %s（DNF 缓存需重新刷新）", cause, id)
	}
	for _, e := range manifest.Entries {
		if err = AtomicWrite(filepath.Join(m.Paths.RepoDir, e.Name), updates[e.Name], e.Mode); err != nil {
			return rollback(err)
		}
	}
	if err = m.Runner.Run(ctx, Command{Name: "dnf", Args: []string{"--refresh", "makecache"}}); err != nil {
		return rollback(err)
	}
	return id, nil
}

var backupID = regexp.MustCompile(`^backup-[0-9]{8}-[0-9]{6}\.[0-9]{9}$`)

func (m Manager) Restore(ctx context.Context, id string) error {
	if !backupID.MatchString(id) {
		return i18n.Errorf("无效备份 ID")
	}
	dir := filepath.Join(m.Paths.StateDir, "backups", id)
	raw, err := os.ReadFile(filepath.Join(dir, "manifest.json"))
	if err != nil {
		return err
	}
	var manifest backupManifest
	if err = json.Unmarshal(raw, &manifest); err != nil {
		return err
	}
	if len(manifest.Entries) == 0 {
		return i18n.Errorf("空备份")
	}
	originals := map[string][]byte{}
	for _, e := range manifest.Entries {
		if filepath.Base(e.Name) != e.Name || !strings.HasSuffix(e.Name, ".repo") {
			return i18n.Errorf("备份文件名无效")
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name))
		if err != nil {
			return err
		}
		if digest(data) != e.Original {
			return i18n.Errorf("备份校验失败 %s", e.Name)
		}
		path := filepath.Join(m.Paths.RepoDir, e.Name)
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return i18n.Errorf("拒绝非普通文件 %s", path)
		}
		current, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if digest(current) != e.Updated && digest(current) != e.Original {
			return i18n.Errorf("%s 已被手动修改，停止恢复", e.Name)
		}
		originals[e.Name] = data
	}
	for _, e := range manifest.Entries {
		if err = AtomicWrite(filepath.Join(m.Paths.RepoDir, e.Name), originals[e.Name], e.Mode); err != nil {
			return i18n.Errorf("部分恢复，请保留备份 %s: %w", id, err)
		}
	}
	return m.Runner.Run(ctx, Command{Name: "dnf", Args: []string{"--refresh", "makecache"}})
}
