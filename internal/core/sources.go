package core

import (
	"bytes"
	"context"
	"crypto/rand"
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

type SourceManager struct {
	Home, StateDir string
	Getenv         func(string) string
	Translate      func(string) string
}
type SourceProvider struct{ Tool, ID, URL string }
type SourceStatus struct {
	Tool, Path, URL string
	Overrides       []string
}

func SourceProviders(tool string) ([]SourceProvider, error) {
	urls := map[string][]string{
		"pip":   {"official", "https://pypi.org/simple/", "tuna", "https://pypi.tuna.tsinghua.edu.cn/simple/", "aliyun", "https://mirrors.aliyun.com/pypi/simple/"},
		"npm":   {"official", "https://registry.npmjs.org/", "npmmirror", "https://registry.npmmirror.com/"},
		"maven": {"official", "https://repo.maven.apache.org/maven2/", "aliyun", "https://maven.aliyun.com/repository/central"},
		"go":    {"official", "https://proxy.golang.org", "goproxy.cn", "https://goproxy.cn"},
	}
	pairs, ok := urls[tool]
	if !ok {
		return nil, i18n.Errorf("Unknown source tool: %s", tool)
	}
	var out []SourceProvider
	for i := 0; i < len(pairs); i += 2 {
		out = append(out, SourceProvider{tool, pairs[i], pairs[i+1]})
	}
	return out, nil
}
func (s SourceManager) tr(key string) string {
	if s.Translate != nil {
		return s.Translate(key)
	}
	return key
}

// sourceSection keeps section names case-sensitive, as Python ConfigParser does.
// A trailing comment does not form part of the section name.
func sourceSection(line string) (string, bool) {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "[") {
		return "", false
	}
	end := strings.LastIndexByte(line, ']')
	if end <= 1 {
		return "", false
	}
	return line[:end+1], true
}
func sourceOption(tool, line string) []string {
	if tool == "pip" {
		if i := strings.IndexAny(line, "=:"); i >= 0 {
			return []string{strings.ToLower(strings.TrimSpace(line[:i])), line[i+1:]}
		}
		return nil
	}
	return strings.SplitN(line, "=", 2)
}
func (s SourceManager) env(k string) string {
	if s.Getenv != nil {
		return s.Getenv(k)
	}
	return os.Getenv(k)
}
func (s SourceManager) configPath(tool string) (string, error) {
	h := s.Home
	if h == "" {
		var e error
		h, e = os.UserHomeDir()
		if e != nil {
			return "", e
		}
	}
	x := s.env("XDG_CONFIG_HOME")
	if x == "" {
		x = filepath.Join(h, ".config")
	}
	var p string
	switch tool {
	case "pip":
		p = s.env("PIP_CONFIG_FILE")
		if p == "" {
			if v := s.env("VIRTUAL_ENV"); v != "" {
				if info, e := os.Stat(v); e == nil && info.IsDir() {
					p = filepath.Join(v, "pip.conf")
				}
			}
		}
		if p == "" {
			p = filepath.Join(x, "pip", "pip.conf")
		}
	case "npm":
		p = s.env("npm_config_userconfig")
		if p == "" {
			p = s.env("NPM_CONFIG_USERCONFIG")
		}
		if p == "" {
			p = filepath.Join(h, ".npmrc")
		}
	case "maven":
		p = filepath.Join(h, ".m2", "settings.xml")
	case "go":
		p = s.env("GOENV")
		if p == "off" {
			return "", i18n.Errorf("GOENV is disabled")
		}
		if p == "" {
			p = filepath.Join(x, "go", "env")
		}
	default:
		return "", i18n.Errorf("Unknown source tool: %s", tool)
	}
	if !filepath.IsAbs(p) {
		return "", i18n.Errorf("Source configuration path must be absolute")
	}
	p = filepath.Clean(p)
	if e := sourceSafePath(p); e != nil {
		return "", e
	}
	return p, nil
}
func sourceSafePath(p string) error {
	for q := p; ; q = filepath.Dir(q) {
		st, e := os.Lstat(q)
		if e == nil {
			if st.Mode()&os.ModeSymlink != 0 {
				return i18n.Errorf("Refusing symbolic link in source path")
			}
			if q == p && !st.Mode().IsRegular() {
				return i18n.Errorf("Source target is not a regular file")
			}
			if q != p && !st.IsDir() {
				return i18n.Errorf("Source parent is not a directory")
			}
		} else if !os.IsNotExist(e) {
			return e
		}
		if filepath.Dir(q) == q {
			break
		}
	}
	return nil
}
func (s SourceManager) Show(tool string) (SourceStatus, error) {
	p, e := s.configPath(tool)
	st := SourceStatus{Tool: tool, Path: p}
	if e != nil {
		return st, e
	}
	b, e := os.ReadFile(p)
	if e != nil && !os.IsNotExist(e) {
		return st, e
	}
	keys := map[string]string{"pip": "index-url", "npm": "registry", "go": "GOPROXY"}
	if tool == "maven" {
		m, _, e := sourceMaven(b, "")
		if e != nil {
			return st, e
		}
		st.URL = m
	} else {
		section := ""
		for _, line := range strings.Split(string(b), "\n") {
			t := strings.TrimSpace(line)
			if name, ok := sourceSection(t); ok {
				section = name
			}
			if tool == "pip" && section != "[global]" {
				continue
			}
			kv := sourceOption(tool, t)
			if len(kv) == 2 && strings.TrimSpace(kv[0]) == keys[tool] {
				st.URL = sourcePublicURL(strings.TrimSpace(kv[1]))
			}
		}
	}
	for _, key := range map[string][]string{"pip": {"PIP_INDEX_URL", "PIP_EXTRA_INDEX_URL", "PIP_TRUSTED_HOST"}, "npm": {"npm_config_registry", "NPM_CONFIG_REGISTRY"}, "go": {"GOPROXY", "GONOPROXY", "GOPRIVATE"}, "maven": {"MAVEN_ARGS"}}[tool] {
		if s.env(key) != "" {
			st.Overrides = append(st.Overrides, key)
		}
	}
	return st, nil
}
func sourcePublicURL(v string) string {
	if strings.Contains(v, "@") || strings.Contains(v, "?") || strings.Contains(v, "#") {
		return "[configured; redacted]"
	}
	return v
}
func sourceProvider(tool, id string) (SourceProvider, error) {
	ps, e := SourceProviders(tool)
	if e != nil {
		return SourceProvider{}, e
	}
	for _, p := range ps {
		if p.ID == id {
			return p, nil
		}
	}
	return SourceProvider{}, i18n.Errorf("Unknown source provider for %s", tool)
}
func sourceEdit(tool string, b []byte, u string) ([]byte, error) {
	if tool == "maven" {
		_, out, e := sourceMaven(b, u)
		return out, e
	}
	key := map[string]string{"pip": "index-url", "npm": "registry", "go": "GOPROXY"}[tool]
	lines := strings.Split(string(b), "\n")
	section := ""
	found := false
	var out []string
	for _, line := range lines {
		trim := strings.TrimSpace(line)
		if name, ok := sourceSection(trim); ok {
			section = name
		}
		kv := sourceOption(tool, trim)
		if len(kv) == 2 && strings.TrimSpace(kv[0]) == key && (tool != "pip" || section == "[global]") {
			if !found {
				out = append(out, key+"="+u)
				found = true
			}
			continue
		}
		out = append(out, line)
	}
	if !found {
		if tool == "pip" { // Insert inside an existing global section, or create it.
			for i, line := range out {
				if name, ok := sourceSection(line); ok && name == "[global]" {
					out = append(out[:i+1], append([]string{key + "=" + u}, out[i+1:]...)...)
					found = true
					break
				}
			}
			if !found {
				out = append(out, "[global]", key+"="+u)
			}
		} else {
			out = append(out, key+"="+u)
		}
	}
	return []byte(strings.TrimRight(strings.Join(out, "\n"), "\n") + "\n"), nil
}

