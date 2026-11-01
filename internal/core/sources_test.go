package core

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSourceMavenShowRedactsURLCredentials(t *testing.T) {
	home := t.TempDir()
	os.MkdirAll(filepath.Join(home, ".m2"), 0700)
	data := `<settings><mirrors><mirror><id>` + sourceMirrorID + `</id><mirrorOf>central</mirrorOf><url>https://name:secret@example.org/</url></mirror></mirrors></settings>`
	os.WriteFile(filepath.Join(home, ".m2", "settings.xml"), []byte(data), 0600)
	s, e := (SourceManager{Home: home, Getenv: func(string) string { return "" }}).Show("maven")
	if e != nil {
		t.Fatal(e)
	}
	if strings.Contains(s.URL, "secret") || strings.Contains(s.URL, "name:") {
		t.Fatal("credentials exposed", s.URL)
	}
}

func TestSourcePreserveAndRestore(t *testing.T) {
	for _, tool := range []string{"pip", "npm", "maven", "go"} {
		t.Run(tool, func(t *testing.T) {
			h := t.TempDir()
			s := SourceManager{Home: h, StateDir: filepath.Join(h, "state"), Getenv: func(string) string { return "" }}
			st, e := s.Show(tool)
			if e != nil {
				t.Fatal(e)
			}
			original := map[string]string{"pip": "[global]\ntimeout = 42\n", "npm": "//private.example/:_authToken=secret\n", "maven": "<settings><servers><server><id>private</id></server></servers></settings>", "go": "GOSUMDB=sum.golang.org\n"}[tool]
			os.MkdirAll(filepath.Dir(st.Path), 0700)
			os.WriteFile(st.Path, []byte(original), 0600)
			p, e := s.Plan(tool, "official")
			if e != nil {
				t.Fatal(e)
			}
			if e = p.Execute(context.Background(), true); e != nil {
				t.Fatal(e)
			}
			b, _ := os.ReadFile(st.Path)
			if string(b) != original {
				t.Fatal("dry run wrote")
			}
			if e = p.Execute(context.Background(), false); e != nil {
				t.Fatal(e)
			}
			b, _ = os.ReadFile(st.Path)
			if !strings.Contains(string(b), strings.TrimSpace(original)) && tool != "maven" && tool != "pip" {
				t.Fatalf("lost config %s", b)
			}
			entries, _ := os.ReadDir(s.StateDir)
			var id string
			for _, ent := range entries {
				if strings.HasSuffix(ent.Name(), ".json") {
					id = strings.TrimSuffix(ent.Name(), ".json")
				}
			}
			r, e := s.RestorePlan(id)
			if e != nil {
				t.Fatal(e)
			}
			if e = r.Execute(context.Background(), false); e != nil {
				t.Fatal(e)
			}
			b, _ = os.ReadFile(st.Path)
			if string(b) != original {
				t.Fatalf("restore mismatch %s", b)
			}
		})
	}
}
func TestSourceRejectUnsafeAndManualEdits(t *testing.T) {
	h := t.TempDir()
	s := SourceManager{Home: h, StateDir: filepath.Join(h, "state"), Getenv: func(string) string { return "" }}
	if _, e := s.Plan("npm", "https://evil.example"); e == nil {
		t.Fatal("accepted custom provider")
	}
	p, e := s.Plan("npm", "official")
	if e != nil {
		t.Fatal(e)
	}
	if e = p.Execute(context.Background(), false); e != nil {
		t.Fatal(e)
	}
	entries, _ := os.ReadDir(s.StateDir)
	id := strings.TrimSuffix(entries[0].Name(), ".json")
	st, _ := s.Show("npm")
	os.WriteFile(st.Path, []byte("manual"), 0600)
	if _, e = s.RestorePlan(id); e == nil {
		t.Fatal("accepted manual edit")
	}
	if _, e = s.RestorePlan("../x"); e == nil {
		t.Fatal("accepted traversal")
	}
}

