package cli

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"rockdev/internal/i18n"
	"strings"
	"time"

	"rockdev/internal/core"
)

const Version = "1.0.0"
const help = `rockdev 0.1.0 — Rocky Linux 环境初始化工具

用法（无参数打开交互菜单）：
  rockdev doctor
  rockdev mirror list
  rockdev mirror check [镜像名]
  rockdev mirror use <official|aliyun|tuna|ustc|auto> [--dry-run] [--yes]
  rockdev mirror restore <备份ID> [--dry-run] [--yes]
  rockdev repo create --dir <新仓库绝对路径> <RPM包名...> [--dry-run] [--yes]
  rockdev env list
  rockdev env install <python|java|go|node|build>... [--dry-run] [--yes]
  rockdev version

--dry-run  仅打印计划，不请求网络、不写文件、不执行安装
--yes      跳过执行确认，供脚本使用
候选目标：Rocky 8/9/10，x86_64/aarch64；实际兼容性等待 VM 验证。
本地仓库为按需 RPM 仓库，不包含 npm/pip/Maven/Go 模块镜像。
`

type App struct {
	Language          string
	Home              string
	Inspect           func(context.Context, []string) ([]core.EnvironmentHealth, error)
	In                io.Reader
	Out, Err          io.Writer
	Detect            func() (core.System, error)
	CheckRoot, HasDNF func() bool
	Manager           core.Manager
	input             *bufio.Reader
}

func (a *App) defaults() {
	if a.In == nil {
		a.In = os.Stdin
	}
	if a.Out == nil {
		a.Out = os.Stdout
	}
	if a.Err == nil {
		a.Err = os.Stderr
	}
	if a.Detect == nil {
		a.Detect = core.Detect
	}
	if a.CheckRoot == nil {
		a.CheckRoot = func() bool { return os.Geteuid() == 0 }
	}
	if a.HasDNF == nil {
		a.HasDNF = func() bool { _, err := exec.LookPath("dnf"); return err == nil }
	}
	if a.Manager.Paths.RepoDir == "" {
		a.Manager.Paths = core.DefaultPaths()
	}
	if a.Manager.Runner == nil {
		a.Manager.Runner = &core.ExecRunner{Out: a.Out}
	}
	if a.input == nil {
		a.input = bufio.NewReader(a.In)
	}
	if a.Inspect == nil {
		a.Inspect = func(ctx context.Context, names []string) ([]core.EnvironmentHealth, error) {
			return (core.HealthInspector{Translate: a.tr}).Inspect(ctx, names)
		}
	}
}

type options struct {
	dry, yes              bool
	update                bool
	userScope             bool
	dir, file, from, venv string
}

