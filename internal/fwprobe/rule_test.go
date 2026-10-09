package fwprobe

import (
	"context"
	"errors"
	"net"
	"reflect"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/blsssss/mamori/internal/check"
)

// fakeFirewall plays both the rule store and the network: a dial to a host fails while an
// enforced rule blocks it
type fakeFirewall struct {
	mu              sync.Mutex
	up              map[string]bool
	enforce         bool
	gpOverride      bool
	leftovers       []string
	addErr          error
	removeErr       error
	downAfterRemove bool
	onAdd           func()

	exe     string
	rules   map[string]string
	blocks  map[string]string
	removes map[string]int
	dials   map[string]int
}

func (f *fakeFirewall) names(string) ([]string, error) { return f.leftovers, nil }

func (f *fakeFirewall) addBlock(name, exe, ip, port string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.addErr != nil {
		return f.addErr
	}
	f.exe = exe
	f.rules[name] = net.JoinHostPort(ip, port)
	f.blocks[name] = f.rules[name]
	if f.onAdd != nil {
		f.onAdd()
	}
	return nil
}

func (f *fakeFirewall) remove(name string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.removes[name]++
	if f.removeErr != nil {
		return f.removeErr
	}
	delete(f.rules, name)
	if f.downAfterRemove {
		f.up = nil
	}
	return nil
}

func (f *fakeFirewall) localRulesApply() (bool, error) { return !f.gpOverride, nil }

func (f *fakeFirewall) dial(ctx context.Context, addr string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.dials[addr]++
	if !f.up[addr] {
		return dialErr(wsaetimedout)
	}
	for _, blocked := range f.rules {
		if f.enforce && blocked == addr {
			return dialErr(wsaeacces)
		}
	}
	return ctx.Err()
}

func TestRuleTest(t *testing.T) {
	const exe = `C:\Program Files\mamori\mamori.exe`
	const leftover = "mamori-probe-0badf00d"
	cases := []struct {
		name            string
		up              []string
		enforce         bool
		gpOverride      bool
		leftovers       []string
		addErr          error
		removeErr       error
		downAfterRemove bool
		cancelOnAdd     bool
		want            []string
		// connections to the chosen control host after the rule was added
		checks int
	}{
		{
			name: "blocked and restored", up: []string{"77.88.8.8:443"}, enforce: true,
			want:   []string{"fw.rule.added", "fw.rule.blocked", "fw.rule.removed", "fw.rule.restored"},
			checks: 2,
		},
		{
			name: "every control host up", up: controlTargets, enforce: true,
			want:   []string{"fw.rule.added", "fw.rule.blocked", "fw.rule.removed", "fw.rule.restored"},
			checks: 2,
		},
		{
			name: "leftovers removed first", up: []string{"77.88.8.8:443"}, enforce: true, leftovers: []string{leftover},
			want:   []string{"fw.rule.cleanup", "fw.rule.added", "fw.rule.blocked", "fw.rule.removed", "fw.rule.restored"},
			checks: 2,
		},
		{
			name: "not enforced", up: []string{"77.88.8.8:443"},
			want:   []string{"fw.rule.added", "fw.rule.not_enforced", "fw.rule.removed"},
			checks: 1,
		},
		{
			name: "network gone after removal", up: []string{"77.88.8.8:443"}, enforce: true, downAfterRemove: true,
			want:   []string{"fw.rule.added", "fw.rule.blocked", "fw.rule.removed", "fw.rule.not_restored"},
			checks: 2,
		},
		{
			name: "removal fails", up: []string{"77.88.8.8:443"}, enforce: true, removeErr: errors.New("access denied"),
			want:   []string{"fw.rule.added", "fw.rule.blocked", "fw.rule.remove_failed"},
			checks: 1,
		},
		{
			name: "cancelled while the rule settles", up: []string{"77.88.8.8:443"}, enforce: true, cancelOnAdd: true,
			want: []string{"fw.rule.added", "fw.rule.removed"},
		},
		{
			name: "add fails", up: []string{"77.88.8.8:443"}, addErr: errors.New("access denied"),
			want: []string{"fw.rule.add_failed"},
		},
		{
			name: "no control host",
			want: []string{"fw.rule.control_failed"},
		},
		{
			name: "local rules overridden by group policy", up: []string{"77.88.8.8:443"}, gpOverride: true,
			want: []string{"fw.rule.gp_override"},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			f := &fakeFirewall{
				up: map[string]bool{}, enforce: c.enforce, gpOverride: c.gpOverride, leftovers: c.leftovers,
				addErr: c.addErr, removeErr: c.removeErr, downAfterRemove: c.downAfterRemove,
				rules: map[string]string{}, blocks: map[string]string{}, removes: map[string]int{}, dials: map[string]int{},
			}
			for _, h := range c.up {
				f.up[h] = true
			}
			if c.cancelOnAdd {
				f.onAdd = cancel
			}
			r := newRecorder(nil)
			ruleTest(ctx, f, f.dial, exe, r)

			var got []string
			for _, fd := range r.seen {
				got = append(got, fd.Code)
			}
			if !reflect.DeepEqual(got, c.want) {
				t.Fatalf("findings %v, want %v", got, c.want)
			}

			f.mu.Lock()
			defer f.mu.Unlock()
			for _, name := range c.leftovers {
				if f.removes[name] != 1 {
					t.Errorf("leftover %s removed %d times, want 1", name, f.removes[name])
				}
			}
			if c.gpOverride && len(f.dials) != 0 {
				t.Errorf("dialled %v although the rule test was skipped", f.dials)
			}
			if len(f.blocks) == 0 {
				return
			}
			if f.exe != exe {
				t.Errorf("rule program %q, want %q", f.exe, exe)
			}
			added := r.seen[slices.IndexFunc(r.seen, func(fd check.Finding) bool { return fd.Code == "fw.rule.added" })]
			name, target := added.Params["name"], added.Params["target"]
			if !strings.HasPrefix(name, rulePrefix) {
				t.Errorf("rule name %q has no %q prefix", name, rulePrefix)
			}
			if f.blocks[name] != target {
				t.Errorf("rule blocks %q, the control host was %q", f.blocks[name], target)
			}
			if f.removes[name] != 1 {
				t.Errorf("rule removed %d times, want exactly 1", f.removes[name])
			}
			if n := f.dials[target] - 1; n != c.checks {
				t.Errorf("%d connections to %s after the rule was added, want %d", n, target, c.checks)
			}
		})
	}
}

func TestControl(t *testing.T) {
	cancelled := make(chan struct{})
	dial := func(ctx context.Context, addr string) error {
		switch addr {
		case "1.1.1.1:443":
			<-ctx.Done()
			close(cancelled)
			return ctx.Err()
		case "8.8.8.8:443":
			return dialErr(wsaetimedout)
		}
		return nil
	}
	target, err := control(context.Background(), dial)
	if err != nil || target != "77.88.8.8:443" {
		t.Fatalf("control = %q, %v, want the host that answers", target, err)
	}
	select {
	case <-cancelled:
	case <-time.After(time.Second):
		t.Error("the attempt still hanging was not cancelled")
	}

	down := func(context.Context, string) error { return dialErr(wsaetimedout) }
	target, err = control(context.Background(), down)
	if err == nil || target != strings.Join(controlTargets, ", ") {
		t.Errorf("all hosts down: control = %q, %v", target, err)
	}
}
