package secprod

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/blsssss/mamori/internal/check"
)

func TestDecodeProductState(t *testing.T) {
	cases := []struct {
		name             string
		v                uint32
		state, signature string
	}{
		{"defender on (0x061100)", 397568, StateOn, SignatureUpToDate},
		{"defender off (0x060100)", 393472, StateOff, SignatureUpToDate},
		{"defender snoozed, service stopped (0x062100)", 401664, StateSnoozed, SignatureUpToDate},
		{"defender expired (0x063100)", 0x063100, StateExpired, SignatureUpToDate},
		{"defender on, out of date (0x061110)", 397584, StateOn, SignatureOutOfDate},
		{"third party on (0x041000)", 266240, StateOn, SignatureUpToDate},
		{"third party off (0x040000)", 262144, StateOff, SignatureUpToDate},
		{"third party snoozed (0x042000)", 0x042000, StateSnoozed, SignatureUpToDate},
		{"third party expired, out of date (0x043010)", 0x043010, StateExpired, SignatureOutOfDate},
		{"state nibble outside the enum (0x064100)", 0x064100, StateUnknown, SignatureUpToDate},
		{"signature nibble outside the enum (0x061120)", 0x061120, StateOn, SignatureUnknown},
	}
	for _, c := range cases {
		state, signature := decodeProductState(c.v)
		if state != c.state || signature != c.signature {
			t.Errorf("%s: decodeProductState(%#x) = %s, %s, want %s, %s", c.name, c.v, state, signature, c.state, c.signature)
		}
	}
}

func TestWscEnums(t *testing.T) {
	states := map[int32]string{0: StateOn, 1: StateOff, 2: StateSnoozed, 3: StateExpired, 4: StateUnknown, -1: StateUnknown}
	for v, want := range states {
		if got := wscState(v); got != want {
			t.Errorf("wscState(%d) = %s, want %s", v, got, want)
		}
	}
	signatures := map[int32]string{0: SignatureOutOfDate, 1: SignatureUpToDate, 2: SignatureUnknown}
	for v, want := range signatures {
		if got := wscSignature(v); got != want {
			t.Errorf("wscSignature(%d) = %s, want %s", v, got, want)
		}
	}
}

func TestServiceStateName(t *testing.T) {
	want := map[uint32]string{
		0: "unknown", 1: "stopped", 2: "start_pending", 3: "stop_pending", 4: "running",
		5: "continue_pending", 6: "pause_pending", 7: "paused", 8: "unknown",
	}
	for v, w := range want {
		if got := serviceStateName(v); got != w {
			t.Errorf("serviceStateName(%d) = %s, want %s", v, got, w)
		}
	}
}

func TestFwAction(t *testing.T) {
	want := map[int64]string{0: "block", 1: "allow", 2: "unknown"}
	for v, w := range want {
		if got := fwAction(v); got != w {
			t.Errorf("fwAction(%d) = %s, want %s", v, got, w)
		}
	}
}

func TestIsDefender(t *testing.T) {
	cases := map[string]bool{
		"Windows Defender":             true,
		"Microsoft Defender Antivirus": true,
		"Bitdefender Antivirus Free":   false,
		"Kaspersky Free":               false,
	}
	for name, want := range cases {
		if got := isDefender(name); got != want {
			t.Errorf("isDefender(%q) = %v, want %v", name, got, want)
		}
	}
}

func TestWithoutWindowsFirewall(t *testing.T) {
	list := []Product{
		{Name: "Windows Firewall", Kind: KindFirewall},
		{Name: "Windows Defender Firewall", Kind: KindFirewall},
		{Name: " microsoft defender firewall ", Kind: KindFirewall},
		{Name: "Брандмауэр Windows", Kind: KindFirewall},
		{Name: "Pare-feu Windows", Kind: KindFirewall, Path: `%windir%\system32\firewall.cpl`},
		{Name: "Windows Firewall Control", Kind: KindFirewall},
		{Name: "Kaspersky Free", Kind: KindFirewall},
		{Name: "Windows Defender", Kind: KindAntivirus},
	}
	got := withoutWindowsFirewall(list)
	want := []string{"Windows Firewall Control", "Kaspersky Free", "Windows Defender"}
	if len(got) != len(want) {
		t.Fatalf("withoutWindowsFirewall kept %+v, want %v", got, want)
	}
	for i, p := range got {
		if p.Name != want[i] {
			t.Errorf("withoutWindowsFirewall[%d] = %q, want %q", i, p.Name, want[i])
		}
	}
}

