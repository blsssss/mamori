package fwprobe

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/blsssss/mamori/internal/check"
	"github.com/blsssss/mamori/internal/elevate"
	"github.com/blsssss/mamori/internal/secprod"
)

var (
	enforcing       = finding(check.Pass, "fw.state.enforcing")
	stateOff        = finding(check.Fail, "fw.state.off")
	serviceStopped  = finding(check.Fail, "fw.state.service")
	stateUnknown    = finding(check.Warn, "fw.state.unavailable")
	thirdPartyOn    = finding(check.Pass, "fw.state.third_party")
	thirdPartyOff   = finding(check.Fail, "fw.state.third_party")
	ruleThirdParty  = finding(check.Warn, "fw.rule.third_party")
	ruleBlocked     = finding(check.Pass, "fw.rule.blocked")
	ruleRestored    = finding(check.Pass, "fw.rule.restored")
	ruleNotRestored = finding(check.Warn, "fw.rule.not_restored")
	ruleNotEnforced = finding(check.Fail, "fw.rule.not_enforced")
	ruleRemoveFail  = finding(check.Error, "fw.rule.remove_failed")
	ruleControlFail = finding(check.Skip, "fw.rule.control_failed")
	ruleAddFail     = finding(check.Error, "fw.rule.add_failed")
	needsAdmin      = finding(check.Skip, "fw.rule.needs_admin")
	uacDeclined     = finding(check.Skip, "fw.rule.uac_declined")
	elevateError    = finding(check.Error, "fw.rule.elevate_error")
	gpOverride      = finding(check.Skip, "fw.rule.gp_override")
	policyDenied    = finding(check.Pass, "fw.policy.blocked", "reason", reasonDenied)
	policyDropped   = finding(check.Pass, "fw.policy.blocked", "reason", reasonTimeout)
	policyReachable = finding(check.Fail, "fw.policy.reachable")
	policyUnclear   = finding(check.Warn, "fw.policy.inconclusive")
	policyNone      = finding(check.Skip, "fw.policy.none")
)

func fs(f ...check.Finding) []check.Finding { return f }

func TestVerdict(t *testing.T) {
	cases := []struct {
		name     string
		findings []check.Finding
		status   check.Status
		code     string
	}{
		{"rule blocked and restored", fs(enforcing, policyNone, ruleBlocked, ruleRestored), check.Pass, "fw.verdict.works"},
		{"blocked but not restored", fs(enforcing, policyNone, ruleBlocked, ruleNotRestored), check.Warn, "fw.verdict.unverified"},
		{"blocked, removal failed", fs(enforcing, policyNone, ruleBlocked, ruleRemoveFail), check.Warn, "fw.verdict.unverified"},
		{"rule not enforced", fs(enforcing, policyNone, ruleNotEnforced), check.Fail, "fw.verdict.not_enforced"},
		{"rule works, policy target reachable", fs(enforcing, policyReachable, ruleBlocked, ruleRestored), check.Fail, "fw.verdict.not_enforced"},
		{"policy target dropped, uac declined", fs(enforcing, policyDropped, uacDeclined), check.Pass, "fw.verdict.works"},
		{"policy target denied, uac declined", fs(enforcing, policyDenied, uacDeclined), check.Pass, "fw.verdict.works"},
		{"one target blocked, another reachable", fs(enforcing, policyDenied, policyReachable, needsAdmin), check.Fail, "fw.verdict.not_enforced"},
		{"on, no admin, no targets", fs(enforcing, policyNone, needsAdmin), check.Warn, "fw.verdict.unverified"},
		{"on, inconclusive target", fs(enforcing, policyUnclear, needsAdmin), check.Warn, "fw.verdict.unverified"},
		{"on, elevation failed", fs(enforcing, policyNone, elevateError), check.Warn, "fw.verdict.unverified"},
		{"on, control failed", fs(enforcing, policyNone, ruleControlFail), check.Warn, "fw.verdict.unverified"},
		{"on, rule not added", fs(enforcing, policyNone, ruleAddFail), check.Warn, "fw.verdict.unverified"},
		{"on, local rules overridden by group policy", fs(enforcing, policyNone, gpOverride), check.Warn, "fw.verdict.unverified"},
		{"profile off, no admin", fs(stateOff, policyNone, needsAdmin), check.Fail, "fw.verdict.disabled"},
		{"service stopped, uac declined", fs(serviceStopped, policyNone, uacDeclined), check.Fail, "fw.verdict.disabled"},
		{"profile off, rule not enforced", fs(stateOff, policyNone, ruleNotEnforced), check.Fail, "fw.verdict.not_enforced"},
		{"profile off, policy target dropped", fs(stateOff, policyDropped, needsAdmin), check.Fail, "fw.verdict.disabled"},
		{"service stopped, policy target dropped", fs(serviceStopped, policyDropped, uacDeclined), check.Fail, "fw.verdict.disabled"},
		{"profile off, policy target denied locally", fs(stateOff, policyDenied, needsAdmin), check.Pass, "fw.verdict.works"},
		{"profile off, third party off", fs(stateOff, thirdPartyOff, policyNone, needsAdmin), check.Fail, "fw.verdict.disabled"},
		{"third party only", fs(stateOff, thirdPartyOn, policyNone, ruleThirdParty), check.Warn, "fw.verdict.unverified"},
		{"third party only, policy target dropped", fs(stateOff, thirdPartyOn, policyDropped, ruleThirdParty), check.Pass, "fw.verdict.works"},
		{"third party off, policy target dropped", fs(stateOff, thirdPartyOff, policyDropped, needsAdmin), check.Fail, "fw.verdict.disabled"},
		{"state unknown", fs(stateUnknown, stateUnknown, policyNone, needsAdmin), check.Warn, "fw.verdict.unverified"},
		{"state unknown, policy target dropped", fs(stateUnknown, policyDropped, needsAdmin), check.Warn, "fw.verdict.unverified"},
		{"state unknown, policy target denied", fs(stateUnknown, policyDenied, needsAdmin), check.Pass, "fw.verdict.works"},
		{"state unknown, rule works", fs(stateUnknown, policyNone, ruleBlocked, ruleRestored), check.Pass, "fw.verdict.works"},
		{"no findings", nil, check.Warn, "fw.verdict.unverified"},
	}
	for _, c := range cases {
		status, code := verdict(c.findings)
		if status != c.status || code != c.code {
			t.Errorf("%s: verdict = %s %s, want %s %s", c.name, status, code, c.status, c.code)
		}
	}
}

