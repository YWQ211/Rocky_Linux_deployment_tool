package core

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

type bundleTestRunner struct {
	unsigned bool
	calls    int
}

func (r *bundleTestRunner) Run(context.Context, Command) error { return nil }
func (r *bundleTestRunner) Capture(context.Context, Command) (string, error) {
	r.calls++
	if r.unsigned {
		return "digests OK", nil
	}
	return "RSA signature: OK", nil
}
func bundleTestSystem() System {
	return System{OS: "linux", ID: "rocky", Version: "9.6", Major: "9", Arch: "x86_64"}
}
func bundleTestRepo(t *testing.T) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "repo")
	if e := os.MkdirAll(filepath.Join(p, "repodata"), 0755); e != nil {
		t.Fatal(e)
	}
	for name, data := range map[string]string{"one.rpm": "rpm bytes", "repodata/primary.xml": "metadata", "repodata/repomd.xml": `<repomd><data><location href="repodata/primary.xml"/></data></repomd>`, "old.repo": "unsafe"} {
		if e := os.WriteFile(filepath.Join(p, filepath.FromSlash(name)), []byte(data), 0644); e != nil {
			t.Fatal(e)
		}
	}
	return p
}
func TestBundleRoundtrip(t *testing.T) {
	repo := bundleTestRepo(t)
	archive := filepath.Join(t.TempDir(), "offline.tar.gz")
	r := &bundleTestRunner{}
	m := Manager{System: bundleTestSystem(), Runner: r}
	p, e := m.BundleExportPlan(repo, archive)
	if e != nil {
		t.Fatal(e)
	}
	if e = p.Execute(context.Background(), true); e != nil {
		t.Fatal(e)
	}
	if _, e = os.Stat(archive); !os.IsNotExist(e) {
		t.Fatal("dry run wrote archive")
	}
	if e = p.Execute(context.Background(), false); e != nil {
		t.Fatal(e)
	}
	b, e := InspectBundle(archive)
	if e != nil || len(b.Files) != 3 {
		t.Fatalf("inspect: %+v %v", b, e)
	}
	target := filepath.Join(t.TempDir(), "imported")
	p, e = m.BundleImportPlan(archive, target)
	if e != nil {
		t.Fatal(e)
	}
	if e = p.Execute(context.Background(), false); e != nil {
		t.Fatal(e)
	}
	if r.calls != 1 {
		t.Fatal("signature was not checked")
	}
	config, e := os.ReadFile(filepath.Join(target, "rockdev-local.repo"))
	if e != nil || !strings.Contains(string(config), "gpgcheck=1\nmodule_hotfixes=1") || !strings.Contains(string(config), "enabled=0") {
		t.Fatal(string(config), e)
	}
	if _, e = m.BundleImportPlan(archive, target); e == nil {
		t.Fatal("existing target accepted")
	}
	if _, e = m.BundleExportPlan(repo, archive); e == nil {
		t.Fatal("existing archive accepted")
	}
	r.unsigned = true
	target = filepath.Join(t.TempDir(), "unsigned")
	p, e = m.BundleImportPlan(archive, target)
	if e != nil {
		t.Fatal(e)
	}
	if e = p.Execute(context.Background(), false); e == nil {
		t.Fatal("unsigned accepted")
	}
	if _, e = os.Stat(target); !os.IsNotExist(e) {
		t.Fatal("unsigned target published")
	}
	m.System.Version = "9.5"
	if _, e = m.BundleImportPlan(archive, target); e == nil {
		t.Fatal("minor mismatch accepted")
	}
}
func TestBundleRejectUnsafeArchives(t *testing.T) {
	for _, kind := range []string{"traversal", "windows", "absolute", "link", "duplicate", "checksum", "size", "schema", "system", "oversized", "missing", "location", "corruption"} {
		t.Run(kind, func(t *testing.T) {
			files := map[string][]byte{"one.rpm": []byte("rpm"), "repodata/p.xml": []byte("primary"), "repodata/repomd.xml": []byte(`<repomd><data><location href="repodata/p.xml"/></data></repomd>`)}
			if kind == "location" {
				files["repodata/repomd.xml"] = []byte(`<repomd><data><location href="../bad"/></data></repomd>`)
			}
			b := BundleManifest{Schema: 1, OS: "rocky", Version: "9.6", Major: "9", Arch: "x86_64"}
			for name, data := range files {
				sum := sha256.Sum256(data)
				b.Files = append(b.Files, BundleFile{name, hex.EncodeToString(sum[:]), int64(len(data))})
			}
			switch kind {
			case "schema":
				b.Schema = 2
			case "system":
				b.OS = "ubuntu"
			case "checksum":
				b.Files[0].SHA256 = "bad"
			case "size":
				b.Files[0].Size++
			case "missing":
				delete(files, "one.rpm")
			}
			raw, _ := json.Marshal(b)
			files[bundleManifestName] = raw
			p := filepath.Join(t.TempDir(), "bad.tar.gz")
			f, e := os.Create(p)
			if e != nil {
				t.Fatal(e)
			}
			gz := gzip.NewWriter(f)
			tw := tar.NewWriter(gz)
			for name, data := range files {
				if e = tw.WriteHeader(&tar.Header{Name: name, Mode: 0644, Size: int64(len(data)), Typeflag: tar.TypeReg}); e != nil {
					t.Fatal(e)
				}
				tw.Write(data)
			}
			switch kind {
			case "traversal":
				tw.WriteHeader(&tar.Header{Name: "../evil", Typeflag: tar.TypeReg})
			case "windows":
				tw.WriteHeader(&tar.Header{Name: `C:\evil`, Typeflag: tar.TypeReg})
			case "absolute":
				tw.WriteHeader(&tar.Header{Name: "/evil", Typeflag: tar.TypeReg})
			case "link":
				tw.WriteHeader(&tar.Header{Name: "link.rpm", Linkname: "one.rpm", Typeflag: tar.TypeSymlink})
			case "duplicate":
				tw.WriteHeader(&tar.Header{Name: "one.rpm", Typeflag: tar.TypeReg})
			case "oversized":
				tw.WriteHeader(&tar.Header{Name: "huge.rpm", Typeflag: tar.TypeReg, Size: bundleMaxBytes + 1})
			}
			tw.Close()
			gz.Close()
			f.Close()
			if kind == "corruption" {
				data, _ := os.ReadFile(p)
				data[len(data)-5] ^= 0xff
				os.WriteFile(p, data, 0644)
			}
			if _, e := InspectBundle(p); e == nil {
				t.Fatal("unsafe archive accepted")
			}
		})
	}
}

