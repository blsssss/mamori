package netcheck

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/blsssss/mamori/internal/check"
)

func one(t *testing.T, notes []note) note {
	t.Helper()
	if len(notes) != 1 {
		t.Fatalf("want one note, got %v", notes)
	}
	return notes[0]
}

func params(n note) map[string]string {
	m := map[string]string{}
	for i := 0; i+1 < len(n.params); i += 2 {
		m[n.params[i]] = n.params[i+1]
	}
	return m
}

func withTimeout(t *testing.T, d time.Duration) context.Context {
	ctx, cancel := context.WithTimeout(context.Background(), d)
	t.Cleanup(cancel)
	return ctx
}

func TestFetch(t *testing.T) {
	var portalVisited atomic.Bool
	mux := http.NewServeMux()
	mux.HandleFunc("/ok", func(w http.ResponseWriter, _ *http.Request) { fmt.Fprint(w, ncsiBody) })
	mux.HandleFunc("/newline", func(w http.ResponseWriter, _ *http.Request) { fmt.Fprint(w, ncsiBody+"\n") })
	mux.HandleFunc("/redirect", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/portal", http.StatusFound)
	})
	mux.HandleFunc("/portal", func(w http.ResponseWriter, _ *http.Request) {
		portalVisited.Store(true)
		fmt.Fprint(w, ncsiBody)
	})
	mux.HandleFunc("/login-page", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, "<html>"+strings.Repeat("sign in to the guest wifi ", 100)+"</html>")
	})
	mux.HandleFunc("/no-content", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	mux.HandleFunc("/network-auth", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "log in to the guest wifi", http.StatusNetworkAuthenticationRequired)
	})
	mux.HandleFunc("/proxy-auth", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, ncsiBody, http.StatusProxyAuthRequired)
	})
	mux.HandleFunc("/bad-gateway", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, ncsiBody, http.StatusBadGateway)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	cases := []struct {
		path   string
		status check.Status
		code   string
		params map[string]string
	}{
		{"/ok", check.Pass, codeHTTPOK, nil},
		{"/redirect", check.Warn, codeHTTPCaptive, map[string]string{"status": "302", "location": "/portal"}},
		{"/login-page", check.Warn, codeHTTPCaptive, map[string]string{"status": "200", "location": ""}},
		{"/newline", check.Warn, codeHTTPCaptive, map[string]string{"status": "200", "location": ""}},
		{"/no-content", check.Warn, codeHTTPCaptive, map[string]string{"status": "204", "location": ""}},
		{"/network-auth", check.Warn, codeHTTPCaptive, map[string]string{"status": "511", "location": ""}},
		{"/proxy-auth", check.Fail, "net.http.fail", map[string]string{"error": "HTTP 407 Proxy Authentication Required"}},
		{"/bad-gateway", check.Fail, "net.http.fail", map[string]string{"error": "HTTP 502 Bad Gateway"}},
	}
	for _, c := range cases {
		t.Run(c.path, func(t *testing.T) {
			n := one(t, fetch(srv.URL+c.path)(withTimeout(t, 2*time.Second)))
			p := params(n)
			if n.status != c.status || n.code != c.code {
				t.Fatalf("got %s %s %v", n.status, n.code, p)
			}
			if p["url"] != srv.URL+c.path {
				t.Errorf("url = %q", p["url"])
			}
			for k, v := range c.params {
				if got, ok := p[k]; !ok || got != v {
					t.Errorf("%s = %q, want %q", k, got, v)
				}
			}
			if c.code == codeHTTPOK && p["ms"] == "" {
				t.Error("no ms")
			}
		})
	}
	if portalVisited.Load() {
		t.Error("the redirect was followed")
	}
}

func TestFetchFails(t *testing.T) {
	hang := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer hang.Close()
	closed := httptest.NewServer(http.NotFoundHandler())
	closed.Close()

	for _, u := range []string{hang.URL, closed.URL, "http://bad host/"} {
		n := one(t, fetch(u)(withTimeout(t, 300*time.Millisecond)))
		p := params(n)
		if n.status != check.Fail || n.code != "net.http.fail" || p["url"] != u || p["error"] == "" {
			t.Errorf("%s: got %s %s %v", u, n.status, n.code, p)
		}
		if strings.Contains(p["error"], u) {
			t.Errorf("%s: error repeats the url: %q", u, p["error"])
		}
	}
}

func closedPort(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	_ = ln.Close()
	return addr
}

