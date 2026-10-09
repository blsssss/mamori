package netcheck

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"slices"
	"testing"
	"time"

	"github.com/blsssss/mamori/internal/check"
)

func TestVerdict(t *testing.T) {
	cases := []struct {
		name   string
		codes  []string
		status check.Status
		code   string
	}{
		{"online", []string{"net.adapter.ok", codeDNSOK, "net.ncsi_dns.ok", codeICMPOK, codeTCPOK, codeHTTPOK},
			check.Pass, "net.verdict.online"},
		{"online with icmp filtered", []string{"net.adapter.ok", codeDNSOK, "net.icmp.fail", codeTCPOK, codeHTTPOK},
			check.Pass, "net.verdict.online"},
		{"online, http outweighs other probes", []string{codeAdapterNone, "net.dns.fail", "net.tcp.fail", codeHTTPOK},
			check.Pass, "net.verdict.online"},
		{"online through a tun that answers ping", []string{"net.adapter.ok", codeDNSOK, codeICMPLocal, codeTCPOK, codeHTTPOK},
			check.Pass, "net.verdict.online"},
		{"captive portal", []string{"net.adapter.ok", codeDNSOK, "net.ncsi_dns.mismatch", codeTCPOK, codeHTTPCaptive},
			check.Warn, "net.verdict.captive"},
		{"captive portal that blocks tcp", []string{"net.adapter.ok", codeDNSOK, "net.icmp.fail", "net.tcp.fail", codeHTTPCaptive},
			check.Warn, "net.verdict.captive"},
		{"dns broken", []string{"net.adapter.ok", "net.dns.fail", "net.ncsi_dns.fail", codeICMPOK, codeTCPOK, "net.http.fail"},
			check.Warn, "net.verdict.dns_broken"},
		{"dns broken, only icmp", []string{"net.adapter.ok", "net.dns.fail", codeICMPOK, "net.tcp.fail", "net.http.fail"},
			check.Warn, "net.verdict.dns_broken"},
		{"limited", []string{"net.adapter.ok", codeDNSOK, "net.ncsi_dns.ok", codeICMPOK, codeTCPOK, "net.http.fail"},
			check.Warn, "net.verdict.limited"},
		{"limited, only tcp", []string{"net.adapter.ok", codeDNSOK, "net.icmp.fail", codeTCPOK, "net.http.fail"},
			check.Warn, "net.verdict.limited"},
		{"adapter list failed, tcp answers", []string{"net.adapter.error", codeDNSOK, "net.icmp.fail", codeTCPOK, "net.http.fail"},
			check.Warn, "net.verdict.limited"},
		{"no adapter", []string{codeAdapterNone, "net.dns.fail", "net.ncsi_dns.fail", "net.icmp.fail", "net.tcp.fail", "net.http.fail"},
			check.Fail, "net.verdict.offline"},
		{"no adapter, yet tcp and icmp answer", []string{codeAdapterNone, codeDNSOK, codeICMPOK, codeTCPOK, "net.http.fail"},
			check.Fail, "net.verdict.offline"},
		{"tun answers ping and tcp itself", []string{"net.adapter.ok", codeDNSOK, codeICMPLocal, codeTCPOK, "net.http.fail"},
			check.Fail, "net.verdict.offline"},
		{"tun answers ping itself, dns broken", []string{"net.adapter.ok", "net.dns.fail", codeICMPLocal, "net.tcp.fail", "net.http.fail"},
			check.Fail, "net.verdict.offline"},
		{"dns answers from the local cache only", []string{"net.adapter.ok", codeDNSOK, "net.icmp.fail", "net.tcp.fail", "net.http.fail"},
			check.Fail, "net.verdict.offline"},
		{"no findings", nil, check.Fail, "net.verdict.offline"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			status, code := verdict(c.codes)
			if status != c.status || code != c.code {
				t.Errorf("verdict = %s %s, want %s %s", status, code, c.status, c.code)
			}
		})
	}
}

func delayed(d time.Duration, notes ...note) probe {
	return func(ctx context.Context) []note {
		select {
		case <-time.After(d):
		case <-ctx.Done():
		}
		return notes
	}
}

