//go:build !linux

package core

import "fmt"

func Lock(dir string) (func(), error) { return nil, fmt.Errorf("系统修改仅支持 Linux") }