func TestBundlePublicationRefusesRacedTarget(t *testing.T) {
	parent := t.TempDir()
	stage := filepath.Join(parent, "stage")
	target := filepath.Join(parent, "target")
	if e := os.Mkdir(stage, 0755); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(filepath.Join(stage, "payload"), []byte("new"), 0644); e != nil {
		t.Fatal(e)
	}
	if e := os.Mkdir(target, 0755); e != nil {
		t.Fatal(e)
	}
	if e := bundlePublishDir(stage, target); e == nil {
		t.Fatal("replaced empty destination")
	}
	if _, e := os.Stat(filepath.Join(stage, "payload")); e != nil {
		t.Fatal("source moved after refused publication", e)
	}
}
func TestBundleExportRejectsLinksAndRace(t *testing.T) {
	m := Manager{System: bundleTestSystem()}
	t.Run("hardlink", func(t *testing.T) {
		repo := bundleTestRepo(t)
		if e := os.Link(filepath.Join(repo, "one.rpm"), filepath.Join(repo, "two.rpm")); e != nil {
			t.Skip(e)
		}
		if _, e := m.BundleExportPlan(repo, filepath.Join(t.TempDir(), "out.tar.gz")); e == nil {
			t.Fatal("hardlinks accepted")
		}
	})
	t.Run("symlink", func(t *testing.T) {
		repo := bundleTestRepo(t)
		if e := os.Symlink(t.TempDir(), filepath.Join(repo, "linked")); e != nil {
			t.Skip(e)
		}
		if _, e := m.BundleExportPlan(repo, filepath.Join(t.TempDir(), "out.tar.gz")); e == nil {
			t.Fatal("directory symlink accepted")
		}
	})
	t.Run("raced export", func(t *testing.T) {
		repo := bundleTestRepo(t)
		archive := filepath.Join(t.TempDir(), "out.tar.gz")
		p, e := m.BundleExportPlan(repo, archive)
		if e != nil {
			t.Fatal(e)
		}
		if e = os.WriteFile(archive, []byte("preserved"), 0644); e != nil {
			t.Fatal(e)
		}
		if e = p.Execute(context.Background(), false); e == nil {
			t.Fatal("overwrote raced archive")
		}
		b, _ := os.ReadFile(archive)
		if string(b) != "preserved" {
			t.Fatal("destination changed")
		}
	})
}

