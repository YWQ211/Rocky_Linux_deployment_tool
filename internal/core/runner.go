package core

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"rockdev/internal/i18n"
	"strings"
	"time"
)

type Command struct {
	Name string
	Args []string
}

func (c Command) String() string { return c.Name + " " + strings.Join(c.Args, " ") }

type Runner interface {
	Run(context.Context, Command) error
}
type ExecRunner struct{ Out io.Writer }

func (r ExecRunner) Capture(ctx context.Context, c Command) (string, error) {
	var b bytes.Buffer
	fmt.Fprintln(r.Out, "$", c.String())
	err := executeCommand(ctx, c, io.MultiWriter(r.Out, &b))
	return b.String(), err
}
func (r ExecRunner) Run(ctx context.Context, c Command) error {
	fmt.Fprintln(r.Out, "$", c.String())
	return executeCommand(ctx, c, r.Out)
}
func executeCommand(ctx context.Context, c Command, out io.Writer) error {
	cmd := exec.CommandContext(ctx, c.Name, c.Args...)
	cmd.Env = append(os.Environ(), "LC_ALL=C", "LANG=C")
	cmd.Stdout = out
	cmd.Stderr = out
	// DNF receives -y; subprocesses must never consume menu/confirmation input.
	if err := cmd.Run(); err != nil {
		return i18n.Errorf("%s: %w", c.Name, err)
	}
	return nil
}

type Step struct {
	Description string
	Do          func(context.Context) error
}
type Plan struct {
	Title string
	Steps []Step
}

func (p Plan) Print(w io.Writer) {
	fmt.Fprintln(w, p.Title)
	for i, s := range p.Steps {
		fmt.Fprintf(w, "  %d. %s\n", i+1, s.Description)
	}
}
func (p Plan) Execute(ctx context.Context, dry bool) error {
	if dry {
		return nil
	}
	for _, s := range p.Steps {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := s.Do(ctx); err != nil {
			return i18n.Errorf("步骤 [%s] 失败: %w", s.Description, err)
		}
	}
	return nil
}

type Paths struct{ RepoDir, StateDir string }

func DefaultPaths() Paths { return Paths{RepoDir: "/etc/yum.repos.d", StateDir: "/var/lib/rockdev"} }

type Manager struct {
	Translate func(string) string
	Paths     Paths
	Runner    Runner
	System    System
}

func (m Manager) CommandStep(c Command) Step {
	return Step{Description: c.String(), Do: func(ctx context.Context) error { return m.Runner.Run(ctx, c) }}
}

// AtomicWrite uses a same-directory temporary file and refuses symbolic links.
func AtomicWrite(path string, data []byte, mode os.FileMode) error {
	if info, err := os.Lstat(path); err == nil {
		if !info.Mode().IsRegular() {
			return i18n.Errorf("拒绝覆盖非普通文件 %s", path)
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".rockdev-*")
	if err != nil {
		return err
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	if err = f.Chmod(mode.Perm()); err == nil {
		_, err = f.Write(data)
	}
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	return os.Rename(tmp, path)
}

func NewLog(dir string) (*os.File, error) {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	return os.OpenFile(filepath.Join(dir, "run-"+time.Now().Format("20060102-150405.000000000")+".log"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
}

func (m Manager) tr(key string) string {
	if m.Translate != nil {
		return m.Translate(key)
	}
	return key
}
