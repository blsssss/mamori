package fwprobe

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/blsssss/mamori/internal/check"
)

const (
	rulePrefix     = "mamori-probe-"
	controlTimeout = 4 * time.Second
	// time for the filtering platform to apply a rule change
	settle = 300 * time.Millisecond
)

// hosts of different providers: in some networks one of them is throttled or blocked on the way
var controlTargets = []string{"1.1.1.1:443", "8.8.8.8:443", "77.88.8.8:443"}

type dialFunc func(ctx context.Context, addr string) error

// ruleStore is the part of Windows Firewall the rule test changes, an interface so that the test
// sequence also runs against a fake
type ruleStore interface {
	names(prefix string) ([]string, error)
	addBlock(name, exe, ip, port string) error
	remove(name string) error
	localRulesApply() (bool, error)
}

// connect never reuses a connection: one opened before the rule was added would bypass it
func connect(ctx context.Context, addr string) error {
	var d net.Dialer
	c, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return err
	}
	_ = c.Close()
	return nil
}

func dialWithin(ctx context.Context, dial dialFunc, addr string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return dial(ctx, addr)
}

func control(ctx context.Context, dial dialFunc) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, controlTimeout)
	defer cancel()
	type attempt struct {
		target string
		err    error
	}
	// buffered so that the attempts still running never block once control has returned
	done := make(chan attempt, len(controlTargets))
	for _, t := range controlTargets {
		go func() { done <- attempt{t, dial(ctx, t)} }()
	}
	var errs []string
	for range controlTargets {
		a := <-done
		if a.err == nil {
			return a.target, nil
		}
		errs = append(errs, a.err.Error())
	}
	return strings.Join(controlTargets, ", "), errors.New(strings.Join(errs, "; "))
}

func pause(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return false
	case <-time.After(settle):
		return true
	}
}

func ruleTest(ctx context.Context, fw ruleStore, dial dialFunc, exe string, r *recorder) {
	removeLeftovers(fw, r)
	// when group policy keeps local rules out, the rule is accepted but ignored and a working
	// firewall would look broken; a failed read leaves the decision to the test itself
	if apply, err := fw.localRulesApply(); err == nil && !apply {
		r.add(finding(check.Skip, "fw.rule.gp_override"))
		return
	}
	blockAndRestore(ctx, fw, dial, exe, r)
}

// removeLeftovers deletes the rules of runs that were killed before their own cleanup; one of
// them could block the control connection
func removeLeftovers(fw ruleStore, r *recorder) {
	names, err := fw.names(rulePrefix)
	if err != nil {
		r.add(finding(check.Error, "fw.rule.remove_failed", "name", rulePrefix+"*", "error", err.Error()))
		return
	}
	removed := 0
	for _, name := range names {
		if err := fw.remove(name); err != nil {
			r.add(finding(check.Error, "fw.rule.remove_failed", "name", name, "error", err.Error()))
			continue
		}
		removed++
	}
	if removed > 0 {
		r.add(finding(check.Pass, "fw.rule.cleanup", "count", strconv.Itoa(removed)))
	}
}

// the rule matches only the program exe, so the traffic of everything else on the machine is
// never affected
func blockAndRestore(ctx context.Context, fw ruleStore, dial dialFunc, exe string, r *recorder) {
	target, err := control(ctx, dial)
	if ctx.Err() != nil {
		return
	}
	if err != nil {
		r.add(finding(check.Skip, "fw.rule.control_failed", "target", target, "error", err.Error()))
		return
	}
	ip, port, _ := net.SplitHostPort(target)
	name := fmt.Sprintf("%s%08x", rulePrefix, rand.Uint32())
	if err := fw.addBlock(name, exe, ip, port); err != nil {
		r.add(finding(check.Error, "fw.rule.add_failed", "error", err.Error()))
		return
	}
	r.add(finding(check.Pass, "fw.rule.added", "name", name, "target", target))
	removed := false
	defer func() {
		if !removed {
			removeRule(fw, name, r)
		}
	}()

	if !pause(ctx) {
		return
	}
	start := time.Now()
	err = dialWithin(ctx, dial, target, controlTimeout)
	if ctx.Err() != nil {
		return
	}
	blocked := err != nil
	if blocked {
		r.add(finding(check.Pass, "fw.rule.blocked", "target", target, "error", err.Error(), "ms", ms(time.Since(start))))
	} else {
		r.add(finding(check.Fail, "fw.rule.not_enforced", "target", target))
	}

	removed = true
	if !removeRule(fw, name, r) || !blocked || !pause(ctx) {
		return
	}
	err = dialWithin(ctx, dial, target, controlTimeout)
	switch {
	case ctx.Err() != nil:
	case err == nil:
		r.add(finding(check.Pass, "fw.rule.restored", "target", target))
	default:
		r.add(finding(check.Warn, "fw.rule.not_restored", "target", target, "error", err.Error()))
	}
}

func removeRule(fw ruleStore, name string, r *recorder) bool {
	if err := fw.remove(name); err != nil {
		r.add(finding(check.Error, "fw.rule.remove_failed", "name", name, "error", err.Error()))
		return false
	}
	r.add(finding(check.Pass, "fw.rule.removed", "name", name))
	return true
}
