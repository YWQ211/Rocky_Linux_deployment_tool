//go:build windows

package core

import "syscall"

func bundlePublishDir(stage, target string) error {
	from, e := syscall.UTF16PtrFromString(stage)
	if e != nil {
		return e
	}
	to, e := syscall.UTF16PtrFromString(target)
	if e != nil {
		return e
	}
	// MoveFile (without MoveFileEx's replacement flag) refuses an existing target.
	return syscall.MoveFile(from, to)
}
