package sysinfo

import "testing"

func TestProductName(t *testing.T) {
	cases := []struct{ product, build, want string }{
		{"Windows 10 Pro", "19045", "Windows 10 Pro"},
		{"Windows 10 Pro", "22631", "Windows 11 Pro"},
		{"Windows 10 Home", "26200", "Windows 11 Home"},
		{"Windows Server 2022 Standard", "20348", "Windows Server 2022 Standard"},
		{"Windows 10 Pro", "", "Windows 10 Pro"},
	}
	for _, c := range cases {
		if got := productName(c.product, c.build); got != c.want {
			t.Errorf("productName(%q, %q) = %q, want %q", c.product, c.build, got, c.want)
		}
	}
}