func TestSourceAbsentRestoreAndIdempotency(t *testing.T) {
	for _, tool := range []string{"pip", "npm", "maven", "go"} {
		t.Run(tool, func(t *testing.T) {
			h := t.TempDir()
			s := SourceManager{Home: h, StateDir: filepath.Join(h, "state"), Getenv: func(string) string { return "" }}
			p, e := s.Plan(tool, "official")
			if e != nil {
				t.Fatal(e)
			}
			if e = p.Execute(context.Background(), false); e != nil {
				t.Fatal(e)
			}
			st, _ := s.Show(tool)
			first, _ := os.ReadFile(st.Path)
			out, e := sourceEdit(tool, first, st.URL)
			if e != nil || string(first) != string(out) {
				t.Fatalf("not idempotent %q %v", out, e)
			}
			entries, _ := os.ReadDir(s.StateDir)
			r, e := s.RestorePlan(strings.TrimSuffix(entries[0].Name(), ".json"))
			if e != nil {
				t.Fatal(e)
			}
			if e = r.Execute(context.Background(), false); e != nil {
				t.Fatal(e)
			}
			if _, e = os.Stat(st.Path); !os.IsNotExist(e) {
				t.Fatal("restore did not remove new file")
			}
		})
	}
}
func TestSourceMavenAmbiguityAndPreservation(t *testing.T) {
	original := []byte(`<?xml version="1.0"?><settings><!--keep--><mirrors><mirror><id>private</id><mirrorOf>internal</mirrorOf><url>https://private/</url></mirror></mirrors><servers><server><password>secret</password></server></servers></settings>`)
	_, b, e := sourceMaven(original, "https://example.org/?a=1&b=2")
	if e != nil {
		t.Fatal(e)
	}
	for _, part := range []string{"<!--keep-->", "<password>secret</password>", "a=1&amp;b=2", "<id>private</id>"} {
		if !strings.Contains(string(b), part) {
			t.Fatal("lost XML", part)
		}
	}
	_, _, e = sourceMaven([]byte(`<settings><mirrors><mirror><mirrorOf>external:*</mirrorOf></mirror></mirrors></settings>`), "https://example.org")
	if e == nil {
		t.Fatal("accepted ambiguous mirror")
	}
}
func TestSourceEnvironmentOverridesRedact(t *testing.T) {
	h := t.TempDir()
	s := SourceManager{Home: h, Getenv: func(k string) string {
		if k == "PIP_INDEX_URL" {
			return "https://secret:token@example.org"
		}
		return ""
	}}
	st, e := s.Show("pip")
	if e != nil {
		t.Fatal(e)
	}
	if len(st.Overrides) != 1 || st.Overrides[0] != "PIP_INDEX_URL" {
		t.Fatal(st)
	}
}
func TestSourceRejectPlanChangedAfterPreview(t *testing.T) {
	h := t.TempDir()
	s := SourceManager{Home: h, StateDir: filepath.Join(h, "state"), Getenv: func(string) string { return "" }}
	p, e := s.Plan("npm", "official")
	if e != nil {
		t.Fatal(e)
	}
	os.WriteFile(filepath.Join(h, ".npmrc"), []byte("manual=1"), 0600)
	if e = p.Execute(context.Background(), false); e == nil {
		t.Fatal("overwrote manual edit after preview")
	}
}
func TestSourceMavenSelfClosingSettings(t *testing.T) {
	_, b, e := sourceMaven([]byte(`<settings xmlns="http://maven.apache.org/SETTINGS/1.0.0"/>`), "https://example.org/")
	if e != nil {
		t.Fatal(e)
	}
	if _, _, e = sourceMaven(b, ""); e != nil {
		t.Fatalf("generated invalid XML: %s: %v", b, e)
	}
}

type sourceTestTransport func(*http.Request) (*http.Response, error)

func (f sourceTestTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestSourceCheckMetadataEndpoints(t *testing.T) {
	paths := map[string]string{"pip": "/simple/pip/", "npm": "/-/ping", "maven": "/maven2/org/apache/maven/maven/maven-metadata.xml", "go": "/golang.org/x/text/@v/list"}
	for tool, path := range paths {
		t.Run(tool, func(t *testing.T) {
			client := &http.Client{Transport: sourceTestTransport(func(r *http.Request) (*http.Response, error) {
				if r.URL.Scheme != "https" || r.URL.Path != path {
					t.Errorf("unsafe or wrong metadata URL: %s", r.URL)
				}
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("metadata")), Header: make(http.Header)}, nil
			})}
			if e := (SourceManager{}).Check(context.Background(), client, tool, "official"); e != nil {
				t.Fatal(e)
			}
		})
	}
}
func TestSourceRejectSymlinkTarget(t *testing.T) {
	h := t.TempDir()
	target := filepath.Join(h, "real")
	os.WriteFile(target, []byte("keep"), 0600)
	if e := os.Symlink(target, filepath.Join(h, ".npmrc")); e != nil {
		t.Skip("symlink unavailable", e)
	}
	s := SourceManager{Home: h, Getenv: func(string) string { return "" }}
	if _, e := s.Plan("npm", "official"); e == nil {
		t.Fatal("accepted symbolic link")
	}
}
func TestSourceMavenSelfClosingMirrors(t *testing.T) {
	_, b, e := sourceMaven([]byte(`<settings><mirrors/></settings>`), "https://example.org/")
	if e != nil {
		t.Fatal(e)
	}
	if !strings.Contains(string(b), "<mirrors><mirror>") {
		t.Fatalf("managed mirror outside mirrors: %s", b)
	}
}
func TestSourcePipColonOption(t *testing.T) {
	b, e := sourceEdit("pip", []byte("[global]\nindex-url: https://old.example/simple\ntimeout: 42\n"), "https://pypi.org/simple/")
	if e != nil {
		t.Fatal(e)
	}
	if strings.Count(string(b), "index-url") != 1 || strings.Contains(string(b), "old.example") || !strings.Contains(string(b), "timeout: 42") {
		t.Fatalf("invalid pip config: %s", b)
	}
}
func TestSourceTranslatePlans(t *testing.T) {
	h := t.TempDir()
	s := SourceManager{Home: h, StateDir: filepath.Join(h, "state"), Getenv: func(string) string { return "" }, Translate: func(k string) string { return "localized: " + k }}
	p, e := s.Plan("npm", "official")
	if e != nil {
		t.Fatal(e)
	}
	if !strings.HasPrefix(p.Title, "localized: ") || !strings.HasPrefix(p.Steps[0].Description, "localized: ") {
		t.Fatal("plan not translated")
	}
	if e = p.Execute(context.Background(), false); e != nil {
		t.Fatal(e)
	}
	entries, _ := os.ReadDir(s.StateDir)
	r, e := s.RestorePlan(strings.TrimSuffix(entries[0].Name(), ".json"))
	if e != nil {
		t.Fatal(e)
	}
	if !strings.HasPrefix(r.Title, "localized: ") || !strings.HasPrefix(r.Steps[0].Description, "localized: ") {
		t.Fatal("restore not translated")
	}
}

