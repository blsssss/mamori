package secprod

import (
	"path"
	"slices"
	"strings"
	"testing"
)

func TestVendorTable(t *testing.T) {
	required := []string{
		"Kaspersky", "Dr.Web", "ESET", "Avast", "AVG", "Avira", "Bitdefender", "Norton", "McAfee",
		"Comodo", "360 Total Security", "Sophos", "Malwarebytes", "Microsoft Defender",
	}
	roots := []string{`%ProgramFiles%\`, `%ProgramFiles(x86)%\`, `%ProgramData%\`}
	seen := map[string]bool{}
	for _, v := range vendors {
		if seen[v.name] {
			t.Errorf("%s: listed twice", v.name)
		}
		seen[v.name] = true
		if !slices.Contains([]string{VendorAV, VendorFW, VendorSuite}, v.kind) {
			t.Errorf("%s: kind %q", v.name, v.kind)
		}
		if len(v.paths)+len(v.processes)+len(v.services) == 0 {
			t.Errorf("%s: no evidence to look for", v.name)
		}
		for _, p := range v.paths {
			if !slices.ContainsFunc(roots, func(r string) bool { return strings.HasPrefix(p, r) && len(p) > len(r) }) {
				t.Errorf("%s: path %q does not start with a known folder variable", v.name, p)
			}
		}
		for _, p := range v.processes {
			if p != strings.ToLower(p) || !strings.HasSuffix(p, ".exe") || strings.ContainsAny(p, `\/`) {
				t.Errorf("%s: process %q must be a lowercase file name ending in .exe", v.name, p)
			}
		}
		for _, s := range v.services {
			if _, err := path.Match(s, ""); err != nil || s != strings.ToLower(s) || s == "*" {
				t.Errorf("%s: service pattern %q must be a lowercase, specific path.Match pattern", v.name, s)
			}
		}
	}
	for _, name := range required {
		if !seen[name] {
			t.Errorf("vendor %s is missing", name)
		}
	}
}

func TestFindEvidence(t *testing.T) {
	table := []vendor{
		{name: "Kaspersky", kind: VendorSuite, paths: []string{`%ProgramFiles(x86)%\Kaspersky Lab`}, processes: []string{"avp.exe"}, services: []string{"avp*"}},
		{name: "ESET", kind: VendorSuite, paths: []string{`%ProgramFiles%\ESET`}, processes: []string{"ekrn.exe"}, services: []string{"ekrn"}},
		{name: "simplewall", kind: VendorFW, processes: []string{"simplewall.exe"}},
	}
	expand := func(p string) string {
		return strings.NewReplacer(`%ProgramFiles(x86)%`, `C:\Program Files (x86)`, `%ProgramFiles%`, `C:\Program Files`).Replace(p)
	}
	dirs := map[string]bool{`C:\Program Files (x86)\Kaspersky Lab`: true}
	isDir := func(p string) bool { return dirs[p] }
	processes := []string{"System", "AVP.EXE", "avp.exe", "explorer.exe", "simplewall.exe"}
	services := []string{"AVP21.3", "klif", "WinDefend", "Ekrn2"}

	want := []Hit{
		{Vendor: "Kaspersky", Kind: VendorSuite, Evidence: EvidencePath, Value: `C:\Program Files (x86)\Kaspersky Lab`},
		{Vendor: "Kaspersky", Kind: VendorSuite, Evidence: EvidenceProcess, Value: "AVP.EXE"},
		{Vendor: "Kaspersky", Kind: VendorSuite, Evidence: EvidenceService, Value: "AVP21.3"},
		{Vendor: "simplewall", Kind: VendorFW, Evidence: EvidenceProcess, Value: "simplewall.exe"},
	}
	got := findEvidence(table, expand, isDir, processes, services)
	if !slices.Equal(got, want) {
		t.Errorf("findEvidence =\n%+v\nwant\n%+v", got, want)
	}
	if got := findEvidence(table, expand, isDir, nil, nil); len(got) != 1 {
		t.Errorf("without processes and services only the path remains, got %+v", got)
	}
}
