package fwprobe

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"slices"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/blsssss/mamori/internal/check"
)

func TestParseTarget(t *testing.T) {
	cases := []struct {
		in, addr, url string
	}{
		{"example.com", "example.com:443", ""},
		{"example.com:8080", "example.com:8080", ""},
		{"10.0.0.1", "10.0.0.1:443", ""},
		{"2001:db8::1", "[2001:db8::1]:443", ""},
		{"[2001:db8::1]", "[2001:db8::1]:443", ""},
		{"[2001:db8::1]:80", "[2001:db8::1]:80", ""},
		{"http://example.com/blocked", "example.com:80", "http://example.com/blocked"},
		{"https://example.com", "example.com:443", "https://example.com"},
		{"HTTPS://example.com:8443/", "example.com:8443", "HTTPS://example.com:8443/"},
		{"http://[2001:db8::1]/", "[2001:db8::1]:80", "http://[2001:db8::1]/"},
	}
	for _, c := range cases {
		got, err := parseTarget(c.in)
		if err != nil {
			t.Errorf("parseTarget(%q): %v", c.in, err)
			continue
		}
		if got.addr != c.addr || got.url != c.url {
			t.Errorf("parseTarget(%q) = %+v, want addr %q url %q", c.in, got, c.addr, c.url)
		}
	}
	for _, in := range []string{"ftp://example.com", "http://", ":443", "http://[::1"} {
		if got, err := parseTarget(in); err == nil {
			t.Errorf("parseTarget(%q) = %+v, want an error", in, got)
		}
	}
}

func dialErr(errno syscall.Errno) error {
	return &net.OpError{Op: "dial", Net: "tcp", Err: os.NewSyscallError("connectex", errno)}
}

// a real timeout from the standard library, without touching the network
func expiredDial(t *testing.T) error {
	ctx, cancel := context.WithDeadline(context.Background(), time.Now())
	defer cancel()
	var d net.Dialer
	c, err := d.DialContext(ctx, "tcp", "192.0.2.1:443")
	if err == nil {
		_ = c.Close()
		t.Fatal("dial with an expired deadline succeeded")
	}
	return err
}

func TestPolicyFinding(t *testing.T) {
	httpTimeout := &url.Error{Op: "Get", URL: "https://example.com", Err: context.DeadlineExceeded}
	httpReset := &url.Error{Op: "Get", URL: "https://example.com", Err: &net.OpError{Op: "read", Net: "tcp", Err: os.NewSyscallError("wsarecv", syscall.Errno(10054))}}
	cases := []struct {
		name      string
		connected bool
		err       error
		networkUp bool
		status    check.Status
		code      string
		reason    string
		asked     bool
	}{
		{"answered", true, nil, true, check.Fail, "fw.policy.reachable", "", false},
		{"wsaeacces", false, dialErr(wsaeacces), false, check.Pass, "fw.policy.blocked", reasonDenied, false},
		{"timeout, network up", false, expiredDial(t), true, check.Pass, "fw.policy.blocked", reasonTimeout, true},
		{"timeout, network down", false, expiredDial(t), false, check.Warn, "fw.policy.inconclusive", "", true},
		{"deadline in op error", false, &net.OpError{Op: "dial", Err: context.DeadlineExceeded}, true, check.Pass, "fw.policy.blocked", reasonTimeout, true},
		{"wsaetimedout", false, dialErr(wsaetimedout), true, check.Pass, "fw.policy.blocked", reasonTimeout, true},
		{"connect ok, http timeout", true, httpTimeout, true, check.Warn, "fw.policy.inconclusive", "", false},
		{"connect ok, http reset", true, httpReset, true, check.Warn, "fw.policy.inconclusive", "", false},
		{"refused", false, dialErr(10061), true, check.Warn, "fw.policy.inconclusive", "", false},
		{"reset", false, dialErr(10054), true, check.Warn, "fw.policy.inconclusive", "", false},
		{"unreachable", false, dialErr(10065), true, check.Warn, "fw.policy.inconclusive", "", false},
		{"no such host", false, &net.OpError{Op: "dial", Err: &net.DNSError{Err: "no such host", Name: "x", IsNotFound: true}}, true, check.Warn, "fw.policy.inconclusive", "", false},
		{"dns timeout", false, &net.OpError{Op: "dial", Err: &net.DNSError{Err: "i/o timeout", Name: "x", IsTimeout: true}}, true, check.Warn, "fw.policy.inconclusive", "", false},
		{"cancelled", false, &net.OpError{Op: "dial", Err: context.Canceled}, true, check.Warn, "fw.policy.inconclusive", "", false},
		{"bad target", false, errors.New(`unsupported scheme "ftp"`), true, check.Warn, "fw.policy.inconclusive", "", false},
	}
	for _, c := range cases {
		asked := false
		networkUp := func() bool {
			asked = true
			return c.networkUp
		}
		f := policyFinding("example.com", c.connected, c.err, 1500*time.Millisecond, networkUp)
		if f.Status != c.status || f.Code != c.code || f.Params["reason"] != c.reason {
			t.Errorf("%s: got %s %s reason %q, want %s %s reason %q", c.name, f.Status, f.Code, f.Params["reason"], c.status, c.code, c.reason)
		}
		if asked != c.asked {
			t.Errorf("%s: control connection asked %v, want %v", c.name, asked, c.asked)
		}
		if f.Params["target"] != "example.com" {
			t.Errorf("%s: target param %q", c.name, f.Params["target"])
		}
	}
	if f := policyFinding("example.com", true, nil, 1500*time.Millisecond, nil); f.Params["ms"] != "1500" {
		t.Errorf("reachable ms param = %q, want 1500", f.Params["ms"])
	}
	if f := policyFinding("example.com", false, dialErr(wsaeacces), 0, nil); f.Params["error"] == "" {
		t.Error("blocked finding has no error param")
	}
}