func TestBundleRejectsHiddenTrailingArchive(t *testing.T) {
	repo := bundleTestRepo(t)
	archive := filepath.Join(t.TempDir(), "trailing.tar.gz")
	p, e := (Manager{System: bundleTestSystem()}).BundleExportPlan(repo, archive)
	if e != nil {
		t.Fatal(e)
	}
	if e = p.Execute(context.Background(), false); e != nil {
		t.Fatal(e)
	}
	f, e := os.OpenFile(archive, os.O_APPEND|os.O_WRONLY, 0)
	if e != nil {
		t.Fatal(e)
	}
	gz := gzip.NewWriter(f)
	tw := tar.NewWriter(gz)
	tw.WriteHeader(&tar.Header{Name: "hidden.rpm", Mode: 0644, Size: 3, Typeflag: tar.TypeReg})
	tw.Write([]byte("bad"))
	tw.Close()
	gz.Close()
	f.Close()
	if _, e = InspectBundle(archive); e == nil {
		t.Fatal("hidden concatenated tar accepted")
	}
}

// A sparse 32MiB RPM exercises real archive/hash I/O without allocating payload
// bytes in the fixture. Retaining even one full RPM violates this allocation
// budget; streaming readers stay below it regardless of payload size.
func TestBundleLargeRPMStreams(t *testing.T) {
	if testing.Short() {
		t.Skip("large streaming fixture")
	}
	repo := bundleTestRepo(t)
	f, e := os.OpenFile(filepath.Join(repo, "one.rpm"), os.O_WRONLY, 0)
	if e != nil {
		t.Fatal(e)
	}
	if e = f.Truncate(32 << 20); e != nil {
		f.Close()
		t.Fatal(e)
	}
	if e = f.Close(); e != nil {
		t.Fatal(e)
	}
	archive := filepath.Join(t.TempDir(), "large.tar.gz")
	m := Manager{System: bundleTestSystem(), Runner: &bundleTestRunner{}}
	measure := func(label string, fn func() error) {
		t.Helper()
		runtime.GC()
		var before, after runtime.MemStats
		runtime.ReadMemStats(&before)
		if e := fn(); e != nil {
			t.Fatal(e)
		}
		runtime.ReadMemStats(&after)
		if allocation := after.TotalAlloc - before.TotalAlloc; allocation > 16<<20 {
			t.Fatalf("%s allocated %d bytes for a 32MiB RPM; expected streaming", label, allocation)
		}
	}
	measure("export", func() error {
		p, e := m.BundleExportPlan(repo, archive)
		if e != nil {
			return e
		}
		return p.Execute(context.Background(), false)
	})
	measure("inspect", func() error { _, e := InspectBundle(archive); return e })
	target := filepath.Join(t.TempDir(), "imported")
	measure("import", func() error {
		p, e := m.BundleImportPlan(archive, target)
		if e != nil {
			return e
		}
		return p.Execute(context.Background(), false)
	})
	st, e := os.Stat(filepath.Join(target, "one.rpm"))
	if e != nil || st.Size() != 32<<20 {
		t.Fatal("large RPM not preserved", e)
	}
}