func TestDial(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	open := ln.Addr().String()
	var d net.Dialer
	n := one(t, dial(d.DialContext, open)(withTimeout(t, 2*time.Second)))
	if n.status != check.Pass || n.code != codeTCPOK || params(n)["target"] != open || params(n)["ms"] == "" {
		t.Errorf("open port: %s %s %v", n.status, n.code, params(n))
	}

	closed := closedPort(t)
	n = one(t, dial(d.DialContext, closed)(withTimeout(t, 300*time.Millisecond)))
	if n.status != check.Fail || n.code != "net.tcp.fail" || !strings.HasPrefix(params(n)["error"], closed+": ") {
		t.Errorf("closed port: %s %s %v", n.status, n.code, params(n))
	}
}

func TestDialSeveral(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	open, closed := ln.Addr().String(), closedPort(t)
	const dropped = "192.0.2.1:443"
	var d net.Dialer
	// an address whose packets are dropped on the way answers nothing until the probe gives up
	connect := func(ctx context.Context, network, address string) (net.Conn, error) {
		if address == dropped {
			<-ctx.Done()
			return nil, ctx.Err()
		}
		return d.DialContext(ctx, network, address)
	}

	start := time.Now()
	n := one(t, dial(connect, dropped, closed, open)(withTimeout(t, 2*time.Second)))
	if n.status != check.Pass || n.code != codeTCPOK || params(n)["target"] != open {
		t.Errorf("one open port: %s %s %v", n.status, n.code, params(n))
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Errorf("took %s, the targets were not dialed at the same time", elapsed)
	}

	n = one(t, dial(connect, dropped, closed)(withTimeout(t, 300*time.Millisecond)))
	p := params(n)
	if n.status != check.Fail || n.code != "net.tcp.fail" || p["target"] != dropped+", "+closed {
		t.Fatalf("no open port: %s %s %v", n.status, n.code, p)
	}
	errs := strings.Split(p["error"], "; ")
	if len(errs) != 2 || errs[0] != dropped+": context deadline exceeded" || !strings.HasPrefix(errs[1], closed+": ") {
		t.Errorf("error = %q, want one per target in order", p["error"])
	}
	if strings.Count(p["error"], closed) != 1 {
		t.Errorf("error repeats the target: %q", p["error"])
	}
}

func TestLookup(t *testing.T) {
	n := one(t, lookup("localhost")(withTimeout(t, 2*time.Second)))
	p := params(n)
	if n.status != check.Pass || n.code != codeDNSOK || p["host"] != "localhost" || p["ms"] == "" {
		t.Fatalf("got %s %s %v", n.status, n.code, p)
	}
	if !strings.Contains(p["addrs"], "127.0.0.1") && !strings.Contains(p["addrs"], "::1") {
		t.Errorf("addrs = %q", p["addrs"])
	}

	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	n = one(t, lookup("localhost")(cancelled))
	if n.status != check.Fail || n.code != "net.dns.fail" || params(n)["error"] == "" {
		t.Errorf("cancelled: %s %s %v", n.status, n.code, params(n))
	}
}

func TestNCSILookup(t *testing.T) {
	loopback := netip.MustParseAddr("127.0.0.1")
	n := one(t, ncsiLookup("localhost", loopback)(withTimeout(t, 2*time.Second)))
	if n.status != check.Pass || n.code != "net.ncsi_dns.ok" || params(n)["addr"] != "127.0.0.1" {
		t.Errorf("match: %s %s %v", n.status, n.code, params(n))
	}

	n = one(t, ncsiLookup("localhost", netip.MustParseAddr("131.107.255.255"))(withTimeout(t, 2*time.Second)))
	p := params(n)
	if n.status != check.Warn || n.code != "net.ncsi_dns.mismatch" || p["addr"] != "127.0.0.1" || p["want"] != "131.107.255.255" {
		t.Errorf("mismatch: %s %s %v", n.status, n.code, p)
	}

	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	n = one(t, ncsiLookup("localhost", loopback)(cancelled))
	if n.status != check.Fail || n.code != "net.ncsi_dns.fail" || params(n)["error"] == "" {
		t.Errorf("cancelled: %s %s %v", n.status, n.code, params(n))
	}
}

func TestUnmap(t *testing.T) {
	got := unmap([]netip.Addr{netip.MustParseAddr("::ffff:131.107.255.255"), netip.MustParseAddr("::1")})
	if got[0] != netip.MustParseAddr("131.107.255.255") || got[1] != netip.MustParseAddr("::1") {
		t.Errorf("unmap = %v", got)
	}
}

