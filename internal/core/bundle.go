package core

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"io"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"regexp"
	"rockdev/internal/i18n"
	"sort"
	"strings"
)

const bundleMaxBytes int64 = 4 << 30
const bundleMaxFiles = 10000
const bundleMaxMetadata int64 = 8 << 20
const bundleManifestName = "manifest.json"

var bundleVersion = regexp.MustCompile(`^(8|9|10)(\.[0-9]+)?$`)

type bundlePaddingWriter struct{}

func (bundlePaddingWriter) Write(p []byte) (int, error) {
	for _, b := range p {
		if b != 0 {
			return 0, bundleError()
		}
	}
	return len(p), nil
}

type BundleFile struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
}
type BundleManifest struct {
	Schema  int          `json:"schema"`
	OS      string       `json:"os"`
	Version string       `json:"version"`
	Major   string       `json:"major"`
	Arch    string       `json:"arch"`
	Files   []BundleFile `json:"files"`
}

func bundleError() error { return i18n.Errorf("Invalid or unsafe offline bundle") }
func bundlePath(s string) bool {
	return s != "" && s != "." && path.Clean(s) == s && !strings.HasPrefix(s, "/") && !strings.ContainsAny(s, "\\:\x00\r\n") && s != ".." && !strings.HasPrefix(s, "../")
}
func bundleAllowed(s string) bool {
	return bundlePath(s) && !strings.EqualFold(path.Ext(s), ".repo") && (strings.HasSuffix(s, ".rpm") || strings.HasPrefix(s, "repodata/"))
}
func bundleSystem(b BundleManifest) error {
	s := System{OS: "linux", ID: b.OS, Version: b.Version, Major: b.Major, Arch: b.Arch}
	if s.Validate() != nil || !bundleVersion.MatchString(b.Version) || strings.Split(b.Version, ".")[0] != b.Major {
		return bundleError()
	}
	return nil
}
func bundleDestination(p string) error {
	if !filepath.IsAbs(p) || strings.ContainsAny(p, "\r\n\x00$%") {
		return bundleError()
	}
	if _, e := os.Lstat(p); !os.IsNotExist(e) {
		return i18n.Errorf("Bundle destination already exists or is inaccessible: %s", p)
	}
	st, e := os.Stat(filepath.Dir(p))
	if e != nil || !st.IsDir() {
		return bundleError()
	}
	return nil
}
func bundleMetadata(data []byte, files map[string]BundleFile) error {
	if len(data) == 0 {
		return bundleError()
	}
	var doc struct {
		XMLName xml.Name `xml:"repomd"`
		Data    []struct {
			Location struct {
				Href string `xml:"href,attr"`
			} `xml:"location"`
		} `xml:"data"`
	}
	if xml.Unmarshal(data, &doc) != nil || len(doc.Data) == 0 {
		return bundleError()
	}
	for _, d := range doc.Data {
		if !bundleAllowed(d.Location.Href) {
			return bundleError()
		}
		if _, ok := files[d.Location.Href]; !ok {
			return bundleError()
		}
	}
	return nil
}

