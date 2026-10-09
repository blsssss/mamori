// Package netcheck checks the internet connection with the probes of the Windows NCSI service plus
// ICMP and TCP, so that a captive portal, broken DNS and no connection are told apart.
package netcheck

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/blsssss/mamori/internal/check"
)

const probeTimeout = 3 * time.Second

// codes the verdict depends on
const (
	codeAdapterNone = "net.adapter.none"
	codeDNSOK       = "net.dns.ok"
	codeICMPOK      = "net.icmp.ok"
	codeICMPLocal   = "net.icmp.local"
	codeTCPOK       = "net.tcp.ok"
	codeHTTPOK      = "net.http.ok"
	codeHTTPCaptive = "net.http.captive"
)

// note is a finding made inside a probe goroutine, the recorder is not safe for concurrent use
type note struct {
	status check.Status
	code   string
	params []string
}

type probe func(context.Context) []note

func run(ctx context.Context, emit check.Emit, probes []probe) check.Result {
	r := check.Start(check.Internet, emit)
	done := make([]chan []note, len(probes))
	for i, p := range probes {
		ch := make(chan []note, 1)
		done[i] = ch
		go func() {
			pctx, cancel := context.WithTimeout(ctx, probeTimeout)
			defer cancel()
			ch <- p(pctx)
		}()
	}
	var codes []string
	// findings keep the probe order and go out as soon as a probe and every probe before it are done
	for _, ch := range done {
		var notes []note
		select {
		case notes = <-ch:
		case <-ctx.Done():
		}
		// a probe stopped by cancellation reports a failure that says nothing about the network, and a
		// probe inside a synchronous system call is not waited for
		if ctx.Err() != nil {
			return r.Finish(check.Skip, "common.cancelled")
		}
		for _, n := range notes {
			r.Add(n.status, n.code, n.params...)
			codes = append(codes, n.code)
		}
	}
	status, code := verdict(codes)
	return r.Finish(status, code)
}

// verdict takes the strongest evidence first: the exact NCSI answer proves the connection whatever
// the other probes say, and any other HTTP answer means a portal on the way
func verdict(codes []string) (check.Status, string) {
	has := func(code string) bool { return slices.Contains(codes, code) }
	// the TUN adapter of a VPN or proxy client that answers ping for public addresses accepts TCP
	// connections itself as well, so neither shows a remote host. A TUN that drops ping goes unnoticed
	reached := has(codeICMPOK) || (has(codeTCPOK) && !has(codeICMPLocal))
	switch {
	case has(codeHTTPOK):
		return check.Pass, "net.verdict.online"
	case has(codeHTTPCaptive):
		return check.Warn, "net.verdict.captive"
	case has(codeAdapterNone) || !reached:
		return check.Fail, "net.verdict.offline"
	case !has(codeDNSOK):
		return check.Warn, "net.verdict.dns_broken"
	default:
		return check.Warn, "net.verdict.limited"
	}
}

// errParam is the text of the "error" param. *url.Error repeats the url, the finding has it already
func errParam(err error) string {
	if ue, ok := errors.AsType[*url.Error](err); ok {
		err = ue.Err
	}
	return err.Error()
}

func single(status check.Status, code string, params ...string) []note {
	return []note{{status, code, params}}
}

func since(start time.Time) string {
	return strconv.FormatInt(time.Since(start).Milliseconds(), 10)
}

func join[T fmt.Stringer](items []T) string {
	s := make([]string, len(items))
	for i, v := range items {
		s[i] = v.String()
	}
	return strings.Join(s, ", ")
}
