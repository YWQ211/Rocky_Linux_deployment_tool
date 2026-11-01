package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"rockdev/internal/core"
	"rockdev/internal/i18n"
)

func (a *App) home() (string, error) {
	if a.Home != "" {
		return a.Home, nil
	}
	return os.UserHomeDir()
}
func (a *App) sourceManager() (core.SourceManager, error) {
	h, e := a.home()
	return core.SourceManager{Home: h, Translate: a.tr}, e
}
func (a *App) systemManager() (core.Manager, error) {
	s, e := a.Detect()
	if e != nil {
		return core.Manager{}, e
	}
	if e = s.Validate(); e != nil {
		return core.Manager{}, e
	}
	m := a.Manager
	m.System = s
	m.Translate = a.tr
	return m, nil
}
func (a *App) releaseCommand(ctx context.Context, pos []string, o options) (bool, error) {
	head := pos[0]
	known := head == "source" || head == "profile" || head == "bundle" || (head == "env" && len(pos) > 1 && (pos[1] == "check" || pos[1] == "install"))
	if !known {
		if o.update || o.venv != "" || o.file != "" || o.from != "" {
			return true, i18n.Errorf("Options do not apply to this command")
		}
		return false, nil
	}
	if head != "env" && head != "profile" && (o.update || o.venv != "") {
		return true, i18n.Errorf("Options do not apply to this command")
	}
	if head == "env" {
		if o.dir != "" || o.file != "" || o.from != "" {
			return true, i18n.Errorf("Options do not apply to this command")
		}
		if pos[1] == "check" {
			if o.update || o.venv != "" {
				return true, i18n.Errorf("Options do not apply to this command")
			}
			if o.dry {
				fmt.Fprintln(a.Out, a.tr("Dry run: environment inspection will execute read-only version checks"))
				return true, nil
			}
			reports, e := a.Inspect(ctx, pos[2:])
			if e == nil {
				a.printHealth(reports)
			}
			return true, e
		}
		if len(pos) < 3 {
			return true, i18n.Errorf("Select at least one environment")
		}
		m, e := a.systemManager()
		if e != nil {
			return true, e
		}
		p, system, e := a.installPlan(ctx, m, pos[2:], o)
		if e != nil {
			return true, e
		}
		return true, a.executeRelease(ctx, p, o, system)
	}
	if head == "source" {
		if o.dir != "" || o.file != "" || o.from != "" {
			return true, i18n.Errorf("Options do not apply to this command")
		}
		sm, e := a.sourceManager()
		if e != nil {
			return true, e
		}
		if len(pos) < 2 {
			return true, i18n.Errorf("Invalid source command")
		}
		switch pos[1] {
		case "list":
			if len(pos) > 3 {
				return true, i18n.Errorf("Invalid source command")
			}
			tools := []string{"pip", "npm", "maven", "go"}
			if len(pos) == 3 {
				tools = pos[2:]
			}
			for _, t := range tools {
				ps, e := core.SourceProviders(t)
				if e != nil {
					return true, e
				}
				for _, p := range ps {
					fmt.Fprintf(a.Out, "%s\t%s\t%s\n", p.Tool, p.ID, p.URL)
				}
			}
			return true, nil
		case "show":
			if len(pos) != 3 {
				return true, i18n.Errorf("Invalid source command")
			}
			s, e := sm.Show(pos[2])
			if e != nil {
				return true, e
			}
			fmt.Fprintf(a.Out, "%s\n%s\n%s\n", s.Tool, s.Path, s.URL)
			if len(s.Overrides) > 0 {
				fmt.Fprintf(a.Out, a.tr("Environment overrides: %s\n"), strings.Join(s.Overrides, ", "))
			}
			return true, nil
		case "check":
			if len(pos) != 4 {
				return true, i18n.Errorf("Invalid source command")
			}
			providers, validationErr := core.SourceProviders(pos[2])
			if validationErr != nil {
				return true, validationErr
			}
			found := false
			for _, v := range providers {
				if v.ID == pos[3] {
					found = true
				}
			}
			if !found {
				return true, i18n.Errorf("Unknown source provider %q for %s", pos[3], pos[2])
			}
			if o.dry {
				fmt.Fprintln(a.Out, a.tr("Dry run: no network request"))
				return true, nil
			}
			e := sm.Check(ctx, &http.Client{Timeout: 12 * time.Second}, pos[2], pos[3])
			if e == nil {
				fmt.Fprintln(a.Out, a.tr("Source endpoint is available"))
			}
			return true, e
		case "use":
			if len(pos) != 4 {
				return true, i18n.Errorf("Invalid source command")
			}
			p, e := sm.Plan(pos[2], pos[3])
			if e != nil {
				return true, e
			}
			status, e := sm.Show(pos[2])
			if e != nil {
				return true, e
			}
			fmt.Fprintf(a.Out, a.tr("Current-user source configuration: %s\n"), status.Path)
			if len(status.Overrides) > 0 {
				fmt.Fprintf(a.Out, a.tr("Environment overrides: %s\n"), strings.Join(status.Overrides, ", "))
			}
			return true, a.executeRelease(ctx, p, o, false)
		case "restore":
			if len(pos) != 3 {
				return true, i18n.Errorf("Invalid source command")
			}
			p, e := sm.RestorePlan(pos[2])
			if e != nil {
				return true, e
			}
			return true, a.executeRelease(ctx, p, o, false)
		}
		return true, i18n.Errorf("Invalid source command")
	}
	if head == "bundle" {
		if len(pos) != 2 || o.file == "" || o.venv != "" || o.update {
			return true, i18n.Errorf("Use bundle export/import with --file and a directory")
		}
		m, e := a.systemManager()
		if e != nil {
			return true, e
		}
		var p core.Plan
		switch pos[1] {
		case "export":
			if o.from == "" || o.dir != "" {
				return true, i18n.Errorf("Bundle export requires --from")
			}
			p, e = m.BundleExportPlan(o.from, o.file)
		case "import":
			if o.dir == "" || o.from != "" {
				return true, i18n.Errorf("Bundle import requires --dir")
			}
			p, e = m.BundleImportPlan(o.file, o.dir)
		default:
			return true, i18n.Errorf("Invalid bundle command")
		}
		if e != nil {
			return true, e
		}
		e = a.executeRelease(ctx, p, o, false)
		if e == nil && !o.dry && pos[1] == "import" {
			if _, statErr := os.Stat(filepath.Join(o.dir, "rockdev-local.repo")); statErr == nil {
				fmt.Fprintf(a.Out, a.tr("Local repository prepared: %s. Enable rockdev-local.repo explicitly for offline installation.\n"), o.dir)
				fmt.Fprintln(a.Out, a.tr("Replace PACKAGE with the package names to install:"))
				fmt.Fprintf(a.Out, "sudo dnf --setopt=reposdir=%s --disablerepo='*' --enablerepo=rockdev-local --setopt=rockdev-local.gpgcheck=1 install PACKAGE\n", shellQuote(o.dir))
			}
		}
		return true, e
	}
	return true, a.profileCommand(ctx, pos, o)
}