// readBundle streams all payload bytes through SHA256. Only the bounded manifest
// and repomd.xml are retained. A nonempty stage receives files, but publication
// and signature verification happen only after the complete archive is checked.
func readBundle(archive, stage string) (BundleManifest, error) {
	var b BundleManifest
	f, e := os.Open(archive)
	if e != nil {
		return b, e
	}
	defer f.Close()
	gz, e := gzip.NewReader(f)
	if e != nil {
		return b, e
	}
	defer gz.Close()
	expanded := &io.LimitedReader{R: gz, N: bundleMaxBytes + 1}
	tr := tar.NewReader(expanded)
	files := map[string]BundleFile{}
	seen := map[string]bool{}
	var manifest, repomd []byte
	buffer := make([]byte, 32<<10)
	for {
		h, e := tr.Next()
		if e == io.EOF {
			break
		}
		if e != nil {
			return b, e
		}
		if h.Typeflag != tar.TypeReg || !bundlePath(h.Name) || (h.Name != bundleManifestName && !bundleAllowed(h.Name)) || h.Size < 0 || h.Size > bundleMaxBytes || len(seen) >= bundleMaxFiles+1 || seen[h.Name] {
			return b, bundleError()
		}
		seen[h.Name] = true
		for key := range h.PAXRecords {
			if strings.HasPrefix(key, "GNU.sparse") {
				return b, bundleError()
			}
		}
		if h.Name == bundleManifestName || h.Name == "repodata/repomd.xml" {
			if h.Size > bundleMaxMetadata {
				return b, bundleError()
			}
		}
		hash := sha256.New()
		var captured bytes.Buffer
		writers := []io.Writer{hash}
		var output *os.File
		if h.Name == bundleManifestName || h.Name == "repodata/repomd.xml" {
			writers = append(writers, &captured)
		}
		if stage != "" && h.Name != bundleManifestName {
			destination := filepath.Join(stage, filepath.FromSlash(h.Name))
			if e = os.MkdirAll(filepath.Dir(destination), 0755); e != nil {
				return b, e
			}
			output, e = os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
			if e != nil {
				return b, e
			}
			writers = append(writers, output)
		}
		n, copyErr := io.CopyBuffer(io.MultiWriter(writers...), io.LimitReader(tr, h.Size+1), buffer)
		if output != nil {
			if ce := output.Close(); copyErr == nil {
				copyErr = ce
			}
		}
		if copyErr != nil {
			return b, copyErr
		}
		if n != h.Size || expanded.N <= 0 {
			return b, bundleError()
		}
		if h.Name == bundleManifestName {
			manifest = captured.Bytes()
		} else {
			files[h.Name] = BundleFile{h.Name, hex.EncodeToString(hash.Sum(nil)), n}
		}
		if h.Name == "repodata/repomd.xml" {
			repomd = captured.Bytes()
		}
	}
	// Reject hidden concatenated tar archives and require gzip CRC validation.
	if _, e = io.CopyBuffer(bundlePaddingWriter{}, expanded, buffer); e != nil || expanded.N <= 0 {
		return b, bundleError()
	}
	dec := json.NewDecoder(bytes.NewReader(manifest))
	dec.DisallowUnknownFields()
	if dec.Decode(&b) != nil || dec.Decode(new(any)) != io.EOF || b.Schema != 1 || bundleSystem(b) != nil || len(files) != len(b.Files) {
		return b, bundleError()
	}
	checked := map[string]bool{}
	for _, entry := range b.Files {
		actual, ok := files[entry.Path]
		if !ok || checked[entry.Path] || !bundleAllowed(entry.Path) || entry != actual {
			return b, bundleError()
		}
		checked[entry.Path] = true
		for parent := path.Dir(entry.Path); parent != "."; parent = path.Dir(parent) {
			if _, exists := files[parent]; exists {
				return b, bundleError()
			}
		}
	}
	if e = bundleMetadata(repomd, files); e != nil {
		return b, e
	}
	return b, nil
}
func InspectBundle(archive string) (BundleManifest, error) { return readBundle(archive, "") }