func parse(args []string) ([]string, options, error) {
	var pos []string
	var o options
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--dry-run":
			o.dry = true
		case "--yes", "-y":
			o.yes = true
		case "--update":
			o.update = true
		case "--file", "--from", "--venv":
			key := args[i]
			i++
			if i == len(args) {
				return nil, o, i18n.Errorf("Missing value for %s", key)
			}
			switch key {
			case "--file":
				o.file = args[i]
			case "--from":
				o.from = args[i]
			case "--venv":
				o.venv = args[i]
			}
		case "--dir":
			i++
			if i == len(args) {
				return nil, o, i18n.Errorf("--dir 需要目录")
			}
			o.dir = args[i]
		default:
			if strings.HasPrefix(args[i], "-") {
				return nil, o, i18n.Errorf("未知选项 %s", args[i])
			}
			pos = append(pos, args[i])
		}
	}
	return pos, o, nil
}
func (a *App) Run(ctx context.Context, args []string) (resultErr error) {
	a.defaults()
	var languageErr error
	args, languageErr = a.selectLanguage(args)
	if languageErr != nil {
		return languageErr
	}
	defer func() { resultErr = i18n.LocalizeError(a.Language, resultErr) }()
	if len(args) == 0 {
		return a.menu(ctx)
	}
	for _, s := range args {
		if s == "--help" || s == "-h" {
			fmt.Fprint(a.Out, a.helpText())
			return nil
		}
	}
	if len(args) == 1 && (args[0] == "version" || args[0] == "--version") {
		fmt.Fprintln(a.Out, Version)
		return nil
	}
	pos, o, err := parse(args)
	if err != nil {
		return err
	}
	if len(pos) == 0 {
		return i18n.Errorf("缺少命令")
	}
	if handled, err := a.releaseCommand(ctx, pos, o); handled {
		return err
	}
	if o.dir != "" && !(len(pos) >= 2 && pos[0] == "repo" && pos[1] == "create") {
		return i18n.Errorf("--dir 仅适用于 repo create")
	}
	if len(pos) == 1 && pos[0] == "doctor" {
		return a.doctor()
	}
	if len(pos) == 2 && pos[0] == "mirror" && pos[1] == "list" {
		for _, m := range core.Mirrors {
			fmt.Fprintf(a.Out, a.tr("%-10s %s  %s\n"), m.ID, a.tr(m.Name), m.URL)
		}
		return nil
	}
	if len(pos) == 2 && pos[0] == "env" && pos[1] == "list" {
		for _, e := range core.Environments {
			path, err := exec.LookPath(e.Checks[0].Name)
			if err != nil {
				path = "未发现主命令"
			}
			fmt.Fprintf(a.Out, a.tr("%-8s %-24s %s\n"), e.ID, e.Name, a.tr(path))
		}
		return nil
	}
	s, err := a.Detect()
	if err != nil {
		return err
	}
	if err = s.Validate(); err != nil {
		return err
	}
	if len(pos) >= 2 && pos[0] == "mirror" && pos[1] == "check" {
		if len(pos) > 3 {
			return i18n.Errorf("mirror check 最多指定一个镜像")
		}
		if o.dry {
			fmt.Fprintln(a.Out, a.tr("预览：检测镜像的 BaseOS/AppStream/extras 元数据"))
			return nil
		}
		id := ""
		if len(pos) == 3 {
			id = pos[2]
			if _, err = core.FindMirror(id); err != nil {
				return err
			}
		}
		_, err = a.checkMirrors(ctx, s, id)
		return err
	}
	var p core.Plan
	m := a.Manager
	m.System = s
	m.Translate = a.tr
	switch {
	case len(pos) >= 3 && pos[0] == "env" && pos[1] == "install":
		p, err = m.EnvironmentPlan(pos[2:])
	case len(pos) >= 3 && pos[0] == "repo" && pos[1] == "create":
		if o.dir == "" {
			return i18n.Errorf("repo create 必须指定 --dir")
		}
		p, err = m.RepositoryPlan(o.dir, pos[2:])
	case len(pos) == 3 && pos[0] == "mirror" && pos[1] == "use":
		id := pos[2]
		var mirror core.Mirror
		if id != "auto" {
			mirror, err = core.FindMirror(id)
			if err != nil {
				return err
			}
		}
		p = core.Plan{Title: a.tr("切换 Rocky 基础仓库镜像: ") + id, Steps: []core.Step{{Description: a.tr("检查目标镜像的版本、架构和仓库元数据"), Do: func(ctx context.Context) error {
			if id == "auto" {
				var e error
				mirror, e = a.checkMirrors(ctx, s, "")
				return e
			}
			return probe(ctx, mirror, s)
		}}, {Description: a.tr("备份并修改基础仓库配置，刷新 DNF 缓存；失败时恢复配置"), Do: func(ctx context.Context) error {
			backup, e := m.SwitchMirror(ctx, mirror.URL)
			if backup != "" {
				fmt.Fprintln(a.Out, a.tr("配置备份 ID:"), backup)
			}
			return e
		}}}}
	case len(pos) == 3 && pos[0] == "mirror" && pos[1] == "restore":
		id := pos[2]
		p = core.Plan{Title: a.tr("恢复镜像配置备份: ") + id, Steps: []core.Step{{Description: a.tr("校验备份与当前配置、恢复文件并刷新缓存"), Do: func(ctx context.Context) error { return m.Restore(ctx, id) }}}}
	default:
		return i18n.Errorf("无效命令，请使用 --help")
	}
	if err != nil {
		return err
	}
	p.Print(a.Out)
	if o.dry {
		fmt.Fprintln(a.Out, a.tr("预览完成：未执行任何修改。"))
		return p.Execute(ctx, true)
	}
	if !a.CheckRoot() {
		return i18n.Errorf("此操作需要 root，请使用 sudo rockdev ...")
	}
	if !a.HasDNF() {
		return i18n.Errorf("未找到 dnf")
	}
	if !o.yes {
		answer, err := a.prompt(ctx, "执行以上操作？[y/N] ")
		if err != nil && !errors.Is(err, io.EOF) {
			return err
		}
		if strings.ToLower(answer) != "y" && strings.ToLower(answer) != "yes" {
			fmt.Fprintln(a.Out, a.tr("已取消。"))
			return nil
		}
	}
	log, err := core.NewLog(filepath.Join(m.Paths.StateDir, "logs"))
	if err != nil {
		return err
	}
	defer log.Close()
	fmt.Fprintln(a.Out, a.tr("日志:"), log.Name())
	unlock, err := core.Lock(m.Paths.StateDir)
	if err != nil {
		return err
	}
	defer unlock()
	if runner, ok := m.Runner.(*core.ExecRunner); ok {
		original := runner.Out
		runner.Out = io.MultiWriter(a.Out, log)
		defer func() { runner.Out = original }()
	}
	p.Print(log)
	err = p.Execute(ctx, false)
	if err != nil {
		fmt.Fprintln(log, "FAILED:", i18n.LocalizeError(a.Language, err))
		return i18n.Errorf("%w；日志 %s", err, log.Name())
	}
	fmt.Fprintln(log, "SUCCESS")
	fmt.Fprintln(a.Out, a.tr("操作完成。"))
	if pos[0] == "repo" {
		fmt.Fprintln(a.Out, a.tr("仓库配置示例:"), filepath.Join(o.dir, "rockdev-local.repo"), a.tr("（默认禁用，请按 README 验证后启用）"))
	}
	return nil
}

