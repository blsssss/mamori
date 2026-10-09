package sysinfo

import (
	"os"
	"os/user"
	"runtime"
	"strconv"
	"strings"

	"golang.org/x/sys/windows/registry"
)

func Get() Info {
	info := Info{Arch: runtime.GOARCH}
	info.Host, _ = os.Hostname()
	if u, err := user.Current(); err == nil {
		info.User = u.Username
	}
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows NT\CurrentVersion`, registry.QUERY_VALUE)
	if err != nil {
		info.Product = "Windows"
		return info
	}
	defer k.Close()
	product, _, _ := k.GetStringValue("ProductName")
	info.Version, _, _ = k.GetStringValue("DisplayVersion")
	build, _, _ := k.GetStringValue("CurrentBuildNumber")
	ubr, _, _ := k.GetIntegerValue("UBR")
	info.Product = productName(product, build)
	info.Build = build
	if ubr > 0 {
		info.Build += "." + strconv.FormatUint(ubr, 10)
	}
	return info
}

// windows 11 still reports "Windows 10" in ProductName, the build number tells them apart
func productName(product, build string) string {
	if n, err := strconv.Atoi(build); err == nil && n >= 22000 {
		return strings.Replace(product, "Windows 10", "Windows 11", 1)
	}
	return product
}