func TestSecurityCenter(t *testing.T) {
	wscAV := []Product{{Name: "Kaspersky", Kind: KindAntivirus, Source: SourceWSC}}
	wscFW := []Product{{Name: "Kaspersky", Kind: KindFirewall, Source: SourceWSC}}
	wmiAV := []Product{{Name: "Kaspersky", Kind: KindAntivirus, Source: SourceWMI}}
	wmiFW := []Product{{Name: "Kaspersky", Kind: KindFirewall, Source: SourceWMI}}
	reader := func(av, fw []Product, avErr, fwErr error) productReader {
		return func(_ context.Context, kind string) ([]Product, error) {
			if kind == KindAntivirus {
				return av, avErr
			}
			return fw, fwErr
		}
	}
	fail := errors.New("fail")
	wscOK := reader(wscAV, wscFW, nil, nil)
	wscNoFW := reader(wscAV, nil, nil, fail)
	wscDown := reader(nil, nil, fail, fail)
	wmiOK := reader(wmiAV, wmiFW, nil, nil)
	wmiDown := reader(nil, nil, fail, fail)

	cases := []struct {
		name     string
		wsc, wmi productReader
		source   string
		findings []string
	}{
		{"wsc answers", wscOK, wmiOK, SourceWSC, nil},
		{"wsc firewalls fail, both lists come from wmi", wscNoFW, wmiOK, SourceWMI, []string{"inv.wsc.unavailable"}},
		{"wsc fails", wscDown, wmiOK, SourceWMI, []string{"inv.wsc.unavailable"}},
		{"both fail", wscDown, wmiDown, "", []string{"inv.wsc.unavailable", "inv.wmi.unavailable"}},
	}
	for _, c := range cases {
		var codes []string
		r := check.Start(check.Inventory, func(_ check.ID, f check.Finding) {
			codes = append(codes, f.Code)
			if f.Status != check.Warn || f.Params["error"] != "fail" {
				t.Errorf("%s: finding %+v", c.name, f)
			}
		})
		source, av, fw := securityCenter(context.Background(), r, c.wsc, c.wmi)
		if source != c.source {
			t.Errorf("%s: source %q, want %q", c.name, source, c.source)
		}
		if av == nil || fw == nil {
			t.Errorf("%s: nil list, av %v, fw %v", c.name, av, fw)
		}
		want := 1
		if c.source == "" {
			want = 0
		}
		if len(av) != want || len(fw) != want {
			t.Errorf("%s: av %+v, fw %+v, want %d of each", c.name, av, fw, want)
		}
		for _, p := range append(av, fw...) {
			if p.Source != c.source {
				t.Errorf("%s: %s %s from %q, want every product from %q", c.name, p.Kind, p.Name, p.Source, c.source)
			}
		}
		if !slices.Equal(codes, c.findings) {
			t.Errorf("%s: findings %v, want %v", c.name, codes, c.findings)
		}
	}
}

func TestProductStatus(t *testing.T) {
	cases := []struct {
		p    Product
		want check.Status
	}{
		{Product{Kind: KindAntivirus, State: StateOn, Signature: SignatureUpToDate}, check.Pass},
		{Product{Kind: KindAntivirus, State: StateOn, Signature: SignatureOutOfDate}, check.Warn},
		{Product{Kind: KindAntivirus, State: StateOn, Signature: SignatureUnknown}, check.Warn},
		{Product{Kind: KindAntivirus, State: StateSnoozed, Signature: SignatureUpToDate}, check.Warn},
		{Product{Kind: KindAntivirus, State: StateOff, Signature: SignatureUpToDate}, check.Fail},
		{Product{Kind: KindAntivirus, State: StateExpired, Signature: SignatureUpToDate}, check.Fail},
		{Product{Kind: KindAntivirus, State: StateUnknown}, check.Warn},
		{Product{Kind: KindFirewall, State: StateOn, Signature: SignatureOutOfDate}, check.Pass},
		{Product{Kind: KindFirewall, State: StateSnoozed}, check.Warn},
		{Product{Kind: KindFirewall, State: StateOff}, check.Fail},
		{Product{Kind: KindFirewall, State: StateExpired}, check.Fail},
		{Product{Kind: KindFirewall, State: StateUnknown}, check.Warn},
	}
	for _, c := range cases {
		if got := productStatus(c.p); got != c.want {
			t.Errorf("productStatus(%+v) = %s, want %s", c.p, got, c.want)
		}
	}
}

func TestProfileStatus(t *testing.T) {
	cases := []struct {
		enabled, active bool
		want            check.Status
	}{
		{true, true, check.Pass},
		{true, false, check.Pass},
		{false, true, check.Fail},
		{false, false, check.Warn},
	}
	for _, c := range cases {
		if got := profileStatus(FirewallProfile{Enabled: c.enabled, Active: c.active}); got != c.want {
			t.Errorf("profileStatus(enabled=%v, active=%v) = %s, want %s", c.enabled, c.active, got, c.want)
		}
	}
}