func (a *App) printHealth(reports []core.EnvironmentHealth) {
	for _, r := range reports {
		fmt.Fprintf(a.Out, a.tr("Environment %s: installed=%t isolated=%t\n"), r.ID, r.Installed, r.Isolated)
		for _, c := range r.Commands {
			version := strings.ReplaceAll(strings.TrimSpace(c.Version), "\n", "; ")
			if len(version) > 500 {
				version = version[:500] + "..."
			}
			fmt.Fprintf(a.Out, "  %s\t%s\t%s\n", c.Name, c.Path, version)
		}
		for _, issue := range r.Issues {
			fmt.Fprintln(a.Out, "  !", issue)
		}
	}
}
func (a *App) installPlan(ctx context.Context, m core.Manager, names []string, o options) (core.Plan, bool, error) {
	if _, e := m.EnvironmentPlan(names); e != nil {
		return core.Plan{}, false, e
	}
	if o.venv != "" {
		if len(names) != 1 || names[0] != "python" {
			return core.Plan{}, false, i18n.Errorf("--venv accepts only the Python preset")
		}
		python, e := exec.LookPath("python3")
		if e != nil {
			return core.Plan{}, false, i18n.Errorf("Python must be installed before creating a virtual environment")
		}
		p, e := core.VenvPlan(python, o.venv, m.Runner)
		p.Title = fmt.Sprintf(a.tr("Create Python virtual environment: %s"), o.venv)
		return p, false, e
	}
	if o.dry {
		p, e := m.EnvironmentPlan(names)
		fmt.Fprintln(a.Out, a.tr("Existing environments are checked before execution; --yes does not authorize updates"))
		return p, true, e
	}
	reports, e := a.Inspect(ctx, nil)
	if e != nil {
		return core.Plan{}, false, e
	}
	a.printHealth(reports)
	requested := map[string]bool{}
	for _, n := range names {
		requested[n] = true
	}
	var selected, protected []string
	seen := map[string]bool{}
	for _, r := range reports {
		if !requested[r.ID] {
			if r.Installed {
				protected = append(protected, r.ID)
			}
			continue
		}
		seen[r.ID] = true
		if !r.Installed {
			selected = append(selected, r.ID)
			continue
		}
		if r.Isolated {
			fmt.Fprintf(a.Out, a.tr("Keeping isolated environment %s; system packages are unchanged\n"), r.ID)
			protected = append(protected, r.ID)
			continue
		}
		update := o.update
		if !update && !o.yes {
			answer, err := a.prompt(ctx, fmt.Sprintf(a.tr("Update existing environment %s? [y/N] "), r.ID))
			if err != nil && !errors.Is(err, io.EOF) {
				return core.Plan{}, false, err
			}
			update = answer == "y" || answer == "yes" || answer == "Y"
		}
		if update {
			for _, command := range r.Commands {
				if command.Path != "" && !core.IsSystemRuntime(command.Path) {
					return core.Plan{}, false, i18n.Errorf("Cannot update custom runtime %s using DNF; use its original manager or a Python --venv", command.Path)
				}
			}
			selected = append(selected, r.ID)
		} else {
			protected = append(protected, r.ID)
			fmt.Fprintf(a.Out, a.tr("Keeping existing environment %s\n"), r.ID)
		}
	}
	for _, n := range names {
		if !seen[n] {
			selected = append(selected, n)
		}
	}
	if len(selected) == 0 {
		return core.Plan{Title: a.tr("No changes required")}, false, nil
	}
	p, e := m.EnvironmentPlan(selected)
	if e != nil {
		return p, false, e
	}
	var exclusions []string
	for _, id := range protected {
		switch id {
		case "python":
			exclusions = append(exclusions, "python3", "python3-libs", "python3-pip")
			continue
		case "java":
			exclusions = append(exclusions, "java-*-openjdk*", "maven*")
			continue
		case "node":
			exclusions = append(exclusions, "nodejs*", "npm")
			continue
		case "go":
			exclusions = append(exclusions, "golang*")
			continue
		}
		for _, v := range core.Environments {
			if v.ID == id {
				for _, pkg := range v.Packages {
					if strings.HasPrefix(pkg, "java-") && m.System.Major == "10" {
						pkg = "java-21-openjdk-devel"
					}
					exclusions = append(exclusions, pkg)
				}
			}
		}
	}
	if len(exclusions) > 0 {
		p.Steps[0] = m.CommandStep(core.Command{Name: "dnf", Args: append([]string{"-y", "--setopt=gpgcheck=1", "--setopt=*.gpgcheck=1", "--exclude=" + strings.Join(exclusions, ","), "install"}, environmentPackages(selected, m.System.Major)...)})
	}
	return p, true, nil
}
func environmentPackages(names []string, major string) []string {
	var out []string
	for _, n := range names {
		for _, e := range core.Environments {
			if n == e.ID {
				for _, p := range e.Packages {
					if p == "java-17-openjdk-devel" && major == "10" {
						p = "java-21-openjdk-devel"
					}
					out = append(out, p)
				}
			}
		}
	}
	return out
}

