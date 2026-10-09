package secprod

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"syscall"
	"time"

	"github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
	"github.com/yusufpapurcu/wmi"
	"golang.org/x/sys/windows"
)

// a broken WMI provider or COM server can hang, one source must not hold the whole check
const sourceTimeout = 15 * time.Second

var wmiClient = &wmi.Client{AllowMissingFields: true, NonePtrZero: true}

// Products lists the Security Center products of one kind: through the WSC API, or through WMI
// when the API fails
func Products(ctx context.Context, kind string) ([]Product, error) {
	list, err := wscProducts(ctx, kind)
	if err == nil {
		return list, nil
	}
	list, werr := wmiProducts(ctx, kind)
	if werr != nil {
		return nil, fmt.Errorf("wsc: %w; wmi: %w", err, werr)
	}
	return list, nil
}

var securityCenterClasses = map[string]string{
	KindAntivirus:   "AntiVirusProduct",
	KindFirewall:    "FirewallProduct",
	KindAntispyware: "AntiSpywareProduct",
}

type securityCenterProduct struct {
	DisplayName            string
	ProductState           uint32
	PathToSignedProductExe string
	Timestamp              string
}

func wmiProducts(ctx context.Context, kind string) ([]Product, error) {
	class, ok := securityCenterClasses[kind]
	if !ok {
		return nil, fmt.Errorf("unknown product kind %q", kind)
	}
	rows, err := await(ctx, func() ([]securityCenterProduct, error) {
		var rows []securityCenterProduct
		q := "SELECT displayName, productState, pathToSignedProductExe, timestamp FROM " + class
		err := wmiClient.Query(q, &rows, nil, `root\SecurityCenter2`)
		return rows, err
	})
	if err != nil {
		return nil, fmt.Errorf(`root\SecurityCenter2 %s: %w`, class, err)
	}
	list := make([]Product, 0, len(rows))
	for _, row := range rows {
		state, signature := decodeProductState(row.ProductState)
		list = append(list, Product{
			Name:      row.DisplayName,
			Kind:      kind,
			State:     state,
			Signature: signature,
			Path:      row.PathToSignedProductExe,
			Timestamp: row.Timestamp,
			Source:    SourceWMI,
		})
	}
	return withoutWindowsFirewall(list), nil
}

// Firewall reads Windows Firewall, Service is filled even when the profiles fail to read
func Firewall(ctx context.Context) (WindowsFirewall, error) {
	wf := WindowsFirewall{Profiles: []FirewallProfile{}}
	state, err := serviceState("mpssvc")
	switch {
	case errors.Is(err, windows.ERROR_SERVICE_DOES_NOT_EXIST):
		wf.Service = ServiceMissing
	case err != nil:
		return wf, err
	default:
		wf.Service = state
	}
	profiles, err := firewallProfiles(ctx)
	if err != nil {
		if wf.Service != "running" {
			// the firewall filters nothing without its service, its settings do not change that
			return wf, nil
		}
		return wf, err
	}
	wf.Profiles = profiles
	return wf, nil
}

// NET_FW_PROFILE_TYPE2 bits from icftypes.h
var profileTypes = []struct {
	name string
	bit  int32
}{
	{"domain", 0x1},
	{"private", 0x2},
	{"public", 0x4},
}

// firewallProfiles uses INetFwPolicy2, which returns the effective settings: the WMI class
// MSFT_NetFirewallProfile reports "not configured" for defaults
func firewallProfiles(ctx context.Context) ([]FirewallProfile, error) {
	return withCOM(ctx, func() ([]FirewallProfile, error) {
		unk, err := oleutil.CreateObject("HNetCfg.FwPolicy2")
		if err != nil {
			return nil, fmt.Errorf("HNetCfg.FwPolicy2: %w", err)
		}
		defer unk.Release()
		policy, err := unk.QueryInterface(ole.IID_IDispatch)
		if err != nil {
			return nil, fmt.Errorf("HNetCfg.FwPolicy2: %w", err)
		}
		defer policy.Release()

		current, err := intProperty(policy, "CurrentProfileTypes")
		if err != nil {
			return nil, err
		}
		profiles := make([]FirewallProfile, 0, len(profileTypes))
		for _, t := range profileTypes {
			enabled, err := boolProperty(policy, "FirewallEnabled", t.bit)
			if err != nil {
				return nil, err
			}
			in, err := intProperty(policy, "DefaultInboundAction", t.bit)
			if err != nil {
				return nil, err
			}
			out, err := intProperty(policy, "DefaultOutboundAction", t.bit)
			if err != nil {
				return nil, err
			}
			profiles = append(profiles, FirewallProfile{
				Name:     t.name,
				Enabled:  enabled,
				Active:   current&int64(t.bit) != 0,
				Inbound:  fwAction(in),
				Outbound: fwAction(out),
			})
		}
		return profiles, nil
	})
}

