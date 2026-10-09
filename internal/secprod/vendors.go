package secprod

import (
	"path"
	"strings"
)

// vendor lists where a security product leaves traces: paths start with an environment variable
// in %VAR% form, process names are lowercase, services are lowercase path.Match patterns, kind
// suite means the product line has a firewall in some editions
type vendor struct {
	name      string
	kind      string
	paths     []string
	processes []string
	services  []string
}

var vendors = []vendor{
	{
		name:      "Kaspersky",
		kind:      VendorSuite,
		paths:     []string{`%ProgramFiles(x86)%\Kaspersky Lab`, `%ProgramFiles%\Kaspersky Lab`, `%ProgramData%\Kaspersky Lab`},
		processes: []string{"avp.exe", "avpui.exe", "kavfs.exe"},
		services:  []string{"avp*", "kavfs"},
	},
	{
		name:      "Dr.Web",
		kind:      VendorSuite,
		paths:     []string{`%ProgramFiles%\DrWeb`, `%ProgramFiles(x86)%\DrWeb`, `%ProgramData%\Doctor Web`},
		processes: []string{"dwservice.exe", "dwengine.exe", "spideragent.exe"},
		services:  []string{"drweb*"},
	},
	{
		name:      "ESET",
		kind:      VendorSuite,
		paths:     []string{`%ProgramFiles%\ESET`, `%ProgramData%\ESET`},
		processes: []string{"ekrn.exe", "egui.exe"},
		services:  []string{"ekrn"},
	},
	{
		name:      "Avast",
		kind:      VendorSuite,
		paths:     []string{`%ProgramFiles%\Avast Software`, `%ProgramData%\Avast Software`},
		processes: []string{"avastsvc.exe", "avastui.exe"},
		services:  []string{"avast*", "aswbidsagent"},
	},
	{
		name:      "AVG",
		kind:      VendorSuite,
		paths:     []string{`%ProgramFiles%\AVG`, `%ProgramData%\AVG`},
		processes: []string{"avgsvc.exe", "avgui.exe"},
		services:  []string{"avg*"},
	},
	{
		name:      "Avira",
		kind:      VendorAV,
		paths:     []string{`%ProgramFiles%\Avira`, `%ProgramFiles(x86)%\Avira`, `%ProgramData%\Avira`},
		processes: []string{"avira.servicehost.exe", "avira.systray.exe", "avguard.exe"},
		services:  []string{"avira*", "antivir*"},
	},
	{
		name:      "Bitdefender",
		kind:      VendorSuite,
		paths:     []string{`%ProgramFiles%\Bitdefender`, `%ProgramData%\Bitdefender`},
		processes: []string{"bdagent.exe", "vsserv.exe", "bdservicehost.exe"},
		services:  []string{"vsserv"},
	},
	{
		name:      "Norton",
		kind:      VendorSuite,
		paths:     []string{`%ProgramFiles%\Norton`, `%ProgramFiles%\Norton Security`, `%ProgramFiles(x86)%\Norton Security`},
		processes: []string{"nortonsvc.exe", "nortonui.exe", "nortonsecurity.exe"},
		services:  []string{"norton*", "nswscsvc"},
	},
	{
		name:      "McAfee",
		kind:      VendorSuite,
		paths:     []string{`%ProgramFiles%\McAfee`, `%ProgramFiles(x86)%\McAfee`, `%ProgramData%\McAfee`},
		processes: []string{"mcshield.exe", "mfemms.exe", "mfevtps.exe"},
		services:  []string{"mcafee*", "mcshield", "mfemms", "mfevtp", "mfefire"},
	},
	{
		name:      "Comodo",
		kind:      VendorSuite,
		paths:     []string{`%ProgramFiles%\COMODO`, `%ProgramFiles(x86)%\COMODO`},
		processes: []string{"cmdagent.exe", "cis.exe"},
		services:  []string{"cmdagent"},
	},
	{
		name:      "360 Total Security",
		kind:      VendorAV,
		paths:     []string{`%ProgramFiles(x86)%\360\Total Security`, `%ProgramFiles%\360\Total Security`},
		processes: []string{"qhactivedefense.exe", "qhsafetray.exe", "360tray.exe"},
		services:  []string{"qhactivedefense"},
	},
	{
		name:      "Sophos",
		kind:      VendorAV,
		paths:     []string{`%ProgramFiles%\Sophos`, `%ProgramFiles(x86)%\Sophos`, `%ProgramData%\Sophos`},
		processes: []string{"sophoshealth.exe", "sspservice.exe", "savservice.exe"},
		services:  []string{"sophos*", "savservice", "sspservice"},
	},
	{
		name:      "Malwarebytes",
		kind:      VendorAV,
		paths:     []string{`%ProgramFiles%\Malwarebytes\Anti-Malware`, `%ProgramData%\Malwarebytes\MBAMService`},
		processes: []string{"mbamservice.exe", "mbamtray.exe", "malwarebytes.exe"},
		services:  []string{"mbamservice"},
	},
	{
		name:      "Microsoft Defender",
		kind:      VendorAV,
		paths:     []string{`%ProgramFiles%\Windows Defender`, `%ProgramData%\Microsoft\Windows Defender`},
		processes: []string{"msmpeng.exe", "nissrv.exe"},
		services:  []string{"windefend", "wdnissvc"},
	},
	{
		name:      "Windows Firewall Control",
		kind:      VendorFW,
		paths:     []string{`%ProgramFiles%\Malwarebytes\Windows Firewall Control`},
		processes: []string{"wfc.exe"},
		services:  []string{"_wfcs"},
	},
	{
		name:      "ZoneAlarm",
		kind:      VendorFW,
		paths:     []string{`%ProgramFiles(x86)%\CheckPoint\ZoneAlarm`},
		processes: []string{"zatray.exe", "vsmon.exe"},
		services:  []string{"vsmon"},
	},
	{
		name:      "TinyWall",
		kind:      VendorFW,
		paths:     []string{`%ProgramFiles(x86)%\TinyWall`, `%ProgramFiles%\TinyWall`},
		processes: []string{"tinywall.exe"},
		services:  []string{"tinywall"},
	},
	{
		name:      "simplewall",
		kind:      VendorFW,
		paths:     []string{`%ProgramFiles%\simplewall`},
		processes: []string{"simplewall.exe"},
	},
	{
		name:      "GlassWire",
		kind:      VendorFW,
		paths:     []string{`%ProgramFiles(x86)%\GlassWire`, `%ProgramFiles%\GlassWire`},
		processes: []string{"glasswire.exe", "gwctlsrv.exe"},
		services:  []string{"glasswire*"},
	},
}

