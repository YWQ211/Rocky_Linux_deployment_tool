package core

import (
	"bufio"
	"io"
	"os"
	"rockdev/internal/i18n"
	"runtime"
	"strings"
)

type System struct{ ID, Version, Major, Name, OS, Arch string }

func ParseSystem(r io.Reader, platform, arch string) (System, error) {
	values := map[string]string{}
	scan := bufio.NewScanner(r)
	for scan.Scan() {
		line := strings.TrimSpace(scan.Text())
		if strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if ok {
			values[k] = strings.Trim(v, "\"'")
		}
	}
	if err := scan.Err(); err != nil {
		return System{}, err
	}
	if arch == "amd64" {
		arch = "x86_64"
	}
	if arch == "arm64" {
		arch = "aarch64"
	}
	s := System{ID: values["ID"], Version: values["VERSION_ID"], Name: values["PRETTY_NAME"], OS: platform, Arch: arch}
	s.Major = strings.Split(s.Version, ".")[0]
	if s.ID == "" || s.Version == "" {
		return s, i18n.Errorf("os-release 缺少 ID 或 VERSION_ID")
	}
	return s, nil
}

func Detect() (System, error) {
	if runtime.GOOS != "linux" {
		return System{OS: runtime.GOOS, Arch: runtime.GOARCH, Name: runtime.GOOS}, nil
	}
	f, err := os.Open("/etc/os-release")
	if err != nil {
		return System{}, err
	}
	defer f.Close()
	return ParseSystem(f, runtime.GOOS, runtime.GOARCH)
}

func (s System) Validate() error {
	if s.OS != "linux" || s.ID != "rocky" {
		return i18n.Errorf("仅支持 Rocky Linux，当前 %s/%s", s.OS, s.ID)
	}
	if s.Major != "8" && s.Major != "9" && s.Major != "10" {
		return i18n.Errorf("未适配 Rocky %s", s.Major)
	}
	if s.Arch != "x86_64" && s.Arch != "aarch64" {
		return i18n.Errorf("未适配架构 %s", s.Arch)
	}
	return nil
}
