package core

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type repoRunner struct {
	fakeRunner
	unsigned bool
}

func (r *repoRunner) Run(ctx context.Context, c Command) error {
	if err := r.fakeRunner.Run(ctx, c); err != nil {
		return err
	}
	if c.Name == "dnf" {
		for i, arg := range c.Args {
			if arg == "--downloaddir" {
				return os.WriteFile(filepath.Join(c.Args[i+1], "example.rpm"), []byte("fixture"), 0644)
			}
		}
	}
	if c.Name == "createrepo_c" {
		dir := filepath.Join(c.Args[0], "repodata")
		os.Mkdir(dir, 0755)
		return os.WriteFile(filepath.Join(dir, "repomd.xml"), []byte("fixture metadata"), 0644)
	}
	return nil
}
func (r *repoRunner) Capture(ctx context.Context, c Command) (string, error) {
	if err := r.fakeRunner.Run(ctx, c); err != nil {
		return "", err
	}
	if r.unsigned {
		return "Header SHA256 digest: OK\n", nil
	}
	return "Header V4 RSA/SHA256 Signature, key ID example: OK\n", nil
}

func TestRepoPublishesOnlyAfterSignedPackagesAndMetadata(t *testing.T) {
	for _, unsigned := range []bool{true, false} {
		parent := t.TempDir()
		target := filepath.Join(parent, "repo")
		r := &repoRunner{unsigned: unsigned}
		m := Manager{Runner: r}
		p, err := m.RepositoryPlan(target, []string{"python3"})
		if err != nil {
			t.Fatal(err)
		}
		err = p.Execute(context.Background(), false)
		if unsigned {
			if err == nil {
				t.Fatal("unsigned RPM accepted")
			}
			if _, e := os.Stat(target); !os.IsNotExist(e) {
				t.Fatal("failed repository was published")
			}
		} else {
			if err != nil {
				t.Fatal(err)
			}
			raw, err := os.ReadFile(filepath.Join(target, "rockdev-local.repo"))
			if err != nil || !strings.Contains(string(raw), "enabled=0\ngpgcheck=1\nmodule_hotfixes=1") {
				t.Fatal("invalid client config", err)
			}
			found := false
			for _, c := range r.calls {
				if strings.Contains(strings.Join(c.Args, " "), "--resolve --alldeps") {
					found = true
				}
			}
			if !found {
				t.Fatal("dependencies incomplete")
			}
		}
		leftovers, _ := filepath.Glob(filepath.Join(parent, ".rockdev-repo-*"))
		if len(leftovers) != 0 {
			t.Fatal("staging leaked", leftovers)
		}
	}
}

func TestEnvValidatesAllNamesBeforeCommands(t *testing.T) {
	r := &fakeRunner{}
	m := Manager{Runner: r}
	if _, err := m.EnvironmentPlan([]string{"python", "unknown"}); err == nil || len(r.calls) != 0 {
		t.Fatal("partial install before validation")
	}
	p, err := m.EnvironmentPlan([]string{"python", "go", "node", "java"})
	if err != nil {
		t.Fatal(err)
	}
	if err = p.Execute(context.Background(), false); err != nil {
		t.Fatal(err)
	}
	if r.calls[0].Name != "dnf" || !strings.Contains(strings.Join(r.calls[0].Args, " "), "--setopt=gpgcheck=1") {
		t.Fatalf("missing signature policy %+v", r.calls)
	}
	if !strings.Contains(strings.Join(r.calls[0].Args, " "), "--setopt=*.gpgcheck=1") {
		t.Fatal("repo-specific settings can disable signatures")
	}
}
func TestRepoRejectsUnsafeInputsAndDryRunDoesNotWrite(t *testing.T) {
	m := Manager{Runner: &fakeRunner{}}
	for _, pkgs := range [][]string{{"--nogpgcheck"}, {"https://evil/x.rpm"}, {"a;touch /x"}, {}} {
		if _, err := m.RepositoryPlan(filepath.Join(t.TempDir(), "new"), pkgs); err == nil {
			t.Fatalf("accepted %v", pkgs)
		}
	}
	target := filepath.Join(t.TempDir(), "new")
	p, err := m.RepositoryPlan(target, []string{"python3"})
	if err != nil {
		t.Fatal(err)
	}
	if err = p.Execute(context.Background(), true); err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(target); !os.IsNotExist(err) {
		t.Fatal("dry run wrote target")
	}
}