func TestDefenderStatus(t *testing.T) {
	on := Defender{Service: "running", Mode: "Normal", RealTime: true, Antivirus: true}
	cases := []struct {
		name    string
		d       Defender
		otherAV bool
		want    check.Status
	}{
		{"protecting", on, false, check.Pass},
		{"passive next to another av", Defender{Service: "running", Mode: "Passive Mode", RealTime: true, Antivirus: true}, true, check.Warn},
		{"sxs passive", Defender{Service: "running", Mode: "SxS Passive Mode"}, false, check.Warn},
		{"edr block mode", Defender{Service: "running", Mode: "EDR Block Mode"}, true, check.Warn},
		{"realtime off, no other av", Defender{Service: "running", Mode: "Normal", Antivirus: true}, false, check.Fail},
		{"service stopped, no other av", Defender{Service: "stopped"}, false, check.Fail},
		{"service stopped, another av on", Defender{Service: "stopped"}, true, check.Warn},
		{"realtime flag without the service", Defender{Service: "stopped", RealTime: true, Antivirus: true}, false, check.Fail},
	}
	for _, c := range cases {
		if got := defenderStatus(c.d, c.otherAV); got != c.want {
			t.Errorf("%s: defenderStatus = %s, want %s", c.name, got, c.want)
		}
	}
}

