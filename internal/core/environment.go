package core

import (
	"rockdev/internal/i18n"
	"strings"
)

type Environment struct {
	ID, Name string
	Packages []string
	Checks   []Command
}

var Environments = []Environment{
	{"python", "Python / pip / venv", []string{"python3", "python3-pip"}, []Command{{"python3", []string{"-c", "import sys, venv; print(sys.version)"}}, {"python3", []string{"-m", "pip", "--version"}}}},
	{"java", "Java JDK / Maven", []string{"java-17-openjdk-devel", "maven"}, []Command{{"java", []string{"-version"}}, {"javac", []string{"-version"}}, {"mvn", []string{"-version"}}}},
	{"go", "Go", []string{"golang"}, []Command{{"go", []string{"version"}}}},
	{"node", "Node.js / npm", []string{"nodejs", "npm"}, []Command{{"node", []string{"-e", "console.log(process.version)"}}, {"npm", []string{"--version"}}}},
	{"build", "Git / GCC / make", []string{"git", "gcc", "gcc-c++", "make", "pkgconf-pkg-config"}, []Command{{"git", []string{"--version"}}, {"gcc", []string{"--version"}}, {"make", []string{"--version"}}}},
}

func (m Manager) EnvironmentPlan(names []string) (Plan, error) {
	if len(names) == 0 {
		return Plan{}, i18n.Errorf("请选择环境：python java go node build")
	}
	var selected []Environment
	seen := map[string]bool{}
	for _, name := range names {
		found := false
		for _, env := range Environments {
			if env.ID == name {
				found = true
				if !seen[name] {
					selected = append(selected, env)
					seen[name] = true
				}
				break
			}
		}
		if !found {
			return Plan{}, i18n.Errorf("未知开发环境 %q", name)
		}
	}
	packages := []string{}
	for _, e := range selected {
		for _, pkg := range e.Packages {
			if pkg == "java-17-openjdk-devel" && m.System.Major == "10" {
				pkg = "java-21-openjdk-devel"
			}
			packages = append(packages, pkg)
		}
	}
	p := Plan{Title: m.tr("安装环境: ") + strings.Join(names, ", ") + m.tr("（使用已启用 DNF 仓库版本）")}
	p.Steps = append(p.Steps, m.CommandStep(Command{"dnf", append([]string{"-y", "--setopt=gpgcheck=1", "--setopt=*.gpgcheck=1", "install"}, packages...)}))
	for _, e := range selected {
		for _, c := range e.Checks {
			p.Steps = append(p.Steps, m.CommandStep(c))
		}
	}
	return p, nil
}
