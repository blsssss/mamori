package fwprobe

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/blsssss/mamori/internal/check"
)

// winsock codes, kept as numbers so that the classification is tested on any OS
const (
	wsaeacces    syscall.Errno = 10013
	wsaetimedout syscall.Errno = 10060
)

const (
	policyTimeout = 5 * time.Second
	reasonDenied  = "wsaeacces"
	reasonTimeout = "timeout"
)

// no proxy and no redirects: the answer has to come from the target itself
var httpClient = &http.Client{
	Transport:     &http.Transport{DisableKeepAlives: true},
	CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
}

type target struct {
	addr string // host:port of the TCP attempt
	url  string // fetched after the TCP attempt, empty for plain hosts
}

// parseTarget accepts a host, host:port or an http(s) URL; a bare host is tried on port 443
func parseTarget(s string) (target, error) {
	if strings.Contains(s, "://") {
		return parseURL(s)
	}
	host, port, err := net.SplitHostPort(s)
	if err != nil {
		host, port = strings.Trim(s, "[]"), "443"
	}
	if host == "" {
		return target{}, fmt.Errorf("no host in %q", s)
	}
	return target{addr: net.JoinHostPort(host, port)}, nil
}

func parseURL(s string) (target, error) {
	u, err := url.Parse(s)
	if err != nil {
		return target{}, err
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return target{}, fmt.Errorf("unsupported scheme %q", u.Scheme)
	}
	if u.Hostname() == "" {
		return target{}, fmt.Errorf("no host in %q", s)
	}
	port := u.Port()
	if port == "" {
		port = "443"
		if u.Scheme == "http" {
			port = "80"
		}
	}
	return target{addr: net.JoinHostPort(u.Hostname(), port), url: s}, nil
}

func policyTest(ctx context.Context, targets []string, dial dialFunc) []check.Finding {
	var list []string
	for _, t := range targets {
		if t = strings.TrimSpace(t); t != "" {
			list = append(list, t)
		}
	}
	if len(list) == 0 {
		return []check.Finding{finding(check.Skip, "fw.policy.none")}
	}
	networkUp := sync.OnceValue(func() bool {
		_, err := control(ctx, dial)
		return err == nil
	})
	out := make([]check.Finding, len(list))
	var wg sync.WaitGroup
	for i, raw := range list {
		wg.Go(func() { out[i] = probeTarget(ctx, raw, dial, networkUp) })
	}
	wg.Wait()
	return out
}

func probeTarget(ctx context.Context, raw string, dial dialFunc, networkUp func() bool) check.Finding {
	ctx, cancel := context.WithTimeout(ctx, policyTimeout)
	defer cancel()
	start := time.Now()
	t, err := parseTarget(raw)
	connected := false
	if err == nil {
		connected, err = reach(ctx, t, dial)
	}
	return policyFinding(raw, connected, err, time.Since(start), networkUp)
}

// reach reports connected as soon as the TCP handshake succeeds, also when the HTTP request after
// it fails
func reach(ctx context.Context, t target, dial dialFunc) (connected bool, err error) {
	if err := dial(ctx, t.addr); err != nil {
		return false, err
	}
	if t.url == "" {
		return true, nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, t.url, nil)
	if err != nil {
		return true, err
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return true, err
	}
	_ = resp.Body.Close()
	return true, nil
}

// only a refusal by the local filtering platform (WSAEACCES) or a dropped handshake while the
// network works is evidence of filtering: a refused or reset connection may come from the remote
// side or anything in between, and after a completed handshake the firewall has already let the
// flow out, so a later HTTP failure proves nothing either way
func policyFinding(raw string, connected bool, err error, took time.Duration, networkUp func() bool) check.Finding {
	if err == nil {
		return finding(check.Fail, "fw.policy.reachable", "target", raw, "ms", ms(took))
	}
	if !connected {
		switch {
		case errors.Is(err, wsaeacces):
			return finding(check.Pass, "fw.policy.blocked", "target", raw, "error", err.Error(), "reason", reasonDenied)
		case timedOut(err) && networkUp():
			return finding(check.Pass, "fw.policy.blocked", "target", raw, "error", err.Error(), "reason", reasonTimeout)
		}
	}
	return finding(check.Warn, "fw.policy.inconclusive", "target", raw, "error", err.Error())
}

func timedOut(err error) bool {
	// a slow resolver says nothing about the firewall
	var dns *net.DNSError
	if errors.As(err, &dns) {
		return false
	}
	// syscall.Errno.Timeout does not cover the winsock code
	if errors.Is(err, wsaetimedout) {
		return true
	}
	var ne net.Error
	return errors.As(err, &ne) && ne.Timeout()
}