// findEvidence matches the vendor table against the machine: expand resolves the %VAR% in a path,
// isDir reports a directory on disk, processes and services are names as Windows returns them
func findEvidence(table []vendor, expand func(string) string, isDir func(string) bool, processes, services []string) []Hit {
	running := make(map[string]string, len(processes))
	for _, p := range processes {
		if k := strings.ToLower(p); running[k] == "" {
			running[k] = p
		}
	}
	hits := []Hit{}
	for _, v := range table {
		for _, p := range v.paths {
			if dir := expand(p); isDir(dir) {
				hits = append(hits, Hit{Vendor: v.name, Kind: v.kind, Evidence: EvidencePath, Value: dir})
			}
		}
		for _, p := range v.processes {
			if name, ok := running[p]; ok {
				hits = append(hits, Hit{Vendor: v.name, Kind: v.kind, Evidence: EvidenceProcess, Value: name})
			}
		}
		for _, s := range services {
			if matchAny(v.services, strings.ToLower(s)) {
				hits = append(hits, Hit{Vendor: v.name, Kind: v.kind, Evidence: EvidenceService, Value: s})
			}
		}
	}
	return hits
}

func matchAny(patterns []string, name string) bool {
	for _, p := range patterns {
		if ok, _ := path.Match(p, name); ok {
			return true
		}
	}
	return false
}
