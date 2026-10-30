package core

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const sampleRepo = "[baseos]\nname=BaseOS\nmirrorlist=https://old.example/list\n#baseurl=https://old.example/$releasever/BaseOS/$basearch/os/\nenabled=1\ngpgcheck=1\ngpgkey=file:///etc/pki/rpm-gpg/key\n\n[thirdparty]\nbaseurl=https://third.example/repo\nenabled=1\n"

func TestRewriteOnlyManagedSections(t *testing.T) {
	out, changed, err := RewriteRepo([]byte(sampleRepo), "https://mirror.example/rocky")
	if err != nil || !changed {
		t.Fatal(err)
	}
	text := string(out)
	for _, want := range []string{"baseurl=https://mirror.example/rocky/$releasever/BaseOS/$basearch/os/", "# rockdev disabled: mirrorlist=", "gpgkey=file:///etc/pki/rpm-gpg/key", "[thirdparty]\nbaseurl=https://third.example/repo"} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q in %s", want, text)
		}
	}
	if _, changed, err = RewriteRepo(out, "https://mirror.example/rocky"); err != nil || changed {
		t.Fatalf("not idempotent %v %v", changed, err)
	}
}
func TestProbeRejectsHTMLAndWrongXML(t *testing.T) {
	for _, body := range []string{"<html>bad</html>", "<repomd></repomd>", "garbage"} {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(body)) }))
		if err := Probe(context.Background(), srv.Client(), srv.URL, System{Major: "9", Arch: "x86_64"}); err == nil {
			t.Errorf("accepted %s", body)
		}
		srv.Close()
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<repomd xmlns="http://linux.duke.edu/metadata/repo"><data type="primary"><location href="repodata/primary.xml.gz"/></data></repomd>`))
	}))
	defer srv.Close()
	if err := Probe(context.Background(), srv.Client(), srv.URL, System{Major: "9", Arch: "x86_64"}); err != nil {
		t.Fatal(err)
	}
}
func TestMirrorFailureRestoresOriginal(t *testing.T) {
	dir := t.TempDir()
	repos := filepath.Join(dir, "repos")
	os.Mkdir(repos, 0755)
	path := filepath.Join(repos, "rocky.repo")
	os.WriteFile(path, []byte(sampleRepo), 0644)
	r := &fakeRunner{failAt: 1}
	m := Manager{Paths: Paths{RepoDir: repos, StateDir: filepath.Join(dir, "state")}, Runner: r}
	if _, err := m.SwitchMirror(context.Background(), "https://mirror.example/rocky"); err == nil {
		t.Fatal("expected refresh failure")
	}
	got, _ := os.ReadFile(path)
	if string(got) != sampleRepo {
		t.Fatal("original not restored", string(got))
	}
}
func TestBackupRestoreDetectsEdits(t *testing.T) {
	dir := t.TempDir()
	repos := filepath.Join(dir, "repos")
	os.Mkdir(repos, 0755)
	path := filepath.Join(repos, "rocky.repo")
	os.WriteFile(path, []byte(sampleRepo), 0644)
	m := Manager{Paths: Paths{RepoDir: repos, StateDir: filepath.Join(dir, "state")}, Runner: &fakeRunner{}}
	id, err := m.SwitchMirror(context.Background(), "https://mirror.example/rocky")
	if err != nil {
		t.Fatal(err)
	}
	os.WriteFile(path, []byte("user edited"), 0644)
	if err = m.Restore(context.Background(), id); err == nil {
		t.Fatal("overwrote manual edits")
	}
	if err = m.Restore(context.Background(), "../escape"); err == nil {
		t.Fatal("accepted traversal")
	}
}

func TestMirrorBackupRoundTrip(t *testing.T) {
	dir := t.TempDir()
	repos := filepath.Join(dir, "repos")
	os.Mkdir(repos, 0755)
	path := filepath.Join(repos, "rocky.repo")
	os.WriteFile(path, []byte(sampleRepo), 0644)
	m := Manager{Paths: Paths{RepoDir: repos, StateDir: filepath.Join(dir, "state")}, Runner: &fakeRunner{}}
	id, err := m.SwitchMirror(context.Background(), "https://mirror.example/rocky")
	if err != nil {
		t.Fatal(err)
	}
	if err = m.Restore(context.Background(), id); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	if string(data) != sampleRepo {
		t.Fatal("round trip corrupted original")
	}
}

func TestSwitchAlreadySelectedMirrorIsNoop(t *testing.T) {
	dir := t.TempDir()
	repos := filepath.Join(dir, "repos")
	os.Mkdir(repos, 0755)
	data, _, _ := RewriteRepo([]byte(sampleRepo), "https://mirror.example/rocky")
	os.WriteFile(filepath.Join(repos, "rocky.repo"), data, 0644)
	m := Manager{Paths: Paths{RepoDir: repos, StateDir: filepath.Join(dir, "state")}, Runner: &fakeRunner{}}
	if _, err := m.SwitchMirror(context.Background(), "https://mirror.example/rocky"); err != nil {
		t.Fatal(err)
	}
}