func (a *App) doctor() error {
	s, err := a.Detect()
	if err != nil {
		return err
	}
	fmt.Fprintf(a.Out, a.tr("系统: %s\nID: %s  版本: %s  架构: %s\nroot: %t  dnf: %t\n"), s.Name, s.ID, s.Version, s.Arch, a.CheckRoot(), a.HasDNF())
	if err = s.Validate(); err != nil {
		return err
	}
	fmt.Fprintln(a.Out, a.tr("系统符合候选适配条件；实际部署兼容性等待 VM 验证。网络请运行 mirror check。"))
	return nil
}
func probe(ctx context.Context, m core.Mirror, s core.System) error {
	child, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	return core.Probe(child, &http.Client{Timeout: 8 * time.Second}, m.URL, s)
}
func (a *App) checkMirrors(ctx context.Context, s core.System, id string) (core.Mirror, error) {
	var best core.Mirror
	bestTime := time.Duration(1<<63 - 1)
	for _, m := range core.Mirrors {
		if id != "" && m.ID != id {
			continue
		}
		start := time.Now()
		err := probe(ctx, m, s)
		elapsed := time.Since(start)
		if err != nil {
			fmt.Fprintf(a.Out, a.tr("%s 不可用: %v\n"), m.ID, i18n.LocalizeError(a.Language, err))
		} else {
			fmt.Fprintf(a.Out, a.tr("%s 可用: %s\n"), m.ID, elapsed.Round(time.Millisecond))
			if elapsed < bestTime {
				best = m
				bestTime = elapsed
			}
		}
		if ctx.Err() != nil {
			return best, ctx.Err()
		}
	}
	if best.ID == "" {
		return best, i18n.Errorf("没有可用镜像，请检查 DNS、网络和目标版本")
	}
	return best, nil
}
func (a *App) prompt(ctx context.Context, label string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	fmt.Fprint(a.Out, a.tr(label))
	type inputResult struct {
		line string
		err  error
	}
	result := make(chan inputResult, 1)
	go func() { line, err := a.input.ReadString('\n'); result <- inputResult{line, err} }()
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case value := <-result:
		return strings.TrimSpace(value.line), value.err
	}
}