func TestVerdict(t *testing.T) {
	avOn := Product{Name: "Kaspersky", Kind: KindAntivirus, State: StateOn, Signature: SignatureUpToDate}
	avOld := Product{Name: "Kaspersky", Kind: KindAntivirus, State: StateOn, Signature: SignatureOutOfDate}
	avSnoozed := Product{Name: "Kaspersky", Kind: KindAntivirus, State: StateSnoozed, Signature: SignatureUpToDate}
	avExpired := Product{Name: "Kaspersky", Kind: KindAntivirus, State: StateExpired, Signature: SignatureUpToDate}
	avOdd := Product{Name: "Kaspersky", Kind: KindAntivirus, State: StateUnknown, Signature: SignatureUnknown}
	defenderOff := Product{Name: "Windows Defender", Kind: KindAntivirus, State: StateOff, Signature: SignatureUpToDate}
	defenderOn := Product{Name: "Windows Defender", Kind: KindAntivirus, State: StateOn, Signature: SignatureUpToDate}
	defenderSnoozed := Product{Name: "Windows Defender", Kind: KindAntivirus, State: StateSnoozed, Signature: SignatureUpToDate}
	fwOn := Product{Name: "Comodo Firewall", Kind: KindFirewall, State: StateOn}
	fwSnoozed := Product{Name: "Comodo Firewall", Kind: KindFirewall, State: StateSnoozed}
	fwOff := Product{Name: "Comodo Firewall", Kind: KindFirewall, State: StateOff}
	fwOdd := Product{Name: "Comodo Firewall", Kind: KindFirewall, State: StateUnknown}
	enforcing := WindowsFirewall{Service: "running", Profiles: []FirewallProfile{
		{Name: "domain", Enabled: true}, {Name: "private", Enabled: true}, {Name: "public", Enabled: true, Active: true},
	}}
	publicOff := WindowsFirewall{Service: "running", Profiles: []FirewallProfile{
		{Name: "domain", Enabled: true}, {Name: "private", Enabled: true}, {Name: "public", Active: true},
	}}
	stopped := WindowsFirewall{Service: "stopped"}
	serverDefender := &Defender{Service: "running", Mode: "Normal", RealTime: true, Antivirus: true}
	defenderStopped := &Defender{Service: "stopped"}

	cases := []struct {
		name      string
		d         Data
		winfwRead bool
		status    check.Status
		code      string
	}{
		{"av and windows firewall", Data{Source: SourceWSC, Antivirus: []Product{avOn}, WindowsFirewall: enforcing}, true, check.Pass, "inv.verdict.ok"},
		{"av and third-party firewall", Data{Source: SourceWSC, Antivirus: []Product{avOn}, Firewalls: []Product{fwOn}, WindowsFirewall: stopped}, true, check.Pass, "inv.verdict.ok"},
		{"one current av is enough", Data{Source: SourceWMI, Antivirus: []Product{defenderOff, avOn}, WindowsFirewall: enforcing}, true, check.Pass, "inv.verdict.ok"},
		{"signatures out of date", Data{Source: SourceWSC, Antivirus: []Product{avOld}, WindowsFirewall: enforcing}, true, check.Warn, "inv.verdict.outdated"},
		{"av snoozed", Data{Source: SourceWSC, Antivirus: []Product{avSnoozed}, WindowsFirewall: enforcing}, true, check.Warn, "inv.verdict.outdated"},
		{"third-party firewall snoozed", Data{Source: SourceWSC, Antivirus: []Product{avOn}, Firewalls: []Product{fwSnoozed}, WindowsFirewall: stopped}, true, check.Warn, "inv.verdict.outdated"},
		{"only a disabled defender", Data{Source: SourceWMI, Antivirus: []Product{defenderOff}, WindowsFirewall: enforcing}, true, check.Fail, "inv.verdict.no_av"},
		{"no av registered", Data{Source: SourceWSC, WindowsFirewall: enforcing}, true, check.Fail, "inv.verdict.no_av"},
		{"only an expired av", Data{Source: SourceWMI, Antivirus: []Product{avExpired}, WindowsFirewall: enforcing}, true, check.Fail, "inv.verdict.no_av"},
		{"av state unknown", Data{Source: SourceWMI, Antivirus: []Product{avOdd}, WindowsFirewall: enforcing}, true, check.Error, "inv.verdict.unknown"},
		{"av state unknown next to a snoozed one", Data{Source: SourceWMI, Antivirus: []Product{avOdd, avSnoozed}, WindowsFirewall: enforcing}, true, check.Warn, "inv.verdict.outdated"},
		{"defender snoozed, service stopped", Data{Source: SourceWMI, Antivirus: []Product{defenderSnoozed}, WindowsFirewall: enforcing, Defender: defenderStopped}, true, check.Fail, "inv.verdict.no_av"},
		{"defender stale on, service stopped", Data{Source: SourceWMI, Antivirus: []Product{defenderOn}, WindowsFirewall: enforcing, Defender: defenderStopped}, true, check.Fail, "inv.verdict.no_av"},
		{"defender stopped, another av on", Data{Source: SourceWMI, Antivirus: []Product{defenderSnoozed, avOn}, WindowsFirewall: enforcing, Defender: defenderStopped}, true, check.Pass, "inv.verdict.ok"},
		{"defender snoozed, service running", Data{Source: SourceWMI, Antivirus: []Product{defenderSnoozed}, WindowsFirewall: enforcing, Defender: serverDefender}, true, check.Warn, "inv.verdict.outdated"},
		{"defender snoozed, defender unread", Data{Source: SourceWMI, Antivirus: []Product{defenderSnoozed}, WindowsFirewall: enforcing}, true, check.Warn, "inv.verdict.outdated"},
		{"defender snoozed and stopped, firewall stopped", Data{Source: SourceWMI, Antivirus: []Product{defenderSnoozed}, WindowsFirewall: stopped, Defender: defenderStopped}, true, check.Fail, "inv.verdict.none"},
		{"active profile disabled", Data{Source: SourceWSC, Antivirus: []Product{avOn}, WindowsFirewall: publicOff}, true, check.Fail, "inv.verdict.no_fw"},
		{"firewall service stopped, third-party off", Data{Source: SourceWSC, Antivirus: []Product{avOn}, Firewalls: []Product{fwOff}, WindowsFirewall: stopped}, true, check.Fail, "inv.verdict.no_fw"},
		{"firewall service stopped, third-party unknown", Data{Source: SourceWSC, Antivirus: []Product{avOn}, Firewalls: []Product{fwOdd}, WindowsFirewall: stopped}, true, check.Error, "inv.verdict.unknown"},
		{"nothing", Data{Source: SourceWMI, Antivirus: []Product{defenderOff}, WindowsFirewall: stopped}, true, check.Fail, "inv.verdict.none"},
		{"security center unread, defender protects", Data{WindowsFirewall: enforcing, Defender: serverDefender}, true, check.Pass, "inv.verdict.ok"},
		{"security center unread, no defender", Data{WindowsFirewall: enforcing}, true, check.Error, "inv.verdict.unknown"},
		{"security center unread, firewall stopped", Data{WindowsFirewall: stopped, Defender: serverDefender}, true, check.Error, "inv.verdict.unknown"},
		{"windows firewall unread", Data{Source: SourceWSC, Antivirus: []Product{avOn}}, false, check.Error, "inv.verdict.unknown"},
		{"windows firewall unread, no av", Data{Source: SourceWSC}, false, check.Fail, "inv.verdict.no_av"},
		{"windows firewall unread, third-party on", Data{Source: SourceWSC, Antivirus: []Product{avOn}, Firewalls: []Product{fwOn}}, false, check.Pass, "inv.verdict.ok"},
	}
	for _, c := range cases {
		status, code := verdict(c.d, c.winfwRead)
		if status != c.status || code != c.code {
			t.Errorf("%s: verdict = %s %s, want %s %s", c.name, status, code, c.status, c.code)
		}
	}
}
