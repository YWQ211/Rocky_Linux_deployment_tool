package core

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"rockdev/internal/i18n"
)

type HealthInspector struct {
	Translate func(string) string
	Runner    Runner
	Path      string
	Getenv    func(string) string
}
type EnvironmentHealth struct {
	ID        string
	Commands  []ToolHealth
	Installed bool
	Isolated  bool
	Issues    []string
}
type ToolHealth struct {
	Name, Path, Version string
	Candidates          []string
	Healthy             bool
}

func (h HealthInspector) Inspect(ctx context.Context, names []string) ([]EnvironmentHealth, error) {
	var selected []Environment
	if len(names) == 0 {
		selected = append(selected, Environments...)
	} else {
		seen := map[string]bool{}
		for _, name := range names {
			found := false
			for _, e := range Environments {
				if name == e.ID {
					found = true
					if !seen[name] {
						selected = append(selected, e)
						seen[name] = true
					}
					break
				}
			}
			if !found {
				return nil, i18n.Errorf("Unknown development environment %q", name)
			}
		}
	}
	runner := h.Runner
	if runner == nil {
		runner = ExecRunner{Out: io.Discard}
	}
	getenv := h.Getenv
	if getenv == nil {
		getenv = os.Getenv
	}
	path := h.Path
	if path == "" {
		path = getenv("PATH")
	}
	var result []EnvironmentHealth
	for _, e := range selected {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		item := EnvironmentHealth{ID: e.ID}
		for _, check := range e.Checks {
			tool := ToolHealth{Name: check.Name, Candidates: healthCandidates(path, check.Name)}
			if len(tool.Candidates) == 0 {
				item.Issues = append(item.Issues, h.issue("Command %s is missing", check.Name))
				item.Commands = append(item.Commands, tool)
				continue
			}
			item.Installed = true
			tool.Path = tool.Candidates[0]
			if len(tool.Candidates) > 1 {
				item.Issues = append(item.Issues, h.issue("Multiple PATH candidates for %s", check.Name))
			}
			if !managedRuntime(tool.Path) {
				item.Issues = append(item.Issues, h.issue("Unmanaged runtime path: %s", tool.Path))
			}
			c := Command{Name: tool.Path, Args: append([]string(nil), check.Args...)}
			output, err := healthRun(ctx, runner, c)
			tool.Version = strings.TrimSpace(output)
			tool.Healthy = err == nil
			if err != nil {
				item.Issues = append(item.Issues, h.issue("Command check failed for %s: %v", check.Name, err))
			}
			item.Commands = append(item.Commands, tool)
		}
		if e.ID == "python" && len(item.Commands) > 0 && item.Commands[0].Path != "" {
			output, err := healthRun(ctx, runner, Command{item.Commands[0].Path, []string{"-c", "import os, sys; print(sys.prefix != sys.base_prefix or (bool(os.environ.get('CONDA_PREFIX')) and os.path.realpath(sys.prefix) == os.path.realpath(os.environ['CONDA_PREFIX'])))"}})
			item.Isolated = err == nil && strings.TrimSpace(output) == "True"
			if err != nil {
				item.Issues = append(item.Issues, h.issue("Python isolation check failed: %v", err))
			}
		}
		if e.ID == "java" && getenv("JAVA_HOME") != "" && len(item.Commands) > 0 && item.Commands[0].Path != "" {
			expected := filepath.Join(getenv("JAVA_HOME"), "bin", "java")
			if runtime.GOOS == "windows" {
				expected += ".exe"
			}
			if canonicalHealthPath(expected) != canonicalHealthPath(item.Commands[0].Path) {
				item.Issues = append(item.Issues, h.issue("JAVA_HOME does not match PATH java"))
			}
		}
		result = append(result, item)
	}
	return result, nil
}
func healthRun(ctx context.Context, r Runner, c Command) (string, error) {
	bounded, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	if capture, ok := r.(captureRunner); ok {
		return capture.Capture(bounded, c)
	}
	return "", r.Run(bounded, c)
}
func canonicalHealthPath(path string) string {
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		path = resolved
	}
	path = filepath.Clean(path)
	if runtime.GOOS == "windows" {
		path = strings.ToLower(path)
	}
	return path
}
func healthCandidates(path, name string) []string {
	var result []string
	seen := map[string]bool{}
	for _, dir := range filepath.SplitList(path) {
		if !filepath.IsAbs(dir) {
			continue
		}
		names := []string{name}
		if runtime.GOOS == "windows" {
			names = append(names, name+".exe", name+".cmd", name+".bat")
		}
		for _, n := range names {
			p := filepath.Join(dir, n)
			info, err := os.Stat(p)
			if err != nil || !info.Mode().IsRegular() || (runtime.GOOS != "windows" && info.Mode().Perm()&0111 == 0) {
				continue
			}
			key := canonicalHealthPath(p)
			if !seen[key] {
				seen[key] = true
				result = append(result, p)
			}
		}
	}
	return result
}
func managedRuntime(path string) bool {
	return IsSystemRuntime(path)
}

