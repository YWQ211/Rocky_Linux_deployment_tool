package core

import "testing"

func TestSystemRuntimeTarget(t *testing.T) {
	for _, tc := range []struct {
		name, entry, target string
		want                bool
	}{
		{"rocky8-python3", "/usr/bin/python3", "/usr/libexec/platform-python3.6", true},
		{"rocky8-bin-alias", "/bin/python3", "/usr/libexec/platform-python3.6", true},
		{"rocky8-versioned", "/usr/bin/python3.6", "/usr/libexec/platform-python3.6", true},
		{"rocky8-versioned-alias", "/bin/python3.6", "/usr/libexec/platform-python3.6", true},
		{"regular-python", "/usr/bin/python3", "/usr/bin/python3.9", true},
		{"java-alternatives", "/usr/bin/java", "/usr/lib/jvm/java-17/bin/java", true},
		{"go-system", "/usr/bin/go", "/usr/lib/golang/bin/go", true},
		{"custom-python", "/usr/bin/python3", "/opt/python/bin/python3", false},
		{"user-python", "/home/dev/bin/python3", "/usr/libexec/platform-python3.6", false},
		{"usr-local-alias", "/usr/local/bin/python3", "/usr/libexec/platform-python3.6", false},
		{"other-libexec", "/usr/bin/python3", "/usr/libexec/custom-python", false},
		{"suffix-lookalike", "/usr/bin/python3", "/usr/libexec/platform-python3.6-custom", false},
		{"nested-lookalike", "/usr/bin/python3", "/usr/libexec/platform-python3.6/python3", false},
		{"different-command", "/usr/bin/go", "/usr/libexec/platform-python3.6", false},
		{"directory-lookalike", "/usr/binary/python3", "/usr/libexec/platform-python3.6", false},
		{"entry-traversal", "/usr/bin/../local/bin/python3", "/usr/libexec/platform-python3.6", false},
		{"target-traversal", "/usr/bin/python3", "/usr/lib/../../opt/python3", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := systemRuntimeTarget(tc.entry, tc.target); got != tc.want {
				t.Fatalf("systemRuntimeTarget(%q, %q)=%v; want %v", tc.entry, tc.target, got, tc.want)
			}
		})
	}
}