func TestRuleVerdict(t *testing.T) {
	added := finding(check.Pass, "fw.rule.added")
	removed := finding(check.Pass, "fw.rule.removed")
	cases := []struct {
		name     string
		findings []check.Finding
		status   check.Status
		code     string
	}{
		{"works", fs(added, ruleBlocked, removed, ruleRestored), check.Pass, "fw.verdict.works"},
		{"not enforced", fs(added, ruleNotEnforced, removed), check.Fail, "fw.verdict.not_enforced"},
		{"not restored", fs(added, ruleBlocked, removed, ruleNotRestored), check.Skip, "fw.verdict.unverified"},
		{"removal failed", fs(added, ruleBlocked, ruleRemoveFail), check.Skip, "fw.verdict.unverified"},
		{"control failed", fs(ruleControlFail), check.Skip, "fw.verdict.unverified"},
		{"add failed", fs(ruleAddFail), check.Skip, "fw.verdict.unverified"},
		{"group policy override", fs(gpOverride), check.Skip, "fw.verdict.unverified"},
	}
	for _, c := range cases {
		status, code := ruleVerdict(c.findings)
		if status != c.status || code != c.code {
			t.Errorf("%s: ruleVerdict = %s %s, want %s %s", c.name, status, code, c.status, c.code)
		}
	}
}