func addrs(s ...string) []netip.Addr {
	out := make([]netip.Addr, len(s))
	for i, a := range s {
		out[i] = netip.MustParseAddr(a)
	}
	return out
}

func TestAdapterNotes(t *testing.T) {
	cases := []struct {
		name string
		list []adapter
		want []note
	}{
		{"none at all", nil, single(check.Fail, "net.adapter.none")},
		{"ethernet with gateway", []adapter{
			{name: "Ethernet", up: true, ipv4: addrs("192.168.1.10"), gateways: addrs("192.168.1.1")},
		}, single(check.Pass, "net.adapter.ok", "name", "Ethernet", "ipv4", "192.168.1.10", "gateway", "192.168.1.1")},
		{"loopback, down and dhcp failure are skipped", []adapter{
			{name: "Loopback Pseudo-Interface 1", up: true, loopback: true, ipv4: addrs("127.0.0.1")},
			{name: "Wi-Fi", up: false, ipv4: addrs("192.168.0.5"), gateways: addrs("192.168.0.1")},
			{name: "Ethernet 2", up: true, ipv4: addrs("169.254.10.20")},
			{name: "Bluetooth", up: true},
		}, single(check.Fail, "net.adapter.none")},
		{"several adapters, host-only without gateway", []adapter{
			{name: "Ethernet", up: true, ipv4: addrs("169.254.3.4", "10.0.0.7", "10.0.0.8"), gateways: addrs("10.0.0.1")},
			{name: "VirtualBox Host-Only Network", up: true, ipv4: addrs("192.168.56.1")},
		}, []note{
			{check.Pass, "net.adapter.ok", []string{"name", "Ethernet", "ipv4", "10.0.0.7, 10.0.0.8", "gateway", "10.0.0.1"}},
			{check.Pass, "net.adapter.ok", []string{"name", "VirtualBox Host-Only Network", "ipv4", "192.168.56.1", "gateway", ""}},
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := adapterNotes(c.list)
			if len(got) != len(c.want) {
				t.Fatalf("got %v, want %v", got, c.want)
			}
			for i := range got {
				if got[i].status != c.want[i].status || got[i].code != c.want[i].code ||
					strings.Join(got[i].params, "|") != strings.Join(c.want[i].params, "|") {
					t.Errorf("note %d = %v, want %v", i, got[i], c.want[i])
				}
			}
		})
	}
	if len(cases[3].list[0].ipv4) != 3 {
		t.Error("adapterNotes changed its input")
	}
}

func TestAdaptersProbe(t *testing.T) {
	failing := func() ([]adapter, error) { return nil, errors.New("GetAdaptersAddresses: Access is denied.") }
	n := one(t, adapters(failing)(context.Background()))
	if n.status != check.Error || n.code != "net.adapter.error" || params(n)["error"] != "GetAdaptersAddresses: Access is denied." {
		t.Errorf("api failure: %s %s %v", n.status, n.code, params(n))
	}
	empty := func() ([]adapter, error) { return nil, nil }
	n = one(t, adapters(empty)(context.Background()))
	if n.status != check.Fail || n.code != codeAdapterNone {
		t.Errorf("no adapters: %s %s %v", n.status, n.code, params(n))
	}
}

type echoAnswer struct {
	reply echoReply
	err   error
	delay time.Duration
}

type fakeNet struct {
	answers map[netip.Addr]echoAnswer
	mu      sync.Mutex
	sent    map[netip.Addr]time.Duration
}

func (f *fakeNet) send(target netip.Addr, timeout time.Duration) (echoReply, error) {
	f.mu.Lock()
	f.sent[target] = timeout
	f.mu.Unlock()
	a := f.answers[target]
	time.Sleep(a.delay)
	return a.reply, a.err
}

