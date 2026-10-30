//go:build !linux

package core

import "rockdev/internal/i18n"

func Lock(dir string) (func(), error) { return nil, i18n.Errorf("系统修改仅支持 Linux") }