func (a *App) executeRelease(ctx context.Context, p core.Plan, o options, privileged bool) error {
	p.Title = a.tr(p.Title)
	for i := range p.Steps {
		p.Steps[i].Description = a.tr(p.Steps[i].Description)
	}
	p.Print(a.Out)
	if o.dry {
		fmt.Fprintln(a.Out, a.tr("预览完成：未执行任何修改。"))
		return nil
	}
	if len(p.Steps) == 0 {
		return nil
	}
	if privileged {
		if !a.CheckRoot() {
			return i18n.Errorf("此操作需要 root，请使用 sudo rockdev ...")
		}
		if !a.HasDNF() {
			return i18n.Errorf("未找到 dnf")
		}
	}
	if !o.yes {
		answer, e := a.prompt(ctx, "执行以上操作？[y/N] ")
		if e != nil && !errors.Is(e, io.EOF) {
			return e
		}
		if strings.ToLower(answer) != "y" && strings.ToLower(answer) != "yes" {
			fmt.Fprintln(a.Out, a.tr("已取消。"))
			return nil
		}
	}
	dir := a.Manager.Paths.StateDir
	if !privileged {
		home, e := a.home()
		if e != nil {
			return e
		}
		dir = filepath.Join(home, ".local", "state", "rockdev")
	}
	unlock, e := core.Lock(dir)
	if e != nil {
		return e
	}
	defer unlock()
	if privileged && o.userScope {
		home, e := a.home()
		if e != nil {
			return e
		}
		releaseUser, e := core.Lock(filepath.Join(home, ".local", "state", "rockdev"))
		if e != nil {
			return e
		}
		defer releaseUser()
	}
	log, e := core.NewLog(filepath.Join(dir, "logs"))
	if e != nil {
		return e
	}
	defer log.Close()
	fmt.Fprintln(a.Out, a.tr("日志:"), log.Name())
	p.Print(log)
	if runner, ok := a.Manager.Runner.(*core.ExecRunner); ok {
		original := runner.Out
		runner.Out = io.MultiWriter(a.Out, log)
		defer func() { runner.Out = original }()
	}
	if e = p.Execute(ctx, false); e != nil {
		fmt.Fprintln(log, "FAILED:", i18n.LocalizeError(a.Language, e))
		return i18n.Errorf("%w；日志 %s", e, log.Name())
	}
	fmt.Fprintln(log, "SUCCESS")
	fmt.Fprintln(a.Out, a.tr("操作完成。"))
	return nil
}

