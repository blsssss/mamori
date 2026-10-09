package netcheck

import (
	"context"
	"fmt"
	"net/netip"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/blsssss/mamori/internal/check"
)

func TestRunLive(t *testing.T) {
	if testing.Short() {
		t.Skip("live network probes")
	}
	res := Run(context.Background(), check.DefaultOptions(), nil)
	for _, f := range res.Findings {
		t.Logf("%-5s %-22s %v", f.Status, f.Code, f.Params)
	}
	t.Logf("verdict %s %s in %d ms", res.Status, res.Code, res.ElapsedMs)
	for _, prefix := range []string{"net.adapter.", "net.dns.", "net.ncsi_dns.", "net.icmp.", "net.tcp.", "net.http."} {
		if !slices.ContainsFunc(res.Findings, func(f check.Finding) bool { return strings.HasPrefix(f.Code, prefix) }) {
			t.Errorf("no %s finding", prefix)
		}
	}
	if !strings.HasPrefix(res.Code, "net.verdict.") {
		t.Errorf("verdict code %q", res.Code)
	}
	// the probes run at the same time, so the check takes one probe timeout, not six
	if limit := (probeTimeout + time.Second).Milliseconds(); res.ElapsedMs > limit {
		t.Errorf("took %d ms, want at most %d", res.ElapsedMs, limit)
	}
}

func TestListAdaptersLive(t *testing.T) {
	if testing.Short() {
		t.Skip("live adapter list")
	}
	list, err := listAdapters()
	if err != nil {
		t.Fatal(err)
	}
	loopback := false
	for _, a := range list {
		t.Logf("%q up=%t loopback=%t ipv4=%v gateways=%v", a.name, a.up, a.loopback, a.ipv4, a.gateways)
		for _, ip := range slices.Concat(a.ipv4, a.gateways) {
			if !ip.Is4() {
				t.Errorf("%q: %s is not ipv4", a.name, ip)
			}
		}
		if a.loopback && slices.Contains(a.ipv4, netip.MustParseAddr("127.0.0.1")) {
			loopback = true
		}
	}
	// windows always lists its loopback pseudo-interface, so this proves the list was walked right
	if !loopback {
		t.Error("no loopback adapter with 127.0.0.1")
	}
}

func TestEchoLive(t *testing.T) {
	if testing.Short() {
		t.Skip("live ping")
	}
	target := netip.MustParseAddr("1.1.1.1")
	reply, err := echo(target, 2*time.Second)
	if err != nil {
		t.Skipf("no reply, ICMP may be filtered: %v", err)
	}
	t.Logf("reply %s", describe(reply))
	if reply.status == ipSuccess && reply.addr != target {
		t.Errorf("reply from %s, want %s", reply.addr, target)
	}
	if reply.ttl == 0 || reply.rtt > 2000 {
		t.Errorf("implausible reply %s, the ICMP_ECHO_REPLY layout is off", describe(reply))
	}
}

func TestEchoLoopbackLive(t *testing.T) {
	if testing.Short() {
		t.Skip("live ping")
	}
	// loopback always answers, so this checks the call and the decoding without the internet
	target := netip.MustParseAddr("127.0.0.1")
	reply, err := echo(target, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("reply %s", describe(reply))
	if reply.status != ipSuccess || reply.addr != target || reply.ttl == 0 {
		t.Errorf("reply %s", describe(reply))
	}
}

func describe(r echoReply) string {
	return fmt.Sprintf("from %s, status %d, rtt %d ms, ttl %d", r.addr, r.status, r.rtt, r.ttl)
}