func property(disp *ole.IDispatch, name string, args ...any) (any, error) {
	v, err := oleutil.GetProperty(disp, name, args...)
	if err != nil {
		return nil, fmt.Errorf("INetFwPolicy2.%s: %w", name, err)
	}
	defer func() { _ = v.Clear() }()
	return v.Value(), nil
}

func boolProperty(disp *ole.IDispatch, name string, args ...any) (bool, error) {
	v, err := property(disp, name, args...)
	if err != nil {
		return false, err
	}
	b, ok := v.(bool)
	if !ok {
		return false, fmt.Errorf("INetFwPolicy2.%s: unexpected %T", name, v)
	}
	return b, nil
}

func intProperty(disp *ole.IDispatch, name string, args ...any) (int64, error) {
	v, err := property(disp, name, args...)
	if err != nil {
		return 0, err
	}
	switch n := v.(type) {
	case int32:
		return int64(n), nil
	case int64:
		return n, nil
	}
	return 0, fmt.Errorf("INetFwPolicy2.%s: unexpected %T", name, v)
}

// serviceState asks only for SERVICE_QUERY_STATUS: svc/mgr opens with full access, which needs admin
func serviceState(name string) (string, error) {
	scm, err := windows.OpenSCManager(nil, nil, windows.SC_MANAGER_CONNECT)
	if err != nil {
		return "", fmt.Errorf("OpenSCManager: %w", err)
	}
	defer func() { _ = windows.CloseServiceHandle(scm) }()
	n, err := windows.UTF16PtrFromString(name)
	if err != nil {
		return "", err
	}
	s, err := windows.OpenService(scm, n, windows.SERVICE_QUERY_STATUS)
	if err != nil {
		return "", fmt.Errorf("OpenService(%s): %w", name, err)
	}
	defer func() { _ = windows.CloseServiceHandle(s) }()
	var st windows.SERVICE_STATUS
	if err := windows.QueryServiceStatus(s, &st); err != nil {
		return "", fmt.Errorf("QueryServiceStatus(%s): %w", name, err)
	}
	return serviceStateName(st.CurrentState), nil
}

// await runs fn on its own goroutine so that a call hung inside WMI or COM gives up at the deadline,
// though a call that never returns keeps its goroutine, and under withCOM its locked OS thread
func await[T any](ctx context.Context, fn func() (T, error)) (T, error) {
	var zero T
	if err := ctx.Err(); err != nil {
		return zero, err
	}
	ctx, cancel := context.WithTimeout(ctx, sourceTimeout)
	defer cancel()
	type result struct {
		v   T
		err error
	}
	done := make(chan result, 1)
	go func() {
		v, err := fn()
		done <- result{v, err}
	}()
	select {
	case r := <-done:
		return r.v, r.err
	case <-ctx.Done():
		return zero, ctx.Err()
	}
}

// WBEM_E_INVALID_NAMESPACE from wbemcli.h
const wbemInvalidNamespace = 0x8004100E

// wbemStatus finds the WBEM error code, which the scripting API returns inside the
// DISP_E_EXCEPTION of IDispatch::Invoke
func wbemStatus(err error) uint32 {
	var oe *ole.OleError
	if !errors.As(err, &oe) {
		return 0
	}
	var ei ole.EXCEPINFO
	if errors.As(oe.SubError(), &ei) && ei.SCODE() != 0 {
		return ei.SCODE()
	}
	return uint32(oe.Code())
}

// withCOM runs fn on a locked OS thread in a single-threaded apartment: WSCProductList is
// registered with ThreadingModel Apartment, so it is then created and called on this very thread
func withCOM[T any](ctx context.Context, fn func() (T, error)) (T, error) {
	return await(ctx, func() (T, error) {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		switch err := windows.CoInitializeEx(0, windows.COINIT_APARTMENTTHREADED); {
		case err == nil, errors.Is(err, syscall.Errno(windows.S_FALSE)):
			defer windows.CoUninitialize()
		case errors.Is(err, syscall.Errno(windows.RPC_E_CHANGED_MODE)):
			// the thread is already in a multithreaded apartment, usable and not ours to uninitialize
		default:
			var zero T
			return zero, fmt.Errorf("CoInitializeEx: %w", err)
		}
		return fn()
	})
}