func serve(t *testing.T, handle func(net.Conn)) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go handle(c)
		}
	}()
	return ln.Addr().String()
}

func TestReach(t *testing.T) {
	// accepts and never answers, like a path that drops everything after the handshake
	silent := serve(t, func(c net.Conn) {
		_, _ = io.Copy(io.Discard, c)
		_ = c.Close()
	})
	// zero linger turns the close into a reset
	reset := serve(t, func(c net.Conn) {
		_ = c.(*net.TCPConn).SetLinger(0)
		_ = c.Close()
	})
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	t.Cleanup(srv.Close)
	denied := func(context.Context, string) error { return dialErr(wsaeacces) }

	cases := []struct {
		name      string
		target    target
		dial      dialFunc
		connected bool
		code      string
	}{
		{"tcp answers", target{addr: silent}, connect, true, "fw.policy.reachable"},
		{"http answers", target{addr: srv.Listener.Addr().String(), url: srv.URL}, connect, true, "fw.policy.reachable"},
		{"connect ok, http timeout", target{addr: silent, url: "http://" + silent + "/"}, connect, true, "fw.policy.inconclusive"},
		{"connect ok, http reset", target{addr: reset, url: "http://" + reset + "/"}, connect, true, "fw.policy.inconclusive"},
		{"connect denied", target{addr: silent, url: "http://" + silent + "/"}, denied, false, "fw.policy.blocked"},
	}
	for _, c := range cases {
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		connected, err := reach(ctx, c.target, c.dial)
		cancel()
		if connected != c.connected {
			t.Errorf("%s: connected = %v (%v), want %v", c.name, connected, err, c.connected)
		}
		networkUp := func() bool {
			t.Errorf("%s: a failure after the handshake must not ask the control hosts", c.name)
			return true
		}
		if f := policyFinding("x", connected, err, 0, networkUp); f.Code != c.code {
			t.Errorf("%s: %s (%v), want %s", c.name, f.Code, err, c.code)
		}
	}
}

func TestPolicyTest(t *testing.T) {
	if got := policyTest(context.Background(), []string{" ", ""}, nil); len(got) != 1 || got[0].Code != "fw.policy.none" {
		t.Errorf("empty list: %v", got)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	t.Cleanup(srv.Close)
	var controlRounds atomic.Int32
	dial := func(ctx context.Context, addr string) error {
		switch {
		case addr == controlTargets[0]:
			controlRounds.Add(1)
			return nil
		case slices.Contains(controlTargets, addr), strings.HasPrefix(addr, "dropped"):
			return dialErr(wsaetimedout)
		case strings.HasPrefix(addr, "denied"):
			return dialErr(wsaeacces)
		}
		return connect(ctx, addr)
	}
	targets := []string{srv.URL, " dropped-a.test ", "dropped-b.test:8443", "denied.test", "ftp://example.com"}
	want := []struct{ code, reason string }{
		{"fw.policy.reachable", ""},
		{"fw.policy.blocked", reasonTimeout},
		{"fw.policy.blocked", reasonTimeout},
		{"fw.policy.blocked", reasonDenied},
		{"fw.policy.inconclusive", ""},
	}
	got := policyTest(context.Background(), targets, dial)
	if len(got) != len(want) {
		t.Fatalf("got %d findings, want %d: %v", len(got), len(want), got)
	}
	for i, w := range want {
		if got[i].Code != w.code || got[i].Params["reason"] != w.reason {
			t.Errorf("%q: got %s reason %q, want %s reason %q", targets[i], got[i].Code, got[i].Params["reason"], w.code, w.reason)
		}
	}
	// the winning control dial finishes before control returns, so this count is exact
	if n := controlRounds.Load(); n != 1 {
		t.Errorf("two dropped targets ran %d control rounds, want 1", n)
	}
}