const sourceMirrorID = "rockdev-central"

func sourceMaven(b []byte, u string) (string, []byte, error) {
	if len(bytes.TrimSpace(b)) == 0 {
		b = []byte("<settings></settings>")
	}
	// Expand empty settings/mirrors elements before using decoder byte offsets.
	if u != "" {
		probe := xml.NewDecoder(bytes.NewReader(b))
		depth := 0
		for {
			token, err := probe.Token()
			if err != nil {
				break
			}
			switch start := token.(type) {
			case xml.StartElement:
				depth++
				end := int(probe.InputOffset())
				managedContainer := (depth == 1 && start.Name.Local == "settings") || (depth == 2 && start.Name.Local == "mirrors")
				if managedContainer && end >= 2 && b[end-2] == '/' {
					nameStart := bytes.LastIndexByte(b[:end], '<') + 1
					nameEnd := nameStart
					for nameEnd < end && !strings.ContainsRune(" \t\r\n/", rune(b[nameEnd])) {
						nameEnd++
					}
					expanded := append([]byte{}, b[:end-2]...)
					expanded = append(expanded, '>')
					expanded = append(expanded, []byte("</"+string(b[nameStart:nameEnd])+">")...)
					expanded = append(expanded, b[end:]...)
					return sourceMaven(expanded, u)
				}
			case xml.EndElement:
				depth--
			}
		}
	}
	d := xml.NewDecoder(bytes.NewReader(b))
	depth := 0
	mirrorStart := -1
	mirrorsEnd := -1
	settingsEnd := -1
	managedStart := -1
	managedEnd := -1
	current := ""
	var item struct {
		ID       string `xml:"id"`
		URL      string `xml:"url"`
		MirrorOf string `xml:"mirrorOf"`
	}
	for {
		before := int(d.InputOffset())
		tok, e := d.Token()
		if e == io.EOF {
			break
		}
		if e != nil {
			return "", nil, i18n.Errorf("Invalid Maven settings XML")
		}
		switch v := tok.(type) {
		case xml.StartElement:
			depth++
			if depth == 1 && v.Name.Local != "settings" {
				return "", nil, i18n.Errorf("Invalid Maven settings root")
			}
			if v.Name.Local == "mirror" && depth == 3 {
				mirrorStart = before
				item = struct {
					ID       string `xml:"id"`
					URL      string `xml:"url"`
					MirrorOf string `xml:"mirrorOf"`
				}{}
				if e = d.DecodeElement(&item, &v); e != nil {
					return "", nil, i18n.Errorf("Invalid Maven mirror")
				}
				depth--
				end := int(d.InputOffset())
				if item.ID == sourceMirrorID {
					if managedStart >= 0 {
						return "", nil, i18n.Errorf("Duplicate managed Maven mirror")
					}
					managedStart = mirrorStart
					managedEnd = end
					current = sourcePublicURL(item.URL)
				} else {
					for _, m := range strings.Split(item.MirrorOf, ",") {
						if strings.TrimSpace(m) == "central" || strings.Contains(m, "*") {
							return "", nil, i18n.Errorf("Existing Maven central mirror requires manual review")
						}
					}
				}
			}
		case xml.EndElement:
			if depth == 2 && v.Name.Local == "mirrors" {
				mirrorsEnd = before
			}
			if depth == 1 && v.Name.Local == "settings" {
				settingsEnd = before
			}
			depth--
		}
	}
	if settingsEnd < 0 {
		return "", nil, i18n.Errorf("Invalid Maven settings XML")
	}
	if u == "" {
		return current, b, nil
	}
	var escaped bytes.Buffer
	xml.EscapeText(&escaped, []byte(u))
	fragment := []byte("<mirror><id>" + sourceMirrorID + "</id><mirrorOf>central</mirrorOf><url>" + escaped.String() + "</url></mirror>")
	if managedStart >= 0 {
		return current, append(append(append([]byte{}, b[:managedStart]...), fragment...), b[managedEnd:]...), nil
	}
	at := mirrorsEnd
	if at < 0 {
		at = settingsEnd
		fragment = append(append([]byte("<mirrors>"), fragment...), []byte("</mirrors>")...)
	}
	return current, append(append(append([]byte{}, b[:at]...), fragment...), b[at:]...), nil
}

