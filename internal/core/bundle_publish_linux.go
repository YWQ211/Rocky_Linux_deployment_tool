//go:build linux

package core

import (
	"rockdev/internal/i18n"
	"runtime"
	"syscall"
	"unsafe"
)

// bundlePublishDir uses renameat2 rather than rename: a concurrently created
// empty destination must never be replaced either.
func bundlePublishDir(stage, target string) error {
	var number uintptr
	switch runtime.GOARCH {
	case "amd64":
		number = 316
	case "arm64":
		number = 276
	default:
		return i18n.Errorf("Atomic bundle publication is unsupported on this platform")
	}
	from, e := syscall.BytePtrFromString(stage)
	if e != nil {
		return e
	}
	to, e := syscall.BytePtrFromString(target)
	if e != nil {
		return e
	}
	atFDCWD := ^uintptr(99)
	_, _, errno := syscall.Syscall6(number, atFDCWD, uintptr(unsafe.Pointer(from)), atFDCWD, uintptr(unsafe.Pointer(to)), 1, 0)
	runtime.KeepAlive(from)
	runtime.KeepAlive(to)
	if errno != 0 {
		return errno
	}
	return nil
}
