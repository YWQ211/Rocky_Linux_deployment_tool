//go:build !linux && !windows

package core

import "rockdev/internal/i18n"

func bundlePublishDir(stage, target string) error {
	return i18n.Errorf("Atomic bundle publication is unsupported on this platform")
}
