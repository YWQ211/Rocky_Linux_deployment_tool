package core

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestCaptureHelper(t *testing.T) {
	if os.Args[len(os.Args)-1] == "rockdev-toolchain-env" {
		fmt.Fprintln(os.Stdout, os.Getenv("GOTOOLCHAIN"))
		os.Exit(0)
	}
	if os.Args[len(os.Args)-1] == "rockdev-capture-helper" {
		fmt.Fprintln(os.Stdout, "True")
		os.Exit(0)
	}
}

func TestGoInspectionCannotAutoDownloadToolchains(t *testing.T) {
	t.Setenv("GOTOOLCHAIN", "auto")
	exe, e := os.Executable()
	if e != nil {
		t.Fatal(e)
	}
	data, e := os.ReadFile(exe)
	if e != nil {
		t.Fatal(e)
	}
	name := "go"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	copy := filepath.Join(t.TempDir(), name)
	if e = os.WriteFile(copy, data, 0755); e != nil {
		t.Fatal(e)
	}
	out, e := (ExecRunner{Out: io.Discard}).Capture(context.Background(), Command{copy, []string{"-test.run=^TestCaptureHelper$", "--", "rockdev-toolchain-env"}})
	if e != nil || out != "local\n" {
		t.Fatalf("inspection toolchain %q: %v", out, e)
	}
	if os.Getenv("GOTOOLCHAIN") != "auto" {
		t.Fatal("changed caller environment")
	}
}
func TestCaptureExcludesCommandEcho(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	var log bytes.Buffer
	out, err := (ExecRunner{Out: &log}).Capture(context.Background(), Command{exe, []string{"-test.run=^TestCaptureHelper$", "--", "rockdev-capture-helper"}})
	if err != nil || out != "True\n" {
		t.Fatalf("captured %q: %v", out, err)
	}
	if !bytes.Contains(log.Bytes(), []byte("$ ")) {
		t.Fatal("missing command log")
	}
}
