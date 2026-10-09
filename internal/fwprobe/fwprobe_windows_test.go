package fwprobe

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/go-ole/go-ole/oleutil"

	"github.com/blsssss/mamori/internal/check"
)

// the live tests only read: no rule is added or removed and nothing runs elevated

func TestPolicyObjectReadOnly(t *testing.T) {
	if testing.Short() {
		t.Skip("reads the live firewall configuration")
	}
	err := withCOM(func() error {
		fw, err := openPolicy()
		if err != nil {
			return err
		}
		defer fw.close()
		all, err := fw.names("")
		if err != nil {
			return err
		}
		if len(all) == 0 {
			t.Error("no rules enumerated, windows ships with built-in ones")
		}
		leftovers, err := fw.names(rulePrefix)
		if err != nil {
			return err
		}
		t.Logf("%d rules, %d left by the rule test: %v", len(all), len(leftovers), leftovers)
		apply, err := fw.localRulesApply()
		if err != nil {
			return err
		}
		t.Logf("local rules apply: %v", apply)

		current, err := oleutil.GetProperty(fw.policy, "CurrentProfileTypes")
		if err != nil {
			return err
		}
		t.Logf("active profiles bitmask: %v", current.Value())
		for _, p := range []struct {
			name string
			id   int32
		}{{"domain", 1}, {"private", 2}, {"public", 4}} {
			v, err := oleutil.GetProperty(fw.policy, "FirewallEnabled", p.id)
			if err != nil {
				return err
			}
			t.Logf("%s profile enabled: %v", p.name, v.Value())
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestControlHosts(t *testing.T) {
	if testing.Short() {
		t.Skip("connects to the control hosts")
	}
	target, err := control(context.Background(), connect)
	if err != nil {
		t.Fatalf("no control host answered: %v", err)
	}
	if !slices.Contains(controlTargets, target) {
		t.Errorf("control target %q is not one of %v", target, controlTargets)
	}
}

func TestRunWithoutAdmin(t *testing.T) {
	if testing.Short() {
		t.Skip("reads the live firewall state and connects to the policy target")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	opts := check.Options{PolicyTargets: []string{" http://www.msftconnecttest.com/connecttest.txt ", ""}}
	res := Run(ctx, opts, nil)
	for _, f := range res.Findings {
		t.Logf("%-5s %s %v", f.Status, f.Code, f.Params)
		if f.Code == "fw.rule.added" {
			t.Errorf("a rule was added without elevation: %v", f.Params)
		}
	}
	if !has(res.Findings, "fw.rule.needs_admin") && !has(res.Findings, "fw.rule.third_party") {
		t.Error("without Elevate and elevation the rule test must be reported as skipped")
	}
	t.Logf("verdict: %s %s", res.Status, res.Code)
}
