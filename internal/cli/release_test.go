package cli

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"rockdev/internal/core"
)

func TestExistingEnvironmentIsKeptUnlessUpdateSelected(t *testing.T) {
	a, out := testApp("")
	a.Inspect = func(context.Context, []string) ([]core.EnvironmentHealth, error) {
		return []core.EnvironmentHealth{{ID: "go", Installed: true, Commands: []core.ToolHealth{{Name: "go", Path: "/usr/local/go/bin/go", Version: "go version old", Healthy: true}}}}, nil
	}
	if err := a.Run(context.Background(), []string{"--lang", "en", "env", "install", "go", "--yes"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "Keeping existing") || strings.Contains(out.String(), "dnf -y") {
		t.Fatal(out.String())
	}
}

func TestProfileVenvSourceIsScopedToVenv(t *testing.T) {
	a, out := testApp("")
	a.Home = t.TempDir()
	target := filepath.Join(a.Home, "venv")
	p := core.DeploymentProfile{Schema: 1, Environments: []string{"python"}, Venv: target, Sources: []core.ProfileSource{{Tool: "pip", Provider: "official"}}}
	data, _ := json.Marshal(p)
	file := filepath.Join(a.Home, "profile.json")
	os.WriteFile(file, data, 0600)
	if err := a.Run(context.Background(), []string{"--lang", "en", "profile", "apply", file, "--dry-run"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), filepath.Join(target, "pip.conf")) {
		t.Fatal(out.String())
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatal("preview created venv")
	}
}

func TestCustomRuntimeUpdateIsRejected(t *testing.T) {
	a, _ := testApp("")
	a.Inspect = func(context.Context, []string) ([]core.EnvironmentHealth, error) {
		return []core.EnvironmentHealth{{ID: "go", Installed: true, Commands: []core.ToolHealth{{Name: "go", Path: "/opt/go/bin/go"}}}}, nil
	}
	if err := a.Run(context.Background(), []string{"--lang", "en", "env", "install", "go", "--update", "--yes"}); err == nil || !strings.Contains(err.Error(), "Cannot update custom runtime") {
		t.Fatal(err)
	}
}
func TestSourceAndProfileDryRunsDoNotWrite(t *testing.T) {
	a, _ := testApp("")
	a.Home = t.TempDir()
	t.Setenv("npm_config_userconfig", "")
	t.Setenv("NPM_CONFIG_USERCONFIG", "")
	if err := a.Run(context.Background(), []string{"source", "use", "npm", "npmmirror", "--dry-run"}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(a.Home, ".npmrc")); !os.IsNotExist(err) {
		t.Fatal("source dry-run wrote config")
	}
	target := filepath.Join(a.Home, "dev.json")
	if err := a.Run(context.Background(), []string{"profile", "export", "--file", target, "--dry-run"}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatal("profile dry-run wrote file")
	}
	if err := a.Run(context.Background(), []string{"source", "check", "npm", "bad", "--dry-run"}); err == nil {
		t.Fatal("invalid provider accepted")
	}
}
func TestProfileRejectsBeforeAnyMutation(t *testing.T) {
	a, _ := testApp("")
	if err := a.Run(context.Background(), []string{"profile", "validate", "missing.json"}); err == nil {
		t.Fatal("missing profile accepted")
	}
}
