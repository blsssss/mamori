// Package fwprobe checks that the firewall filters traffic, not only that it is switched on.
package fwprobe

import (
	"context"
	"errors"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/blsssss/mamori/internal/check"
	"github.com/blsssss/mamori/internal/elevate"
	"github.com/blsssss/mamori/internal/secprod"
)

// RuleProbe is the probe name the elevated copy of the program runs the rule test under
const RuleProbe = "firewall-rule"

func stateFindings(w secprod.WindowsFirewall, werr error, products []secprod.Product, perr error) []check.Finding {
	var fs []check.Finding
	switch {
	case werr != nil:
		fs = append(fs, finding(check.Warn, "fw.state.unavailable", "error", "windows firewall: "+werr.Error()))
	case w.Service != "running":
		fs = append(fs, finding(check.Fail, "fw.state.service", "state", w.Service))
	case w.Enforcing():
		var active []string
		for _, p := range w.Profiles {
			if p.Active {
				active = append(active, p.Name)
			}
		}
		fs = append(fs, finding(check.Pass, "fw.state.enforcing", "profiles", strings.Join(active, ", ")))
	default:
		for _, p := range w.Profiles {
			if p.Active && !p.Enabled {
				fs = append(fs, finding(check.Fail, "fw.state.off", "profile", p.Name))
			}
		}
		if len(fs) == 0 {
			fs = append(fs, finding(check.Warn, "fw.state.unavailable", "error", "windows firewall: no active profile"))
		}
	}
	if perr != nil {
		fs = append(fs, finding(check.Warn, "fw.state.unavailable", "error", "security center: "+perr.Error()))
	}
	for _, p := range products {
		status := check.Fail
		switch p.State {
		case secprod.StateOn:
			status = check.Pass
		case secprod.StateUnknown:
			status = check.Warn
		}
		fs = append(fs, finding(status, "fw.state.third_party", "name", p.Name, "state", p.State))
	}
	return fs
}

// thirdPartyOnly names the firewalls that are on while Windows Firewall is not known to enforce:
// their rules are not Windows Firewall rules, so the rule test cannot test them
func thirdPartyOnly(w secprod.WindowsFirewall, werr error, products []secprod.Product) string {
	if werr == nil && w.Enforcing() {
		return ""
	}
	var names []string
	for _, p := range products {
		if p.State == secprod.StateOn {
			names = append(names, p.Name)
		}
	}
	return strings.Join(names, ", ")
}

func elevatedFindings(res check.Result, err error) []check.Finding {
	switch {
	case errors.Is(err, elevate.ErrCancelled):
		return []check.Finding{finding(check.Skip, "fw.rule.uac_declined")}
	case err != nil:
		return []check.Finding{finding(check.Error, "fw.rule.elevate_error", "error", err.Error())}
	}
	return res.Findings
}

func verdict(fs []check.Finding) (check.Status, string) {
	var on, off bool
	for _, f := range fs {
		if strings.HasPrefix(f.Code, "fw.state.") {
			on = on || f.Status == check.Pass
			off = off || f.Status == check.Fail
		}
	}
	switch {
	case has(fs, "fw.rule.not_enforced") || has(fs, "fw.policy.reachable"):
		return check.Fail, "fw.verdict.not_enforced"
	case ruleProved(fs) || policyProved(fs, on):
		return check.Pass, "fw.verdict.works"
	case off && !on:
		return check.Fail, "fw.verdict.disabled"
	}
	return check.Warn, "fw.verdict.unverified"
}

// ruleVerdict decides the rule test when it runs on its own in the elevated copy
func ruleVerdict(fs []check.Finding) (check.Status, string) {
	switch {
	case has(fs, "fw.rule.not_enforced"):
		return check.Fail, "fw.verdict.not_enforced"
	case ruleProved(fs):
		return check.Pass, "fw.verdict.works"
	}
	return check.Skip, "fw.verdict.unverified"
}

// a failed connection proves the rule only if the same connection works again once the rule is
// gone, otherwise the network may simply have dropped at that moment
func ruleProved(fs []check.Finding) bool {
	return has(fs, "fw.rule.blocked") && has(fs, "fw.rule.restored")
}

// a refusal from the local filtering platform proves filtering on its own; a silent drop does only
// while some firewall is on, otherwise a provider block or a dead host looks exactly the same
func policyProved(fs []check.Finding, firewallOn bool) bool {
	return slices.ContainsFunc(fs, func(f check.Finding) bool {
		return f.Code == "fw.policy.blocked" && (f.Params["reason"] == reasonDenied || firewallOn)
	})
}

func has(fs []check.Finding, code string) bool {
	return slices.ContainsFunc(fs, func(f check.Finding) bool { return f.Code == code })
}

// finding mirrors check.Recorder.Add, which does not return what it built
func finding(status check.Status, code string, kv ...string) check.Finding {
	f := check.Finding{Code: code, Status: status}
	if len(kv) > 1 {
		f.Params = make(map[string]string, len(kv)/2)
		for i := 0; i+1 < len(kv); i += 2 {
			f.Params[kv[i]] = kv[i+1]
		}
	}
	return f
}

func ms(d time.Duration) string {
	return strconv.FormatInt(d.Milliseconds(), 10)
}

// recorder keeps the findings it streams: check.Recorder does not hand them back, and the verdict
// is decided from them
type recorder struct {
	rec  *check.Recorder
	seen []check.Finding
}

func newRecorder(emit check.Emit) *recorder {
	return &recorder{rec: check.Start(check.Firewall, emit)}
}

func (r *recorder) add(fs ...check.Finding) {
	for _, f := range fs {
		r.rec.AddFinding(f)
		r.seen = append(r.seen, f)
	}
}

func (r *recorder) finish(ctx context.Context, decide func([]check.Finding) (check.Status, string)) check.Result {
	if ctx.Err() != nil {
		return r.rec.Finish(check.Skip, "common.cancelled")
	}
	status, code := decide(r.seen)
	return r.rec.Finish(status, code)
}
