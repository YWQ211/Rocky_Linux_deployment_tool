package core

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type healthFake struct {
	calls  []Command
	output string
	fail   bool
}

func (f *healthFake) Run(ctx context.Context, c Command) error {
	f.calls = append(f.calls, c)
	if f.fail {
		return os.ErrPermission
	}
	return nil
}
func (f *healthFake) Capture(ctx context.Context, c Command) (string, error) {
	return f.output, f.Run(ctx, c)
}
func healthTool(t *testing.T, dir, name string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte("test"), 0755); err != nil {
		t.Fatal(err)
	}
	return p
}
func TestHealthValidationBeforeExecution(t *testing.T) {
	f := &healthFake{}
	_, err := (HealthInspector{Runner: f}).Inspect(context.Background(), []string{"go", "bad"})
	if err == nil || len(f.calls) != 0 {
		t.Fatal(err, f.calls)
	}
}
func TestHealthDuplicatesAndMissingChecks(t *testing.T) {
	a, b := t.TempDir(), t.TempDir()
	healthTool(t, a, "java")
	healthTool(t, b, "java")
	f := &healthFake{output: "version"}
	got, err := (HealthInspector{Runner: f, Path: a + string(os.PathListSeparator) + b}).Inspect(context.Background(), []string{"java"})
	if err != nil || !got[0].Installed || len(got[0].Commands) != 3 || got[0].Commands[1].Healthy || len(got[0].Commands[0].Candidates) != 2 || len(got[0].Issues) == 0 {
		t.Fatal(got, err)
	}
}
func TestHealthIsolationRequiresInterpreter(t *testing.T) {
	d := t.TempDir()
	healthTool(t, d, "python3")
	for _, tc := range []struct {
		out      string
		isolated bool
	}{{"False", false}, {"True", true}} {
		f := &healthFake{output: tc.out}
		got, err := (HealthInspector{Runner: f, Path: d, Getenv: func(string) string { return "fake" }}).Inspect(context.Background(), []string{"python"})
		if err != nil || got[0].Isolated != tc.isolated {
			t.Fatal(got, err)
		}
		for _, c := range f.calls {
			if !filepath.IsAbs(c.Name) {
				t.Fatal(c)
			}
		}
	}
}
func TestHealthRejectsRelativePath(t *testing.T) {
	f := &healthFake{}
	got, err := (HealthInspector{Runner: f, Path: "."}).Inspect(context.Background(), []string{"go"})
	if err != nil || got[0].Installed || len(f.calls) != 0 {
		t.Fatal(got, err)
	}
}
func TestVenvPlanSafety(t *testing.T) {
	d := t.TempDir()
	target := filepath.Join(d, "venv")
	f := &healthFake{}
	p, err := VenvPlan("python3", target, f)
	if err != nil {
		t.Fatal(err)
	}
	if err = p.Execute(context.Background(), true); err != nil {
		t.Fatal(err)
	}
	if _, err = os.Lstat(target); !os.IsNotExist(err) {
		t.Fatal(err)
	}
	if err = os.Mkdir(target, 0700); err != nil {
		t.Fatal(err)
	}
	if p.Execute(context.Background(), false) == nil || len(f.calls) != 0 {
		t.Fatal("existing target accepted")
	}
	if _, err = VenvPlan("python3", target, f); err == nil {
		t.Fatal("existing target planned")
	}
}
func TestVenvFailureCleanup(t *testing.T) {
	target := filepath.Join(t.TempDir(), "venv")
	f := &healthFake{fail: true}
	p, err := VenvPlan("python3", target, f)
	if err != nil {
		t.Fatal(err)
	}
	if err = p.Execute(context.Background(), false); err == nil {
		t.Fatal("expected failure")
	}
	if _, err = os.Lstat(target); !os.IsNotExist(err) {
		t.Fatal(err)
	}
	if !strings.Contains(f.calls[0].String(), "-m venv") {
		t.Fatal(f.calls)
	}
}

func TestHealthNonzeroCheckAndArguments(t *testing.T) {
	d := t.TempDir()
	healthTool(t, d, "python3")
	f := &healthFake{fail: true}
	got, err := (HealthInspector{Runner: f, Path: d}).Inspect(context.Background(), []string{"python"})
	if err != nil || !got[0].Installed || got[0].Commands[0].Healthy || len(got[0].Issues) == 0 {
		t.Fatal(got, err)
	}
	if len(f.calls) < 2 || len(f.calls[1].Args) != 3 || f.calls[1].Args[0] != "-m" || f.calls[1].Args[1] != "pip" {
		t.Fatal(f.calls)
	}
}
func TestHealthDeduplicatesAlias(t *testing.T) {
	a, b := t.TempDir(), t.TempDir()
	original := healthTool(t, a, "go")
	if err := os.Symlink(original, filepath.Join(b, "go")); err != nil {
		t.Skip(err)
	}
	paths := a + string(os.PathListSeparator) + b + string(os.PathListSeparator) + a
	got := healthCandidates(paths, "go")
	if len(got) != 1 {
		t.Fatal(got)
	}
}
func TestVenvNewDirectoryChecks(t *testing.T) {
	target := filepath.Join(t.TempDir(), "new")
	f := &healthFake{}
	p, err := VenvPlan("python3", target, f)
	if err != nil {
		t.Fatal(err)
	}
	if err = p.Execute(context.Background(), false); err != nil {
		t.Fatal(err)
	}
	if len(f.calls) != 3 || f.calls[2].Args[0] != "-m" || f.calls[2].Args[1] != "pip" {
		t.Fatal(f.calls)
	}
	if info, err := os.Stat(target); err != nil || !info.IsDir() {
		t.Fatal(info, err)
	}
	if _, err = VenvPlan("python3", "relative", f); err == nil {
		t.Fatal("relative target accepted")
	}
	if _, err = VenvPlan("python3", filepath.Join(target, "absent", "child"), f); err == nil {
		t.Fatal("missing parent accepted")
	}
}
