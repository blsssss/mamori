package secprod

import "strings"

const (
	KindAntivirus   = "antivirus"
	KindFirewall    = "firewall"
	KindAntispyware = "antispyware"
)

const (
	StateOn      = "on"
	StateOff     = "off"
	StateSnoozed = "snoozed"
	StateExpired = "expired"
	StateUnknown = "unknown"
)

const (
	SignatureUpToDate  = "up_to_date"
	SignatureOutOfDate = "out_of_date"
	SignatureUnknown   = "unknown"
)

const (
	SourceWSC = "wsc"
	SourceWMI = "wmi"
)

const ServiceMissing = "missing"

type Product struct {
	Name      string `json:"name"`
	Kind      string `json:"kind"`
	State     string `json:"state"`
	Signature string `json:"signature"`
	Path      string `json:"path,omitempty"`
	Timestamp string `json:"timestamp,omitempty"`
	Source    string `json:"source"`
}

type FirewallProfile struct {
	Name     string `json:"name"`
	Enabled  bool   `json:"enabled"`
	Active   bool   `json:"active"`
	Inbound  string `json:"inbound"`
	Outbound string `json:"outbound"`
}

// WindowsFirewall is the built-in Windows Defender Firewall, read from its service and policy rather
// than from Security Center
type WindowsFirewall struct {
	Service  string            `json:"service"`
	Profiles []FirewallProfile `json:"profiles"`
}

// Enforcing reports whether the built-in firewall filters traffic on the active network
func (w WindowsFirewall) Enforcing() bool {
	if w.Service != "running" {
		return false
	}
	active := 0
	for _, p := range w.Profiles {
		if p.Active {
			active++
			if !p.Enabled {
				return false
			}
		}
	}
	return active > 0
}

// Defender is Microsoft Defender Antivirus as it reports itself, apart from Security Center: Mode is
// AMRunningMode, SignatureAge is in days and nil when the stopped service could not answer
type Defender struct {
	Service          string  `json:"service"`
	Mode             string  `json:"mode,omitempty"`
	RealTime         bool    `json:"realtime"`
	Antivirus        bool    `json:"antivirus"`
	SignatureAge     *uint32 `json:"signatureAge,omitempty"`
	SignatureVersion string  `json:"signatureVersion,omitempty"`
}

// passive covers "Passive Mode", "SxS Passive Mode" and "EDR Block Mode", which runs on top of passive
func (d Defender) passive() bool {
	m := strings.ToLower(d.Mode)
	return strings.Contains(m, "passive") || strings.Contains(m, "edr block")
}

func (d Defender) protecting() bool {
	return d.Service == "running" && d.RealTime && d.Antivirus && !d.passive()
}

const (
	VendorAV    = "av"
	VendorFW    = "fw"
	VendorSuite = "suite"
)

const (
	EvidencePath    = "path"
	EvidenceProcess = "process"
	EvidenceService = "service"
)

// Hit is a trace of a known security product on disk, among processes or among services
type Hit struct {
	Vendor   string `json:"vendor"`
	Kind     string `json:"kind"`
	Evidence string `json:"evidence"`
	Value    string `json:"value"`
}

// Data is what the inventory check found: Source is empty when Security Center could not be read,
// Defender is nil when Defender is absent or its status could not be read
type Data struct {
	Source          string          `json:"source"`
	Antivirus       []Product       `json:"antivirus"`
	Firewalls       []Product       `json:"firewalls"`
	WindowsFirewall WindowsFirewall `json:"windowsFirewall"`
	Defender        *Defender       `json:"defender"`
	Files           []Hit           `json:"files"`
}
