package netcheck

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/netip"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/blsssss/mamori/internal/check"
)

const ncsiBody = "Microsoft Connect Test"

func lookup(host string) probe {
	return func(ctx context.Context) []note {
		start := time.Now()
		addrs, err := net.DefaultResolver.LookupNetIP(ctx, "ip", host)
		if err != nil {
			return single(check.Fail, "net.dns.fail", "host", host, "error", errParam(err))
		}
		return single(check.Pass, codeDNSOK, "host", host, "addrs", join(unmap(addrs)), "ms", since(start))
	}
}

// ncsiLookup repeats the DNS probe of NCSI: the name has one fixed address, and another answer means
// that something on the way rewrites DNS, as captive portals do
func ncsiLookup(host string, want netip.Addr) probe {
	return func(ctx context.Context) []note {
		addrs, err := net.DefaultResolver.LookupNetIP(ctx, "ip4", host)
		if err != nil {
			return single(check.Fail, "net.ncsi_dns.fail", "host", host, "error", errParam(err))
		}
		addrs = unmap(addrs)
		if slices.Contains(addrs, want) {
			return single(check.Pass, "net.ncsi_dns.ok", "host", host, "addr", want.String())
		}
		return single(check.Warn, "net.ncsi_dns.mismatch", "host", host, "addr", join(addrs), "want", want.String())
	}
}

// LookupNetIP may return ipv4 addresses in the ipv4-mapped ipv6 form
func unmap(addrs []netip.Addr) []netip.Addr {
	for i := range addrs {
		addrs[i] = addrs[i].Unmap()
	}
	return addrs
}

type connector func(ctx context.Context, network, address string) (net.Conn, error)

// dial connects to all targets at the same time and passes with the first connection
func dial(connect connector, targets ...string) probe {
	return func(ctx context.Context) []note {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()
		type result struct {
			i   int
			ms  string
			err error
		}
		results := make(chan result, len(targets))
		start := time.Now()
		for i, target := range targets {
			go func() {
				conn, err := connect(ctx, "tcp", target)
				ms := since(start)
				if err == nil {
					_ = conn.Close()
				}
				results <- result{i, ms, err}
			}()
		}
		errs := make([]string, len(targets))
		for range targets {
			r := <-results
			if r.err == nil {
				return single(check.Pass, codeTCPOK, "target", targets[r.i], "ms", r.ms)
			}
			errs[r.i] = dialError(targets[r.i], r.err)
		}
		return single(check.Fail, "net.tcp.fail", "target", strings.Join(targets, ", "), "error", strings.Join(errs, "; "))
	}
}

// dialError starts with the target, as the errors of ping do
func dialError(target string, err error) string {
	// *net.OpError would repeat the address
	if oe, ok := errors.AsType[*net.OpError](err); ok {
		err = oe.Err
	}
	return target + ": " + err.Error()
}

// fetch repeats the active HTTP probe of NCSI. A captive portal answers the plain HTTP request
// itself, with a redirect to its login page or with that page
func fetch(rawURL string) probe {
	client := &http.Client{
		// a fresh connection on every run, a pooled one would not test the network again
		Transport: &http.Transport{Proxy: http.ProxyFromEnvironment, DisableKeepAlives: true},
		// following the redirect would land on the portal and hide it
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	return func(ctx context.Context) []note {
		fail := func(err error) []note {
			return single(check.Fail, "net.http.fail", "url", rawURL, "error", errParam(err))
		}
		start := time.Now()
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
		if err != nil {
			return fail(err)
		}
		resp, err := client.Do(req)
		if err != nil {
			return fail(err)
		}
		defer resp.Body.Close()
		// enough to compare with the expected body without downloading a whole portal page
		body, err := io.ReadAll(io.LimitReader(resp.Body, 512))
		if err != nil {
			return fail(err)
		}
		switch {
		case resp.StatusCode == http.StatusOK && string(body) == ncsiBody:
			return single(check.Pass, codeHTTPOK, "url", rawURL, "ms", since(start))
		// a portal redirects to its login page, serves that page instead, or answers 511 that RFC 6585
		// made for it. Other errors come from a proxy or the server
		case resp.StatusCode >= 400 && resp.StatusCode != http.StatusNetworkAuthenticationRequired:
			return fail(errors.New("HTTP " + resp.Status))
		}
		return single(check.Warn, codeHTTPCaptive, "url", rawURL,
			"status", strconv.Itoa(resp.StatusCode), "location", resp.Header.Get("Location"))
	}
}

type adapter struct {
	name     string
	up       bool
	loopback bool
	ipv4     []netip.Addr
	gateways []netip.Addr
}

func adapters(list func() ([]adapter, error)) probe {
	return func(context.Context) []note {
		found, err := list()
		if err != nil {
			return single(check.Error, "net.adapter.error", "error", errParam(err))
		}
		return adapterNotes(found)
	}
}

func adapterNotes(list []adapter) []note {
	var notes []note
	for _, a := range list {
		// windows gives itself 169.254.x.x when DHCP did not answer, that is no working network
		usable := slices.DeleteFunc(slices.Clone(a.ipv4), netip.Addr.IsLinkLocalUnicast)
		if !a.up || a.loopback || len(usable) == 0 {
			continue
		}
		notes = append(notes, note{check.Pass, "net.adapter.ok",
			[]string{"name", a.name, "ipv4", join(usable), "gateway", join(a.gateways)}})
	}
	if len(notes) == 0 {
		return single(check.Fail, codeAdapterNone)
	}
	return notes
}

type sender func(target netip.Addr, timeout time.Duration) (echoReply, error)

// ping sends to all targets at the same time, each with the whole probe time, and passes with the
// first reply from a remote host. The targets are public addresses, see echoReply.local
func ping(send sender, targets ...netip.Addr) probe {
	return func(ctx context.Context) []note {
		if err := ctx.Err(); err != nil {
			return single(check.Warn, "net.icmp.fail", "target", join(targets), "error", err.Error())
		}
		deadline, ok := ctx.Deadline()
		if !ok {
			deadline = time.Now().Add(probeTimeout)
		}
		timeout := time.Until(deadline)
		type result struct {
			i     int
			reply echoReply
			err   error
		}
		// buffered: the sends are synchronous and finish after a remote reply has ended the probe
		results := make(chan result, len(targets))
		for i, target := range targets {
			go func() {
				reply, err := send(target, timeout)
				if err == nil {
					err = reply.err(target)
				}
				results <- result{i, reply, err}
			}()
		}
		errs := make([]string, len(targets))
		var local *result
		for range targets {
			r := <-results
			switch {
			case r.err != nil:
				errs[r.i] = targets[r.i].String() + ": " + errParam(r.err)
			case r.reply.local():
				if local == nil {
					local = &r
				}
			default:
				return single(check.Pass, codeICMPOK, replyParams(targets[r.i], r.reply)...)
			}
		}
		if local != nil {
			return single(check.Warn, codeICMPLocal, replyParams(targets[local.i], local.reply)...)
		}
		return single(check.Warn, "net.icmp.fail", "target", join(targets), "error", strings.Join(errs, "; "))
	}
}

func replyParams(target netip.Addr, r echoReply) []string {
	return []string{"target", target.String(), "rtt_ms", strconv.FormatUint(uint64(r.rtt), 10), "ttl", strconv.Itoa(int(r.ttl))}
}
