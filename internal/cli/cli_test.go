package cli

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"rockdev/internal/core"
)

type notifyingReader struct {
	io.Reader
	started chan struct{}
	once    sync.Once
}

func (r *notifyingReader) Read(p []byte) (int, error) {
	r.once.Do(func() { close(r.started) })
	return r.Reader.Read(p)
}
func TestCancelWhileWaitingForInput(t *testing.T) {
	for _, args := range [][]string{nil, {"env", "install", "go"}} {
		reader, writer := io.Pipe()
		started := make(chan struct{})
		a, _ := testApp("")
		a.In = &notifyingReader{Reader: reader, started: started}
		a.Out = io.Discard
		a.Err = io.Discard
		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan error, 1)
		go func() { done <- a.Run(ctx, args) }()
		select {
		case <-started:
		case <-time.After(2 * time.Second):
			t.Fatal("prompt not reached")
		}
		cancel()
		select {
		case err := <-done:
			if !errors.Is(err, context.Canceled) {
				t.Errorf("expected cancellation: %v", err)
			}
		case <-time.After(time.Second):
			t.Error("cancelled prompt blocked")
		}
		writer.Close()
		reader.Close()
	}
}

func testApp(input string) (*App, *bytes.Buffer) {
	out := &bytes.Buffer{}
	return &App{In: strings.NewReader(input), Out: out, Err: out, Detect: func() (core.System, error) {
		return core.System{OS: "linux", ID: "rocky", Major: "9", Arch: "x86_64", Name: "Rocky test"}, nil
	}, CheckRoot: func() bool { return true }, HasDNF: func() bool { return true }}, out
}
func TestHelpAndInvalidCommands(t *testing.T) {
	a, out := testApp("")
	if err := a.Run(context.Background(), []string{"--help"}); err != nil || !strings.Contains(out.String(), "repo create") {
		t.Fatal(err, out.String())
	}
	for _, args := range [][]string{{"bad"}, {"env", "install"}, {"mirror", "use", "evil"}, {"repo", "create", "--dir", "/tmp/x", "--bad"}} {
		if err := a.Run(context.Background(), args); err == nil {
			t.Fatalf("accepted %v", args)
		}
	}
}
func TestInstallDryRunDoesNotNeedRootOrDNF(t *testing.T) {
	a, out := testApp("")
	a.CheckRoot = func() bool { return false }
	a.HasDNF = func() bool { return false }
	if err := a.Run(context.Background(), []string{"env", "install", "python", "--dry-run"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "python3") || !strings.Contains(out.String(), "预览") {
		t.Fatal(out.String())
	}
}
func TestDecliningDoesNotExecute(t *testing.T) {
	a, out := testApp("n\n")
	if err := a.Run(context.Background(), []string{"env", "install", "go"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "取消") {
		t.Fatal(out.String())
	}
}
func TestMenuExitAndEOF(t *testing.T) {
	for _, input := range []string{"0\n", ""} {
		a, _ := testApp(input)
		if err := a.Run(context.Background(), nil); err != nil {
			t.Fatal(err)
		}
	}
}