// bundleSource opens an unchanged ordinary file. The descriptor is checked
// against both the walk entry and the current path before bytes are consumed.
func bundleSource(p string, info os.FileInfo) (*os.File, error) {
	f, e := os.Open(p)
	if e != nil {
		return nil, e
	}
	opened, e := f.Stat()
	current, le := os.Lstat(p)
	if e != nil || le != nil || !current.Mode().IsRegular() || !os.SameFile(info, opened) || !os.SameFile(info, current) {
		f.Close()
		return nil, bundleError()
	}
	return f, nil
}
func bundleHardlink(info os.FileInfo) bool {
	v := reflect.ValueOf(info.Sys())
	if v.IsValid() && v.Kind() == reflect.Pointer {
		v = v.Elem()
		if v.Kind() == reflect.Struct {
			n := v.FieldByName("Nlink")
			return n.IsValid() && n.CanUint() && n.Uint() > 1
		}
	}
	return false
}
func (m Manager) BundleExportPlan(repoDir, archive string) (Plan, error) {
	if e := m.System.Validate(); e != nil {
		return Plan{}, e
	}
	if !filepath.IsAbs(repoDir) {
		return Plan{}, bundleError()
	}
	if e := bundleDestination(archive); e != nil {
		return Plan{}, e
	}
	collect := func() (BundleManifest, map[string]os.FileInfo, error) {
		b := BundleManifest{Schema: 1, OS: m.System.ID, Version: m.System.Version, Major: m.System.Major, Arch: m.System.Arch}
		if e := bundleSystem(b); e != nil {
			return b, nil, e
		}
		infos := map[string]os.FileInfo{}
		entries := map[string]BundleFile{}
		var repomd []byte
		var total int64
		buffer := make([]byte, 32<<10)
		e := filepath.Walk(repoDir, func(p string, info os.FileInfo, e error) error {
			if e != nil {
				return e
			}
			if info.Mode()&os.ModeSymlink != 0 {
				return bundleError()
			}
			if info.IsDir() {
				return nil
			}
			if !info.Mode().IsRegular() || bundleHardlink(info) {
				return bundleError()
			}
			rel, e := filepath.Rel(repoDir, p)
			if e != nil {
				return e
			}
			rel = filepath.ToSlash(rel)
			if strings.EqualFold(path.Ext(rel), ".repo") {
				return nil
			}
			for _, old := range infos {
				if os.SameFile(old, info) {
					return bundleError()
				}
			}
			// Reserve space for tar headers and the bounded manifest so our own export
			// always fits the reader's decompressed archive limit.
			if !bundleAllowed(rel) || info.Size() < 0 || info.Size() > bundleMaxBytes-bundleMaxMetadata-int64(bundleMaxFiles)*2048-total || len(infos) >= bundleMaxFiles {
				return bundleError()
			}
			if rel == "repodata/repomd.xml" && info.Size() > bundleMaxMetadata {
				return bundleError()
			}
			f, e := bundleSource(p, info)
			if e != nil {
				return e
			}
			hash := sha256.New()
			var metadata bytes.Buffer
			var destination io.Writer = hash
			if rel == "repodata/repomd.xml" {
				destination = io.MultiWriter(hash, &metadata)
			}
			n, e := io.CopyBuffer(destination, io.LimitReader(f, info.Size()+1), buffer)
			ce := f.Close()
			if e != nil {
				return e
			}
			if ce != nil {
				return ce
			}
			if n != info.Size() {
				return bundleError()
			}
			total += n
			entry := BundleFile{rel, hex.EncodeToString(hash.Sum(nil)), n}
			b.Files = append(b.Files, entry)
			entries[rel] = entry
			infos[rel] = info
			if rel == "repodata/repomd.xml" {
				repomd = metadata.Bytes()
			}
			return nil
		})
		sort.Slice(b.Files, func(i, j int) bool { return b.Files[i].Path < b.Files[j].Path })
		if e == nil {
			e = bundleMetadata(repomd, entries)
		}
		return b, infos, e
	}
	if _, _, e := collect(); e != nil {
		return Plan{}, e
	}
	return Plan{Title: m.tr("Export offline RPM bundle"), Steps: []Step{{Description: m.tr("Validate repository and atomically publish bundle"), Do: func(ctx context.Context) error {
		b, infos, e := collect()
		if e != nil {
			return e
		}
		f, e := os.CreateTemp(filepath.Dir(archive), ".rockdev-bundle-*")
		if e != nil {
			return e
		}
		defer os.Remove(f.Name())
		gz := gzip.NewWriter(f)
		tw := tar.NewWriter(gz)
		write := func() error {
			raw, e := json.Marshal(b)
			if e != nil {
				return e
			}
			if int64(len(raw)) > bundleMaxMetadata {
				return bundleError()
			}
			if e = tw.WriteHeader(&tar.Header{Name: bundleManifestName, Mode: 0644, Size: int64(len(raw)), Typeflag: tar.TypeReg}); e != nil {
				return e
			}
			if _, e = tw.Write(raw); e != nil {
				return e
			}
			buffer := make([]byte, 32<<10)
			for _, entry := range b.Files {
				if e = ctx.Err(); e != nil {
					return e
				}
				source, e := bundleSource(filepath.Join(repoDir, filepath.FromSlash(entry.Path)), infos[entry.Path])
				if e != nil {
					return e
				}
				if e = tw.WriteHeader(&tar.Header{Name: entry.Path, Mode: 0644, Size: entry.Size, Typeflag: tar.TypeReg}); e != nil {
					source.Close()
					return e
				}
				hash := sha256.New()
				n, e := io.CopyBuffer(io.MultiWriter(tw, hash), io.LimitReader(source, entry.Size+1), buffer)
				ce := source.Close()
				if e != nil {
					return e
				}
				if ce != nil {
					return ce
				}
				if n != entry.Size || hex.EncodeToString(hash.Sum(nil)) != entry.SHA256 {
					return bundleError()
				}
			}
			return nil
		}
		e = write()
		for _, close := range []func() error{tw.Close, gz.Close, f.Sync, f.Close} {
			if ce := close(); e == nil {
				e = ce
			}
		}
		if e != nil {
			return e
		}
		return os.Link(f.Name(), archive)
	}}}}, nil
}
func (m Manager) BundleImportPlan(archive, target string) (Plan, error) {
	if e := m.System.Validate(); e != nil {
		return Plan{}, e
	}
	if e := bundleDestination(target); e != nil {
		return Plan{}, e
	}
	validate := func(stage string) (BundleManifest, error) {
		b, e := readBundle(archive, stage)
		if e != nil {
			return b, e
		}
		// Exact minor match avoids packages requiring a newer Rocky minor release.
		if b.OS != m.System.ID || b.Major != m.System.Major || b.Version != m.System.Version || b.Arch != m.System.Arch {
			return b, i18n.Errorf("Bundle platform does not match target system")
		}
		return b, nil
	}
	if _, e := validate(""); e != nil {
		return Plan{}, e
	}
	return Plan{Title: m.tr("Import offline RPM bundle"), Steps: []Step{{Description: m.tr("Verify bundle and RPM signatures before publishing repository"), Do: func(ctx context.Context) error {
		stage, e := os.MkdirTemp(filepath.Dir(target), ".rockdev-import-*")
		if e != nil {
			return e
		}
		defer os.RemoveAll(stage)
		b, e := validate(stage)
		if e != nil {
			return e
		}
		for _, entry := range b.Files {
			if e = ctx.Err(); e != nil {
				return e
			}
			if strings.HasSuffix(entry.Path, ".rpm") {
				if e = VerifyRPM(ctx, m.Runner, filepath.Join(stage, filepath.FromSlash(entry.Path))); e != nil {
					return e
				}
			}
		}
		base := (&url.URL{Scheme: "file", Path: filepath.ToSlash(target)}).String()
		config := "[rockdev-local]\nname=Rockdev local RPM repository\nbaseurl=" + base + "\nenabled=0\ngpgcheck=1\nmodule_hotfixes=1\n"
		if e = os.WriteFile(filepath.Join(stage, "rockdev-local.repo"), []byte(config), 0644); e != nil {
			return e
		}
		if e = bundleDestination(target); e != nil {
			return e
		}
		if e = os.Chmod(stage, 0755); e != nil {
			return e
		}
		return bundlePublishDir(stage, target)
	}}}}, nil
}
