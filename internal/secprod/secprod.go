// Package secprod finds the antivirus and firewall products of the machine and reads their state.
package secprod

import (
	"context"
	"slices"
	"strings"

	"github.com/blsssss/mamori/internal/check"
)

// decodeProductState reads productState of root\SecurityCenter2. Microsoft does not document its
// layout, this is a community heuristic: bits 12-15 hold the product state, bits 4-7 the signature
// status
func decodeProductState(v uint32) (state, signature string) {
	switch v & 0xF000 {
	case 0x0000:
		state = StateOff
	case 0x1000:
		state = StateOn
	case 0x2000:
		state = StateSnoozed
	case 0x3000:
		state = StateExpired
	default:
		state = StateUnknown
	}
	switch v & 0x00F0 {
	case 0x00:
		signature = SignatureUpToDate
	case 0x10:
		signature = SignatureOutOfDate
	default:
		signature = SignatureUnknown
	}
	return state, signature
}

// wscState maps WSC_SECURITY_PRODUCT_STATE from iwscapi.h
func wscState(v int32) string {
	switch v {
	case 0:
		return StateOn
	case 1:
		return StateOff
	case 2:
		return StateSnoozed
	case 3:
		return StateExpired
	}
	return StateUnknown
}

// wscSignature maps WSC_SECURITY_SIGNATURE_STATUS from iwscapi.h
func wscSignature(v int32) string {
	switch v {
	case 0:
		return SignatureOutOfDate
	case 1:
		return SignatureUpToDate
	}
	return SignatureUnknown
}

// serviceStateName maps SERVICE_STATUS.dwCurrentState from winsvc.h
func serviceStateName(v uint32) string {
	switch v {
	case 1:
		return "stopped"
	case 2:
		return "start_pending"
	case 3:
		return "stop_pending"
	case 4:
		return "running"
	case 5:
		return "continue_pending"
	case 6:
		return "pause_pending"
	case 7:
		return "paused"
	}
	return "unknown"
}

// fwAction maps NET_FW_ACTION from icftypes.h
func fwAction(v int64) string {
	switch v {
	case 0:
		return "block"
	case 1:
		return "allow"
	}
	return "unknown"
}

// isDefender matches the names Defender registers under, "Windows Defender" and
// "Microsoft Defender Antivirus", but not Bitdefender
func isDefender(name string) bool {
	n := strings.ToLower(name)
	return strings.HasPrefix(n, "windows defender") || strings.HasPrefix(n, "microsoft defender")
}

// isWindowsFirewall recognises the built-in firewall. Security Center gives it a localized name
// ("Брандмауэр Windows" on Russian Windows) but always the firewall.cpl remediation path; the exact
// names cover WMI, which has no path, and keep a third-party product named after it
func isWindowsFirewall(p Product) bool {
	if strings.HasSuffix(strings.ToLower(strings.TrimSpace(p.Path)), `\system32\firewall.cpl`) {
		return true
	}
	switch strings.ToLower(strings.TrimSpace(p.Name)) {
	case "windows firewall", "windows defender firewall", "microsoft defender firewall",
		"брандмауэр windows", "брандмауэр защитника windows", "брандмауэр microsoft defender":
		return true
	}
	return false
}

// withoutWindowsFirewall drops the built-in firewall should Security Center list it: Firewall reads
// it from its service and policy, and as a product it would pass for a third-party firewall
func withoutWindowsFirewall(list []Product) []Product {
	return slices.DeleteFunc(list, func(p Product) bool {
		return p.Kind == KindFirewall && isWindowsFirewall(p)
	})
}

type productReader func(ctx context.Context, kind string) ([]Product, error)

// securityCenter takes both lists from one source, so that a report never mixes WSC and WMI
func securityCenter(ctx context.Context, r *check.Recorder, wsc, wmi productReader) (source string, av, fw []Product) {
	av, fw, err := antivirusAndFirewalls(ctx, wsc)
	if err == nil {
		return SourceWSC, av, fw
	}
	r.Add(check.Warn, "inv.wsc.unavailable", "error", err.Error())
	av, fw, err = antivirusAndFirewalls(ctx, wmi)
	if err == nil {
		return SourceWMI, av, fw
	}
	r.Add(check.Warn, "inv.wmi.unavailable", "error", err.Error())
	return "", []Product{}, []Product{}
}