// System entrypoints may resolve through alternatives, but not into custom homes.
func IsSystemRuntime(path string) bool {
	resolved, e := filepath.EvalSymlinks(path)
	if e != nil {
		return false
	}
	return systemRuntimeTarget(path, resolved)
}

func systemRuntimeTarget(path, resolved string) bool {
	p := filepath.ToSlash(filepath.Clean(path))
	if !strings.HasPrefix(p, "/usr/bin/") && !strings.HasPrefix(p, "/bin/") {
		return false
	}
	target := filepath.ToSlash(filepath.Clean(resolved))
	// Rocky 8's packaged Python resolves through alternatives into libexec.
	// Only these Python entrypoints may use this exact target; other libexec
	// programs and custom Python locations still require their original manager.
	if target == "/usr/libexec/platform-python3.6" {
		return p == "/usr/bin/python3" || p == "/bin/python3" || p == "/usr/bin/python3.6" || p == "/bin/python3.6"
	}
	return strings.HasPrefix(target, "/usr/bin/") || strings.HasPrefix(target, "/bin/") || strings.HasPrefix(target, "/usr/lib/") || strings.HasPrefix(target, "/usr/lib64/")
}

// VenvPlan creates only a new directory; it never invokes a package manager.
func VenvPlan(pythonPath, target string, runner Runner) (Plan, error) {
	if strings.TrimSpace(pythonPath) == "" {
		return Plan{}, i18n.Errorf("Python command is required")
	}
	if !filepath.IsAbs(target) || filepath.Clean(target) == filepath.VolumeName(target)+string(os.PathSeparator) {
		return Plan{}, i18n.Errorf("Virtual environment target must be an absolute new directory")
	}
	target = filepath.Clean(target)
	parent := filepath.Dir(target)
	info, err := os.Stat(parent)
	if err != nil || !info.IsDir() {
		return Plan{}, i18n.Errorf("Virtual environment parent must be an existing directory")
	}
	resolved, err := filepath.EvalSymlinks(parent)
	if err != nil {
		return Plan{}, err
	}
	target = filepath.Join(resolved, filepath.Base(target))
	if _, err = os.Lstat(target); err == nil {
		return Plan{}, i18n.Errorf("Virtual environment target already exists: %s", target)
	} else if !os.IsNotExist(err) {
		return Plan{}, err
	}
	if runner == nil {
		runner = ExecRunner{Out: io.Discard}
	}
	python := filepath.Join(target, "bin", "python")
	if runtime.GOOS == "windows" {
		python = filepath.Join(target, "Scripts", "python.exe")
	}
	commands := []Command{{pythonPath, []string{"-m", "venv", target}}, {python, []string{"-c", "import sys; assert sys.prefix != sys.base_prefix; print(sys.version)"}}, {python, []string{"-m", "pip", "--version"}}}
	plan := Plan{Title: fmt.Sprintf("Create Python virtual environment: %s", target)}
	// All commands belong to one step so failures can safely remove our own directory.
	plan.Steps = []Step{{Description: commands[0].String() + "; " + commands[1].String() + "; " + commands[2].String(), Do: func(ctx context.Context) (err error) {
		if canonicalHealthPath(parent) != canonicalHealthPath(resolved) {
			return i18n.Errorf("Virtual environment parent changed")
		}
		if err = os.Mkdir(target, 0700); err != nil {
			return err
		}
		created, statErr := os.Lstat(target)
		if statErr != nil {
			return statErr
		}
		defer func() {
			if err != nil {
				current, e := os.Lstat(target)
				if e == nil && current.IsDir() && current.Mode()&os.ModeSymlink == 0 && os.SameFile(created, current) {
					_ = os.RemoveAll(target)
				}
			}
		}()
		for _, c := range commands {
			if err = ctx.Err(); err != nil {
				return err
			}
			if err = runner.Run(ctx, c); err != nil {
				return err
			}
		}
		return nil
	}}}
	return plan, nil
}

func (h HealthInspector) issue(format string, args ...any) string {
	if h.Translate != nil {
		format = h.Translate(format)
	}
	return fmt.Sprintf(format, args...)
}