func TestStateFindings(t *testing.T) {
	private := secprod.FirewallProfile{Name: "private", Enabled: true, Active: true}
	public := secprod.FirewallProfile{Name: "public", Enabled: true, Active: true}
	domainOff := secprod.FirewallProfile{Name: "domain"}
	publicOff := secprod.FirewallProfile{Name: "public", Active: true}
	cases := []struct {
		name     string
		fw       secprod.WindowsFirewall
		fwErr    error
		products []secprod.Product
		prodErr  error
		want     []check.Finding
	}{
		{
			name: "enforcing",
			fw:   secprod.WindowsFirewall{Service: "running", Profiles: []secprod.FirewallProfile{domainOff, private, public}},
			want: fs(finding(check.Pass, "fw.state.enforcing", "profiles", "private, public")),
		},
		{
			name: "active profile off",
			fw:   secprod.WindowsFirewall{Service: "running", Profiles: []secprod.FirewallProfile{domainOff, private, publicOff}},
			want: fs(finding(check.Fail, "fw.state.off", "profile", "public")),
		},
		{
			name: "no active profile",
			fw:   secprod.WindowsFirewall{Service: "running", Profiles: []secprod.FirewallProfile{domainOff}},
			want: fs(finding(check.Warn, "fw.state.unavailable", "error", "windows firewall: no active profile")),
		},
		{
			name: "service stopped",
			fw:   secprod.WindowsFirewall{Service: "stopped", Profiles: []secprod.FirewallProfile{private}},
			want: fs(finding(check.Fail, "fw.state.service", "state", "stopped")),
		},
		{
			name:    "nothing readable",
			fwErr:   errors.New("access denied"),
			prodErr: errors.New("class not registered"),
			want: fs(
				finding(check.Warn, "fw.state.unavailable", "error", "windows firewall: access denied"),
				finding(check.Warn, "fw.state.unavailable", "error", "security center: class not registered"),
			),
		},
		{
			name: "third party products",
			fw:   secprod.WindowsFirewall{Service: "running", Profiles: []secprod.FirewallProfile{publicOff}},
			products: []secprod.Product{
				{Name: "Kaspersky", State: secprod.StateOn},
				{Name: "Comodo", State: secprod.StateSnoozed},
				{Name: "Old", State: secprod.StateExpired},
				{Name: "Odd", State: secprod.StateUnknown},
			},
			want: fs(
				finding(check.Fail, "fw.state.off", "profile", "public"),
				finding(check.Pass, "fw.state.third_party", "name", "Kaspersky", "state", "on"),
				finding(check.Fail, "fw.state.third_party", "name", "Comodo", "state", "snoozed"),
				finding(check.Fail, "fw.state.third_party", "name", "Old", "state", "expired"),
				finding(check.Warn, "fw.state.third_party", "name", "Odd", "state", "unknown"),
			),
		},
	}
	for _, c := range cases {
		got := stateFindings(c.fw, c.fwErr, c.products, c.prodErr)
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s:\n got %v\nwant %v", c.name, got, c.want)
		}
	}
}

func TestThirdPartyOnly(t *testing.T) {
	on := secprod.WindowsFirewall{Service: "running", Profiles: []secprod.FirewallProfile{{Name: "public", Enabled: true, Active: true}}}
	off := secprod.WindowsFirewall{Service: "running", Profiles: []secprod.FirewallProfile{{Name: "public", Active: true}}}
	kaspersky := secprod.Product{Name: "Kaspersky", State: secprod.StateOn}
	comodo := secprod.Product{Name: "Comodo", State: secprod.StateOn}
	snoozed := secprod.Product{Name: "Snoozed", State: secprod.StateSnoozed}
	cases := []struct {
		name     string
		fw       secprod.WindowsFirewall
		fwErr    error
		products []secprod.Product
		want     string
	}{
		{"windows firewall enforcing", on, nil, []secprod.Product{kaspersky}, ""},
		{"windows firewall off", off, nil, []secprod.Product{kaspersky, snoozed, comodo}, "Kaspersky, Comodo"},
		{"windows firewall unknown", secprod.WindowsFirewall{}, errors.New("x"), []secprod.Product{kaspersky}, "Kaspersky"},
		{"no third party on", off, nil, []secprod.Product{snoozed}, ""},
		{"nothing at all", off, nil, nil, ""},
	}
	for _, c := range cases {
		if got := thirdPartyOnly(c.fw, c.fwErr, c.products); got != c.want {
			t.Errorf("%s: thirdPartyOnly = %q, want %q", c.name, got, c.want)
		}
	}
}

func TestElevatedFindings(t *testing.T) {
	res := check.Result{Findings: fs(ruleBlocked, ruleRestored)}
	cases := []struct {
		name string
		res  check.Result
		err  error
		want []check.Finding
	}{
		{"result merged", res, nil, res.Findings},
		{"uac declined", check.Result{}, elevate.ErrCancelled, fs(uacDeclined)},
		{"uac declined, wrapped", check.Result{}, fmt.Errorf("probe: %w", elevate.ErrCancelled), fs(uacDeclined)},
		{"other error", check.Result{}, errors.New("exit code 3"), fs(finding(check.Error, "fw.rule.elevate_error", "error", "exit code 3"))},
	}
	for _, c := range cases {
		if got := elevatedFindings(c.res, c.err); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s:\n got %v\nwant %v", c.name, got, c.want)
		}
	}
}