func (a *App) profileCommand(ctx context.Context, pos []string, o options) error {
	if len(pos) < 2 || o.dir != "" || o.from != "" {
		return i18n.Errorf("Invalid profile command")
	}
	if pos[1] == "export" {
		if o.file == "" {
			return i18n.Errorf("Profile export requires --file")
		}
		p := core.DeploymentProfile{Schema: 1, Environments: pos[2:], Update: o.update, Venv: o.venv}
		if len(p.Environments) == 0 {
			p.Environments = []string{"python", "go"}
		}
		data, e := core.EncodeProfile(p)
		if e != nil {
			return e
		}
		target, e := filepath.Abs(o.file)
		if e != nil {
			return e
		}
		plan := core.Plan{Title: a.tr("Export deployment profile"), Steps: []core.Step{{Description: target, Do: func(context.Context) error {
			f, e := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
			if e != nil {
				return e
			}
			_, e = f.Write(data)
			closeErr := f.Close()
			if e != nil {
				return e
			}
			return closeErr
		}}}}
		return a.executeRelease(ctx, plan, o, false)
	}
	if len(pos) != 3 || o.file != "" || o.venv != "" || o.update {
		return i18n.Errorf("Use profile validate/apply <file>; update policy belongs in the profile")
	}
	f, e := os.Open(pos[2])
	if e != nil {
		return e
	}
	p, e := core.DecodeProfile(f)
	f.Close()
	if e != nil {
		return e
	}
	if pos[1] == "validate" {
		fmt.Fprintln(a.Out, a.tr("Profile is valid"))
		return nil
	}
	if pos[1] != "apply" {
		return i18n.Errorf("Invalid profile command")
	}
	m, e := a.systemManager()
	if e != nil {
		return e
	}
	combined := core.Plan{Title: a.tr("Apply deployment profile")}
	system := false
	names := append([]string(nil), p.Environments...)
	if p.Venv != "" {
		var rest []string
		for _, n := range names {
			if n != "python" {
				rest = append(rest, n)
			}
		}
		vp, _, e := a.installPlan(ctx, m, []string{"python"}, options{dry: o.dry, yes: true, venv: p.Venv})
		if e != nil {
			return e
		}
		combined.Steps = append(combined.Steps, vp.Steps...)
		names = rest
	}
	if len(names) > 0 {
		ep, s, e := a.installPlan(ctx, m, names, options{dry: o.dry, yes: true, update: p.Update})
		if e != nil {
			return e
		}
		combined.Steps = append(combined.Steps, ep.Steps...)
		system = s
	}
	sm, e := a.sourceManager()
	if e != nil {
		return e
	}
	for _, s := range p.Sources {
		selectedManager := sm
		if s.Tool == "pip" && p.Venv != "" {
			selectedManager.Getenv = func(key string) string {
				if key == "PIP_CONFIG_FILE" {
					return filepath.Join(p.Venv, "pip.conf")
				}
				return os.Getenv(key)
			}
		}
		status, statusErr := selectedManager.Show(s.Tool)
		if statusErr != nil {
			return statusErr
		}
		if len(status.Overrides) > 0 {
			fmt.Fprintf(a.Out, a.tr("Environment overrides: %s\n"), strings.Join(status.Overrides, ", "))
		}
		sp, e := selectedManager.Plan(s.Tool, s.Provider)
		if e != nil {
			return e
		}
		combined.Steps = append(combined.Steps, sp.Steps...)
	}
	o.userScope = len(p.Sources) > 0
	return a.executeRelease(ctx, combined, o, system)
}

func shellQuote(s string) string { return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'" }