func TestRunKeepsProbeOrder(t *testing.T) {
	var emitted []string
	res := run(context.Background(), func(id check.ID, f check.Finding) {
		if id != check.Internet {
			t.Errorf("emitted for %s", id)
		}
		emitted = append(emitted, f.Code)
	}, []probe{
		delayed(30*time.Millisecond, note{check.Pass, "net.adapter.ok", []string{"name", "Ethernet"}},
			note{check.Pass, "net.adapter.ok", []string{"name", "Wi-Fi"}}),
		delayed(20*time.Millisecond, note{check.Pass, codeDNSOK, nil}),
		delayed(10*time.Millisecond, note{check.Pass, codeTCPOK, nil}),
		delayed(0, note{check.Warn, codeHTTPCaptive, []string{"status", "302"}}),
	})
	want := []string{"net.adapter.ok", "net.adapter.ok", codeDNSOK, codeTCPOK, codeHTTPCaptive}
	if !slices.Equal(emitted, want) {
		t.Errorf("emitted %v, want %v", emitted, want)
	}
	if len(res.Findings) != len(want) {
		t.Fatalf("findings %v", res.Findings)
	}
	for i, f := range res.Findings {
		if f.Code != want[i] {
			t.Errorf("finding %d = %s, want %s", i, f.Code, want[i])
		}
	}
	if p := res.Findings[1].Params; p["name"] != "Wi-Fi" {
		t.Errorf("params = %v", p)
	}
	if res.Check != check.Internet || res.Status != check.Warn || res.Code != "net.verdict.captive" {
		t.Errorf("result = %s %s %s", res.Check, res.Status, res.Code)
	}
}

func TestRunGivesEachProbeItsOwnTimeout(t *testing.T) {
	left := make([]time.Duration, 3)
	probes := make([]probe, len(left))
	for i := range probes {
		probes[i] = func(ctx context.Context) []note {
			if dl, ok := ctx.Deadline(); ok {
				left[i] = time.Until(dl)
			}
			return single(check.Pass, codeTCPOK)
		}
	}
	run(context.Background(), nil, probes)
	for i, d := range left {
		if d <= 0 || d > probeTimeout {
			t.Errorf("probe %d had %s left, want (0, %s]", i, d, probeTimeout)
		}
	}
}

func TestRunStreamsFindings(t *testing.T) {
	first := make(chan struct{})
	res := run(context.Background(), func(_ check.ID, f check.Finding) {
		if f.Code == codeDNSOK {
			close(first)
		}
	}, []probe{
		delayed(0, note{check.Pass, codeDNSOK, nil}),
		// finishes only once the finding of the probe before it is out
		func(ctx context.Context) []note {
			select {
			case <-first:
				return single(check.Pass, codeTCPOK)
			case <-ctx.Done():
				return single(check.Fail, "net.tcp.fail")
			}
		},
	})
	if len(res.Findings) != 2 || res.Findings[1].Code != codeTCPOK {
		t.Errorf("findings %v, the first one waited for the last probe", res.Findings)
	}
}

func TestRunCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	res := run(ctx, func(check.ID, check.Finding) { t.Error("emitted a finding after cancellation") }, []probe{
		func(ctx context.Context) []note {
			return single(check.Fail, "net.tcp.fail", "error", ctx.Err().Error())
		},
	})
	if res.Status != check.Skip || res.Code != "common.cancelled" || len(res.Findings) != 0 {
		t.Errorf("result = %s %s %v", res.Status, res.Code, res.Findings)
	}
}

func TestRunCancelledDoesNotWaitForStuckProbe(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	start := time.Now()
	res := run(ctx, func(_ check.ID, f check.Finding) {
		if f.Code == codeDNSOK {
			cancel()
		}
	}, []probe{
		delayed(0, note{check.Pass, codeDNSOK, nil}),
		// like IcmpSendEcho2, it does not watch the context
		func(context.Context) []note {
			time.Sleep(probeTimeout)
			return single(check.Pass, codeICMPOK)
		},
	})
	if elapsed := time.Since(start); elapsed > probeTimeout/2 {
		t.Errorf("took %s after cancellation", elapsed)
	}
	if res.Status != check.Skip || res.Code != "common.cancelled" {
		t.Errorf("result = %s %s", res.Status, res.Code)
	}
	if len(res.Findings) != 1 || res.Findings[0].Code != codeDNSOK {
		t.Errorf("findings %v, want the one emitted before cancellation", res.Findings)
	}
}

func TestErrParam(t *testing.T) {
	inner := errors.New("dial tcp 13.107.4.52:80: i/o timeout")
	cases := []struct {
		err  error
		want string
	}{
		{inner, inner.Error()},
		{&url.Error{Op: "Get", URL: "http://www.msftconnecttest.com/connecttest.txt", Err: inner}, inner.Error()},
		{&url.Error{Op: "Get", URL: "http://x", Err: context.DeadlineExceeded}, "context deadline exceeded"},
		{fmt.Errorf("IcmpCreateFile: %w", errors.New("access denied")), "IcmpCreateFile: access denied"},
	}
	for _, c := range cases {
		if got := errParam(c.err); got != c.want {
			t.Errorf("errParam(%v) = %q, want %q", c.err, got, c.want)
		}
	}
}
