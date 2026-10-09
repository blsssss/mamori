package secprod

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/blsssss/mamori/internal/check"
)

// these tests only read: Security Center, service states, firewall settings, process and service names

func TestLiveSources(t *testing.T) {
	if testing.Short() {
		t.Skip("reads the live system")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	for _, kind := range []string{KindAntivirus, KindFirewall, KindAntispyware} {
		list, err := wscProducts(ctx, kind)
		t.Logf("wsc %s: %+v, err: %v", kind, list, err)
		list, err = wmiProducts(ctx, kind)
		t.Logf("wmi %s: %+v, err: %v", kind, list, err)
		list, err = Products(ctx, kind)
		t.Logf("Products(%s): %+v, err: %v", kind, list, err)
	}
	wf, err := Firewall(ctx)
	t.Logf("windows firewall: %+v, enforcing: %v, err: %v", wf, wf.Enforcing(), err)
	state, err := serviceState("WinDefend")
	t.Logf("WinDefend: %s, err: %v", state, err)
	d, err := readDefender(ctx, state)
	if d != nil {
		t.Logf("defender: %+v, passive: %v, protecting: %v", *d, d.passive(), d.protecting())
	} else {
		t.Logf("defender: err: %v, wbem status: %#x", err, wbemStatus(err))
	}
	processes, err := processNames()
	if err != nil || len(processes) == 0 {
		t.Errorf("processNames: %d names, err: %v", len(processes), err)
	}
	services, err := serviceNames()
	if err != nil || len(services) == 0 {
		t.Errorf("serviceNames: %d names, err: %v", len(services), err)
	}
	t.Logf("evidence: %+v", findEvidence(vendors, expand, isDir, processes, services))
}

func TestLiveRun(t *testing.T) {
	if testing.Short() {
		t.Skip("reads the live system")
	}
	known := map[string]bool{
		"inv.av.product": true, "inv.fw.product": true, "inv.wsc.unavailable": true, "inv.wmi.unavailable": true,
		"inv.winfw.service": true, "inv.winfw.profile": true, "inv.winfw.unavailable": true,
		"inv.defender.status": true, "inv.defender.absent": true, "inv.defender.unavailable": true,
		"inv.files.found": true, "inv.files.none": true, "inv.files.unavailable": true,
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	emitted := 0
	res := Run(ctx, check.DefaultOptions(), func(check.ID, check.Finding) { emitted++ })
	out, err := json.MarshalIndent(res, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%s", out)
	if res.Check != check.Inventory || !strings.HasPrefix(res.Code, "inv.verdict.") {
		t.Errorf("result %s %s", res.Check, res.Code)
	}
	if emitted != len(res.Findings) {
		t.Errorf("emitted %d findings, recorded %d", emitted, len(res.Findings))
	}
	for _, f := range res.Findings {
		if !known[f.Code] {
			t.Errorf("unknown finding code %s", f.Code)
		}
	}
}
