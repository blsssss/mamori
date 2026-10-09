package fwprobe

import (
	"errors"
	"fmt"
	"runtime"
	"strings"

	ole "github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
)

const (
	sFalse          = 0x00000001
	rpcEChangedMode = 0x80010106
)

// values from netfw.h
const (
	fwProtocolTCP  int32 = 6
	fwDirectionOut int32 = 2
	fwActionBlock  int32 = 0
	fwProfileAll   int32 = 0x7FFFFFFF
	// NET_FW_MODIFY_STATE_INBOUND_BLOCKED (2) concerns inbound rules only, the probe rule is outbound
	fwModifyGPOverride int32 = 1
)

// withCOM runs fn on one locked OS thread with COM initialized: COM objects belong to the
// apartment of the thread that created them
func withCOM(fn func() error) error {
	done := make(chan error, 1)
	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		err := ole.CoInitializeEx(0, ole.COINIT_APARTMENTTHREADED)
		var oe *ole.OleError
		switch {
		case err == nil, errors.As(err, &oe) && oe.Code() == sFalse:
			defer ole.CoUninitialize()
		case errors.As(err, &oe) && oe.Code() == rpcEChangedMode:
			// the thread already has COM in another mode, which works too and is not ours to undo
		default:
			done <- fmt.Errorf("CoInitializeEx: %w", err)
			return
		}
		done <- fn()
	}()
	return <-done
}

type fwPolicy struct {
	policy *ole.IDispatch
	rules  *ole.IDispatch
}

func openPolicy() (*fwPolicy, error) {
	policy, err := create("HNetCfg.FwPolicy2")
	if err != nil {
		return nil, err
	}
	v, err := oleutil.GetProperty(policy, "Rules")
	if err != nil {
		policy.Release()
		return nil, fmt.Errorf("get Rules: %w", err)
	}
	rules := v.ToIDispatch()
	if rules == nil {
		policy.Release()
		return nil, fmt.Errorf("get Rules: unexpected variant type %d", v.VT)
	}
	return &fwPolicy{policy: policy, rules: rules}, nil
}

func (p *fwPolicy) close() {
	p.rules.Release()
	p.policy.Release()
}

func (p *fwPolicy) names(prefix string) ([]string, error) {
	var out []string
	err := oleutil.ForEach(p.rules, func(v *ole.VARIANT) error {
		rule := v.ToIDispatch()
		if rule == nil {
			return v.Clear()
		}
		// the variant holds the only reference, releasing the rule frees it
		defer rule.Release()
		name, err := getString(rule, "Name")
		if err == nil && strings.HasPrefix(name, prefix) {
			out = append(out, name)
		}
		return err
	})
	return out, err
}

func (p *fwPolicy) addBlock(name, exe, ip, port string) error {
	rule, err := create("HNetCfg.FWRule")
	if err != nil {
		return err
	}
	defer rule.Release()
	props := []struct {
		name  string
		value any
	}{
		{"Name", name},
		{"Description", "Temporary rule of the mamori firewall test, removed right after the test"},
		{"Grouping", "mamori"},
		{"ApplicationName", exe},
		// INetFwRule rejects ports until the protocol is set
		{"Protocol", fwProtocolTCP},
		{"RemoteAddresses", ip},
		{"RemotePorts", port},
		{"Direction", fwDirectionOut},
		{"Action", fwActionBlock},
		{"Profiles", fwProfileAll},
		{"Enabled", true},
	}
	for _, pr := range props {
		if err := put(rule, pr.name, pr.value); err != nil {
			return err
		}
	}
	return call(p.rules, "Add", rule)
}

func (p *fwPolicy) remove(name string) error {
	return call(p.rules, "Remove", name)
}

func (p *fwPolicy) localRulesApply() (bool, error) {
	v, err := oleutil.GetProperty(p.policy, "LocalPolicyModifyState")
	if err != nil {
		return false, fmt.Errorf("LocalPolicyModifyState: %w", err)
	}
	state, ok := v.Value().(int32)
	vt := v.VT
	if err := v.Clear(); err != nil {
		return false, fmt.Errorf("LocalPolicyModifyState: %w", err)
	}
	if !ok {
		return false, fmt.Errorf("LocalPolicyModifyState: unexpected variant type %d", vt)
	}
	return state != fwModifyGPOverride, nil
}

func create(progID string) (*ole.IDispatch, error) {
	unknown, err := oleutil.CreateObject(progID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", progID, err)
	}
	defer unknown.Release()
	disp, err := unknown.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", progID, err)
	}
	return disp, nil
}

func getString(d *ole.IDispatch, name string) (string, error) {
	v, err := oleutil.GetProperty(d, name)
	if err != nil {
		return "", fmt.Errorf("%s: %w", name, err)
	}
	s := v.ToString()
	return s, v.Clear()
}

func put(d *ole.IDispatch, name string, value any) error {
	v, err := oleutil.PutProperty(d, name, value)
	if err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	return v.Clear()
}

func call(d *ole.IDispatch, method string, args ...any) error {
	v, err := oleutil.CallMethod(d, method, args...)
	if err != nil {
		return fmt.Errorf("%s: %w", method, err)
	}
	return v.Clear()
}