type sourceBackup struct {
	Tool, Path, OriginalHash, ManagedHash string
	MetadataHash                          string
	Original                              []byte
	Existed                               bool
	Mode                                  uint32
}

// sourceBackupDigest catches corruption and single-field metadata edits. It is
// not authentication against a user who can rewrite the backup and its digest.
func sourceBackupDigest(b sourceBackup) string {
	b.MetadataHash = ""
	raw, _ := json.Marshal(b)
	return sourceHash(raw)
}
func sourceHash(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }
func (s SourceManager) stateDir() (string, error) {
	p := s.StateDir
	if p == "" {
		h := s.Home
		if h == "" {
			h, _ = os.UserHomeDir()
		}
		p = filepath.Join(h, ".local", "state", "rockdev", "sources")
	}
	if !filepath.IsAbs(p) {
		return "", i18n.Errorf("Source state path must be absolute")
	}
	if e := sourceSafeDirectory(p); e != nil {
		return "", e
	}
	return p, nil
}
func sourceSafeDirectory(p string) error {
	return sourceSafePath(filepath.Join(p, ".source-path-check"))
}
func (s SourceManager) Plan(tool, provider string) (Plan, error) {
	pr, e := sourceProvider(tool, provider)
	if e != nil {
		return Plan{}, e
	}
	p, e := s.configPath(tool)
	if e != nil {
		return Plan{}, e
	}
	dir, e := s.stateDir()
	if e != nil {
		return Plan{}, e
	}
	b, e := os.ReadFile(p)
	exists := e == nil
	if e != nil && !os.IsNotExist(e) {
		return Plan{}, e
	}
	mode := os.FileMode(0600)
	if exists {
		st, e := os.Stat(p)
		if e != nil {
			return Plan{}, e
		}
		mode = st.Mode().Perm()
	}
	out, e := sourceEdit(tool, b, pr.URL)
	if e != nil {
		return Plan{}, e
	}
	random := make([]byte, 16)
	if _, e = rand.Read(random); e != nil {
		return Plan{}, e
	}
	id := hex.EncodeToString(random)
	backup := sourceBackup{Tool: tool, Path: p, OriginalHash: sourceHash(b), ManagedHash: sourceHash(out), Original: b, Existed: exists, Mode: uint32(mode)}
	backup.MetadataHash = sourceBackupDigest(backup)
	return Plan{Title: fmt.Sprintf(s.tr("Configure %s source: %s"), tool, pr.URL), Steps: []Step{{Description: fmt.Sprintf(s.tr("Write %s (backup: %s)"), p, id), Do: func(ctx context.Context) error {
		if e := ctx.Err(); e != nil {
			return e
		}
		if e := sourceSafePath(p); e != nil {
			return e
		}
		now, e := os.ReadFile(p)
		if e != nil && !os.IsNotExist(e) {
			return e
		}
		if (e == nil) != exists || sourceHash(now) != backup.OriginalHash {
			return i18n.Errorf("Source configuration changed after preview")
		}
		if e = sourceSafeDirectory(dir); e != nil {
			return e
		}
		if e = os.MkdirAll(dir, 0700); e != nil {
			return e
		}
		data, _ := json.Marshal(backup)
		if e = AtomicWrite(filepath.Join(dir, id+".json"), data, 0600); e != nil {
			return e
		}
		if e = os.MkdirAll(filepath.Dir(p), 0700); e != nil {
			return e
		}
		if e = sourceSafePath(p); e != nil {
			return e
		}
		return AtomicWrite(p, out, mode)
	}}}}, nil
}
func (s SourceManager) RestorePlan(id string) (Plan, error) {
	if !regexp.MustCompile(`^[a-f0-9]{32}$`).MatchString(id) {
		return Plan{}, i18n.Errorf("Invalid source backup ID")
	}
	dir, e := s.stateDir()
	if e != nil {
		return Plan{}, e
	}
	bp := filepath.Join(dir, id+".json")
	if e = sourceSafePath(bp); e != nil {
		return Plan{}, e
	}
	raw, e := os.ReadFile(bp)
	if e != nil {
		return Plan{}, e
	}
	var b sourceBackup
	if e = json.Unmarshal(raw, &b); e != nil {
		return Plan{}, i18n.Errorf("Invalid source backup")
	}
	p, e := s.configPath(b.Tool)
	if e != nil {
		return Plan{}, e
	}
	if p != b.Path || sourceHash(b.Original) != b.OriginalHash || b.Mode&^0777 != 0 || b.MetadataHash != sourceBackupDigest(b) || (!b.Existed && (len(b.Original) != 0 || b.OriginalHash != sourceHash(nil))) {
		return Plan{}, i18n.Errorf("Invalid source backup integrity")
	}
	check := func() error {
		if e := sourceSafePath(p); e != nil {
			return e
		}
		now, e := os.ReadFile(p)
		if e != nil {
			return e
		}
		if sourceHash(now) != b.ManagedHash {
			return i18n.Errorf("Source configuration was edited; refusing restore")
		}
		r, e := os.ReadFile(bp)
		if e != nil {
			return e
		}
		if !bytes.Equal(r, raw) {
			return i18n.Errorf("Source backup changed after preview")
		}
		return nil
	}
	if e = check(); e != nil {
		return Plan{}, e
	}
	return Plan{Title: fmt.Sprintf(s.tr("Restore source backup %s"), id), Steps: []Step{{Description: fmt.Sprintf(s.tr("Restore %s"), p), Do: func(ctx context.Context) error {
		if e := ctx.Err(); e != nil {
			return e
		}
		if e := check(); e != nil {
			return e
		}
		if !b.Existed {
			return os.Remove(p)
		}
		return AtomicWrite(p, b.Original, os.FileMode(b.Mode))
	}}}}, nil
}
func (s SourceManager) Check(ctx context.Context, client *http.Client, tool, provider string) error {
	p, e := sourceProvider(tool, provider)
	if e != nil {
		return e
	}
	endpoint := strings.TrimRight(p.URL, "/") + map[string]string{"pip": "/pip/", "npm": "/-/ping", "maven": "/org/apache/maven/maven/maven-metadata.xml", "go": "/golang.org/x/text/@v/list"}[tool]
	req, e := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if e != nil {
		return e
	}
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	copyClient := *client
	copyClient.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if req.URL.Scheme != "https" || req.URL.User != nil || len(via) > 10 {
			return i18n.Errorf("Unsafe source redirect")
		}
		return nil
	}
	resp, e := copyClient.Do(req)
	if e != nil {
		return i18n.Errorf("Source connectivity check failed for %s", tool)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return i18n.Errorf("Source metadata returned HTTP %d", resp.StatusCode)
	}
	_, e = io.Copy(io.Discard, io.LimitReader(resp.Body, 1024*1024))
	return e
}
