package core

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestSystemGate(t *testing.T) {
	s, err := ParseSystem(strings.NewReader("ID=rocky\nVERSION_ID=\"9.4\"\nPRETTY_NAME=\"Rocky Linux 9.4\"\n"), "linux", "amd64")
	if err != nil || s.Major != "9" || s.Arch != "x86_64" {
		t.Fatalf("%+v %v", s, err)
	}
	if err := s.Validate(); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []System{{ID: "ubuntu", Major: "9", OS: "linux", Arch: "x86_64"}, {ID: "rocky", Major: "99", OS: "linux", Arch: "x86_64"}, {ID: "rocky", Major: "9", OS: "windows", Arch: "x86_64"}, {ID: "rocky", Major: "9", OS: "linux", Arch: "mips"}} {
		if bad.Validate() == nil {
			t.Fatalf("accepted %+v", bad)
		}
	}
}

type fakeRunner struct {
	calls  []Command
	failAt int
}

func (r *fakeRunner) Run(_ context.Context, c Command) error {
	r.calls = append(r.calls, c)
	if r.failAt > 0 && len(r.calls) == r.failAt {
		return errors.New("simulated failure")
	}
	return nil
}

func TestExecutionStopsAndDryRunDoesNothing(t *testing.T) {
	r := &fakeRunner{failAt: 2}
	n := 0
	p := Plan{Title: "test", Steps: []Step{{Description: "one", Do: func(ctx context.Context) error { n++; return r.Run(ctx, Command{Name: "one"}) }}, {Description: "two", Do: func(ctx context.Context) error { return r.Run(ctx, Command{Name: "two"}) }}, {Description: "three", Do: func(context.Context) error { n++; return nil }}}}
	if err := p.Execute(context.Background(), true); err != nil || n != 0 {
		t.Fatal("dry run executed actions", err)
	}
	if err := p.Execute(context.Background(), false); err == nil || n != 1 || len(r.calls) != 2 {
		t.Fatalf("did not stop: %v %d %+v", err, n, r.calls)
	}
}
