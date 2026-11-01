package core

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"testing"
)

func TestCaptureHelper(t *testing.T) {
	if os.Args[len(os.Args)-1] == "rockdev-capture-helper" {
		fmt.Fprintln(os.Stdout, "True")
		os.Exit(0)
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