func TestSourceRejectBackupMetadataTampering(t *testing.T) {
	for _, field := range []string{"Existed", "Mode", "Tool", "Path", "ManagedHash", "OriginalHash"} {
		t.Run(field, func(t *testing.T) {
			h := t.TempDir()
			s := SourceManager{Home: h, StateDir: filepath.Join(h, "state"), Getenv: func(string) string { return "" }}
			config := filepath.Join(h, ".npmrc")
			if e := os.WriteFile(config, []byte("auth=keep\n"), 0600); e != nil {
				t.Fatal(e)
			}
			p, e := s.Plan("npm", "official")
			if e != nil {
				t.Fatal(e)
			}
			if e = p.Execute(context.Background(), false); e != nil {
				t.Fatal(e)
			}
			before, _ := os.ReadFile(config)
			entries, _ := os.ReadDir(s.StateDir)
			bp := filepath.Join(s.StateDir, entries[0].Name())
			raw, _ := os.ReadFile(bp)
			var backup map[string]any
			if e = json.Unmarshal(raw, &backup); e != nil {
				t.Fatal(e)
			}
			switch field {
			case "Existed":
				backup[field] = false
			case "Mode":
				backup[field] = float64(0644)
			case "Tool":
				backup[field] = "pip"
			case "Path":
				backup[field] = filepath.Join(h, "other")
			default:
				backup[field] = "changed"
			}
			raw, _ = json.Marshal(backup)
			if e = os.WriteFile(bp, raw, 0600); e != nil {
				t.Fatal(e)
			}
			restore, e := s.RestorePlan(strings.TrimSuffix(entries[0].Name(), ".json"))
			if e == nil {
				e = restore.Execute(context.Background(), false)
			}
			if e == nil {
				t.Fatalf("accepted changed %s", field)
			}
			after, e := os.ReadFile(config)
			if e != nil || string(after) != string(before) {
				t.Fatalf("configuration changed after corrupt backup: %s %v", after, e)
			}
		})
	}
}
func TestSourcePipCommentedGlobalSection(t *testing.T) {
	for _, comment := range []string{"# keep section comment", "; keep section comment"} {
		t.Run(comment, func(t *testing.T) {
			h := t.TempDir()
			s := SourceManager{Home: h, Getenv: func(string) string { return "" }}
			st, e := s.Show("pip")
			if e != nil {
				t.Fatal(e)
			}
			os.MkdirAll(filepath.Dir(st.Path), 0700)
			original := "[global] " + comment + "\nindex-url=https://old.example/simple/\ntimeout=42\n"
			if e = os.WriteFile(st.Path, []byte(original), 0600); e != nil {
				t.Fatal(e)
			}
			st, e = s.Show("pip")
			if e != nil || st.URL != "https://old.example/simple/" {
				t.Fatalf("show missed commented section: %#v %v", st, e)
			}
			b, e := sourceEdit("pip", []byte(original), "https://pypi.org/simple/")
			if e != nil {
				t.Fatal(e)
			}
			if strings.Count(string(b), "[global]") != 1 || strings.Count(string(b), "index-url") != 1 || strings.Contains(string(b), "old.example") || !strings.Contains(string(b), comment) {
				t.Fatalf("invalid edited config: %s", b)
			}
		})
	}
}