func TestPing(t *testing.T) {
	cf, google, yandex := netip.MustParseAddr("1.1.1.1"), netip.MustParseAddr("8.8.8.8"), netip.MustParseAddr("77.88.8.8")
	router := netip.MustParseAddr("192.168.0.1")
	timeout := echoAnswer{err: ipStatus(11010)}
	remote := func(addr netip.Addr, rtt uint32, ttl uint8, delay time.Duration) echoAnswer {
		return echoAnswer{reply: echoReply{addr: addr, rtt: rtt, ttl: ttl}, delay: delay}
	}
	cases := []struct {
		name    string
		answers map[netip.Addr]echoAnswer
		status  check.Status
		code    string
		params  map[string]string
	}{
		{"one target answers, the others time out",
			map[netip.Addr]echoAnswer{cf: timeout, google: remote(google, 31, 116, 0), yandex: timeout},
			check.Pass, codeICMPOK, map[string]string{"target": "8.8.8.8", "rtt_ms": "31", "ttl": "116"}},
		{"the fastest reply wins",
			map[netip.Addr]echoAnswer{cf: remote(cf, 80, 57, 80*time.Millisecond), google: timeout, yandex: remote(yandex, 9, 58, 0)},
			check.Pass, codeICMPOK, map[string]string{"target": "77.88.8.8", "rtt_ms": "9", "ttl": "58"}},
		{"a local reply does not hide a remote one",
			map[netip.Addr]echoAnswer{cf: remote(cf, 0, 255, 0), google: remote(google, 31, 116, 30*time.Millisecond), yandex: timeout},
			check.Pass, codeICMPOK, map[string]string{"target": "8.8.8.8", "rtt_ms": "31", "ttl": "116"}},
		{"a tun answers for every address",
			map[netip.Addr]echoAnswer{cf: remote(cf, 0, 255, 0), google: remote(google, 0, 255, 0), yandex: remote(yandex, 0, 255, 0)},
			check.Warn, codeICMPLocal, map[string]string{"rtt_ms": "0", "ttl": "255"}},
		{"router reports unreachable, the others time out",
			map[netip.Addr]echoAnswer{cf: {reply: echoReply{addr: router, status: 11003, ttl: 64}}, google: timeout, yandex: timeout},
			check.Warn, "net.icmp.fail", map[string]string{
				"target": "1.1.1.1, 8.8.8.8, 77.88.8.8",
				"error":  "1.1.1.1: IP_DEST_HOST_UNREACHABLE (11003) from 192.168.0.1; 8.8.8.8: IP_REQ_TIMED_OUT (11010); 77.88.8.8: IP_REQ_TIMED_OUT (11010)",
			}},
		{"a success from another address is no answer",
			map[netip.Addr]echoAnswer{cf: {}, google: timeout, yandex: timeout},
			check.Warn, "net.icmp.fail", map[string]string{
				"error": "1.1.1.1: echo reply from invalid IP; 8.8.8.8: IP_REQ_TIMED_OUT (11010); 77.88.8.8: IP_REQ_TIMED_OUT (11010)",
			}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := &fakeNet{answers: c.answers, sent: map[netip.Addr]time.Duration{}}
			n := one(t, ping(f.send, cf, google, yandex)(withTimeout(t, probeTimeout)))
			p := params(n)
			if n.status != c.status || n.code != c.code {
				t.Fatalf("got %s %s %v", n.status, n.code, p)
			}
			for k, v := range c.params {
				if p[k] != v {
					t.Errorf("%s = %q, want %q", k, p[k], v)
				}
			}
		})
	}
}

func TestPingGivesEveryTargetTheWholeTime(t *testing.T) {
	targets := []netip.Addr{netip.MustParseAddr("1.1.1.1"), netip.MustParseAddr("8.8.8.8"), netip.MustParseAddr("77.88.8.8")}
	f := &fakeNet{answers: map[netip.Addr]echoAnswer{}, sent: map[netip.Addr]time.Duration{}}
	for _, a := range targets {
		f.answers[a] = echoAnswer{err: ipStatus(11010), delay: 50 * time.Millisecond}
	}
	start := time.Now()
	n := one(t, ping(f.send, targets...)(withTimeout(t, probeTimeout)))
	if n.code != "net.icmp.fail" {
		t.Fatalf("got %s %s %v", n.status, n.code, params(n))
	}
	if elapsed := time.Since(start); elapsed > 140*time.Millisecond {
		t.Errorf("took %s, the targets were not pinged at the same time", elapsed)
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, a := range targets {
		if d, ok := f.sent[a]; !ok || d > probeTimeout || d < probeTimeout-200*time.Millisecond {
			t.Errorf("%s: timeout %s, sent %t, want about %s", a, d, ok, probeTimeout)
		}
	}
}

func TestPingCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	f := &fakeNet{sent: map[netip.Addr]time.Duration{}}
	n := one(t, ping(f.send, netip.MustParseAddr("1.1.1.1"))(ctx))
	if n.code != "net.icmp.fail" || params(n)["error"] != context.Canceled.Error() || len(f.sent) != 0 {
		t.Errorf("got %s %s %v, sent %v", n.status, n.code, params(n), f.sent)
	}
}