func antivirusAndFirewalls(ctx context.Context, read productReader) (av, fw []Product, err error) {
	if av, err = read(ctx, KindAntivirus); err != nil {
		return nil, nil, err
	}
	if fw, err = read(ctx, KindFirewall); err != nil {
		return nil, nil, err
	}
	return av, fw, nil
}

func otherAntivirusOn(products []Product) bool {
	for _, p := range products {
		if p.State == StateOn && !isDefender(p.Name) {
			return true
		}
	}
	return false
}

// productStatus judges one Security Center product, a firewall has no signatures to be out of date
func productStatus(p Product) check.Status {
	switch p.State {
	case StateOn:
		if p.Kind == KindFirewall || p.Signature == SignatureUpToDate {
			return check.Pass
		}
		return check.Warn
	case StateSnoozed, StateUnknown:
		return check.Warn
	}
	return check.Fail
}

func serviceStatus(state string) check.Status {
	if state == "running" {
		return check.Pass
	}
	return check.Fail
}

// profileStatus: a disabled profile only matters while the machine is on that kind of network
func profileStatus(p FirewallProfile) check.Status {
	switch {
	case p.Enabled:
		return check.Pass
	case p.Active:
		return check.Fail
	}
	return check.Warn
}

// defenderStatus: Defender steps aside when another antivirus takes over, so then it is a note,
// not a gap
func defenderStatus(d Defender, otherAV bool) check.Status {
	switch {
	case d.passive():
		return check.Warn
	case d.protecting():
		return check.Pass
	case otherAV:
		return check.Warn
	}
	return check.Fail
}

type level int

const (
	levelUnknown level = iota
	levelNone
	levelDegraded
	levelOK
)

func avLevel(d Data) level {
	if d.Source == "" {
		// windows server has no security center, there Defender is the only witness
		if d.Defender != nil && d.Defender.protecting() {
			return levelOK
		}
		return levelUnknown
	}
	// Security Center keeps listing Defender after its service stops (seen as snoozed), yet a stopped
	// Defender protects nothing whatever it last reported
	defenderDown := d.Defender != nil && d.Defender.Service != "running"
	l := levelNone
	for _, p := range d.Antivirus {
		if defenderDown && isDefender(p.Name) {
			continue
		}
		switch {
		case p.State == StateOn && p.Signature == SignatureUpToDate:
			return levelOK
		case p.State == StateOn, p.State == StateSnoozed:
			l = levelDegraded
		case p.State == StateUnknown && l == levelNone:
			l = levelUnknown
		}
	}
	return l
}

func fwLevel(d Data, winfwRead bool) level {
	if d.WindowsFirewall.Enforcing() {
		return levelOK
	}
	l := levelNone
	for _, p := range d.Firewalls {
		switch {
		case p.State == StateOn:
			return levelOK
		case p.State == StateSnoozed:
			l = levelDegraded
		case p.State == StateUnknown && l == levelNone:
			l = levelUnknown
		}
	}
	// with either source unread a working firewall may have gone unseen
	if l == levelNone && (!winfwRead || d.Source == "") {
		return levelUnknown
	}
	return l
}

// verdict needs one antivirus that is on with current signatures and one firewall that filters:
// Windows Firewall enforcing on the active profiles or a third-party firewall on. winfwRead is
// false when the state of Windows Firewall could not be read
func verdict(d Data, winfwRead bool) (check.Status, string) {
	av, fw := avLevel(d), fwLevel(d, winfwRead)
	switch {
	case av == levelNone && fw == levelNone:
		return check.Fail, "inv.verdict.none"
	case av == levelNone:
		return check.Fail, "inv.verdict.no_av"
	case fw == levelNone:
		return check.Fail, "inv.verdict.no_fw"
	case av == levelUnknown || fw == levelUnknown:
		return check.Error, "inv.verdict.unknown"
	case av == levelDegraded || fw == levelDegraded:
		return check.Warn, "inv.verdict.outdated"
	}
	return check.Pass, "inv.verdict.ok"
}
