package core

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var packageName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9+_.-]*$`)

func (m Manager) RepositoryPlan(target string, packages []string) (Plan, error) {
	if !filepath.IsAbs(target) || strings.ContainsAny(target, "\r\n\x00") || strings.Contains(target, "$") {
		return Plan{}, fmt.Errorf("仓库目录必须是绝对路径，不能包含换行或变量")
	}
	target = filepath.Clean(target)
	if _, err := os.Lstat(target); err == nil {
		return Plan{}, fmt.Errorf("目标已存在，使用新目录以保护已有仓库: %s", target)
	} else if !os.IsNotExist(err) {
		return Plan{}, err
	}
	if len(packages) == 0 {
		return Plan{}, fmt.Errorf("至少指定一个 RPM 包名")
	}
	for _, name := range packages {
		if !packageName.MatchString(name) {
			return Plan{}, fmt.Errorf("无效 RPM 包名 %q（不接受 URL、选项和通配符）", name)
		}
	}
	parent := filepath.Dir(target)
	if info, err := os.Stat(parent); err != nil || !info.IsDir() {
		return Plan{}, fmt.Errorf("父目录不存在: %s", parent)
	}
	p := Plan{Title: "创建按需 RPM 仓库: " + target}
	p.Steps = append(p.Steps, m.CommandStep(Command{"dnf", []string{"-y", "--setopt=gpgcheck=1", "--setopt=*.gpgcheck=1", "install", "dnf-plugins-core", "createrepo_c"}}))
	description := "下载 " + strings.Join(packages, " ") + " 及所有依赖（--resolve --alldeps）；校验签名；生成元数据；原子发布到 " + target
	p.Steps = append(p.Steps, Step{Description: description, Do: func(ctx context.Context) error {
		staging, err := os.MkdirTemp(parent, ".rockdev-repo-*")
		if err != nil {
			return err
		}
		// MkdirTemp guarantees a new direct child of the specified parent.
		defer os.RemoveAll(staging)
		if err = os.Chmod(staging, 0755); err != nil {
			return err
		}
		args := append([]string{"--setopt=gpgcheck=1", "--setopt=*.gpgcheck=1", "download", "--resolve", "--alldeps", "--downloaddir", staging}, packages...)
		if err = m.Runner.Run(ctx, Command{"dnf", args}); err != nil {
			return err
		}
		rpms, err := filepath.Glob(filepath.Join(staging, "*.rpm"))
		if err != nil {
			return err
		}
		if len(rpms) == 0 {
			return fmt.Errorf("没有下载到 RPM 包")
		}
		for _, rpm := range rpms {
			if err = VerifyRPM(ctx, m.Runner, rpm); err != nil {
				return err
			}
		}
		if err = m.Runner.Run(ctx, Command{"createrepo_c", []string{staging}}); err != nil {
			return err
		}
		if _, err = os.Stat(filepath.Join(staging, "repodata", "repomd.xml")); err != nil {
			return fmt.Errorf("仓库元数据未生成: %w", err)
		}
		localURL := (&url.URL{Scheme: "file", Path: filepath.ToSlash(target)}).String()
		config := "[rockdev-local]\nname=Rockdev local RPM repository\nbaseurl=" + localURL + "\nenabled=0\ngpgcheck=1\nmodule_hotfixes=1\n# Selected RPM set, not a complete module mirror.\n# Uses RPM keys already trusted by this machine. Import official Rocky keys on clients.\n"
		if err = os.WriteFile(filepath.Join(staging, "rockdev-local.repo"), []byte(config), 0644); err != nil {
			return err
		}
		if _, err = os.Lstat(target); !os.IsNotExist(err) {
			return fmt.Errorf("目标目录在执行期间出现，停止发布: %s", target)
		}
		return os.Rename(staging, target)
	}})
	return p, nil
}

type captureRunner interface {
	Capture(context.Context, Command) (string, error)
}

func VerifyRPM(ctx context.Context, r Runner, path string) error {
	capture, ok := r.(captureRunner)
	if !ok {
		return fmt.Errorf("运行器不支持签名验证")
	}
	out, err := capture.Capture(ctx, Command{"rpmkeys", []string{"--checksig", "--verbose", path}})
	if err != nil {
		return err
	}
	lower := strings.ToLower(out)
	if strings.Contains(lower, "not ok") || strings.Contains(lower, "nokey") || strings.Contains(lower, "nottrusted") {
		return fmt.Errorf("RPM 签名不可信: %s", path)
	}
	for _, line := range strings.Split(lower, "\n") {
		if strings.Contains(line, "signature") && strings.HasSuffix(strings.TrimSpace(line), ": ok") {
			return nil
		}
	}
	return fmt.Errorf("RPM 缺少有效签名: %s", path)
}
