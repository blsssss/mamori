//go:build !windows

package sysinfo

import (
	"os"
	"runtime"
)

func Get() Info {
	host, _ := os.Hostname()
	return Info{Product: runtime.GOOS, Arch: runtime.GOARCH, Host: host}
}
