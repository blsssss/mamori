package secprod

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
	"golang.org/x/sys/windows/svc/mgr"

	"github.com/blsssss/mamori/internal/check"
)

func Run(ctx context.Context, _ check.Options, emit check.Emit) check.Result {
	r := check.Start(check.Inventory, emit)
	var d Data
	d.Source, d.Antivirus, d.Firewalls = securityCenter(ctx, r, wscProducts, wmiProducts)
	for _, p := range d.Antivirus {
		r.Add(productStatus(p), "inv.av.product", "name", p.Name, "state", p.State, "signature", p.Signature,
			"source", p.Source, "path", p.Path, "timestamp", p.Timestamp)
	}
	for _, p := range d.Firewalls {
		r.Add(productStatus(p), "inv.fw.product", "name", p.Name, "state", p.State, "source", p.Source)
	}
	var winfwRead bool
	d.WindowsFirewall, winfwRead = windowsFirewall(ctx, r)
	d.Defender = defender(ctx, r, otherAntivirusOn(d.Antivirus))
	d.Files = evidence(r)
	r.SetData(d)
	if ctx.Err() != nil {
		return r.Finish(check.Skip, "common.cancelled")
	}
	status, code := verdict(d, winfwRead)
	return r.Finish(status, code)
}

func windowsFirewall(ctx context.Context, r *check.Recorder) (WindowsFirewall, bool) {
	wf, err := Firewall(ctx)
	if wf.Service != "" {
		r.Add(serviceStatus(wf.Service), "inv.winfw.service", "state", wf.Service)
	}
	if err != nil {
		r.Add(check.Warn, "inv.winfw.unavailable", "error", err.Error())
	}
	for _, p := range wf.Profiles {
		r.Add(profileStatus(p), "inv.winfw.profile", "profile", p.Name, "enabled", strconv.FormatBool(p.Enabled),
			"active", strconv.FormatBool(p.Active), "inbound", p.Inbound, "outbound", p.Outbound)
	}
	return wf, err == nil
}

func defender(ctx context.Context, r *check.Recorder, otherAV bool) *Defender {
	state, err := serviceState("WinDefend")
	if err != nil {
		if errors.Is(err, windows.ERROR_SERVICE_DOES_NOT_EXIST) {
			r.Add(check.Skip, "inv.defender.absent", "error", err.Error())
		} else {
			r.Add(check.Warn, "inv.defender.unavailable", "service", "unknown", "error", err.Error())
		}
		return nil
	}
	d, err := readDefender(ctx, state)
	switch {
	case wbemStatus(err) == wbemInvalidNamespace:
		r.Add(check.Skip, "inv.defender.absent", "error", err.Error())
		return nil
	case err != nil:
		// the service is there but its status is not, so Data claims nothing about real-time protection
		r.Add(check.Warn, "inv.defender.unavailable", "service", state, "error", err.Error())
		return nil
	}
	age := ""
	if d.SignatureAge != nil {
		age = strconv.FormatUint(uint64(*d.SignatureAge), 10)
	}
	r.Add(defenderStatus(*d, otherAV), "inv.defender.status", "realtime", strconv.FormatBool(d.RealTime),
		"antivirus", strconv.FormatBool(d.Antivirus), "service", d.Service, "mode", d.Mode,
		"signature_age", age, "signature_version", d.SignatureVersion)
	return d
}

type mpComputerStatus struct {
	AMRunningMode             string
	AntivirusEnabled          bool
	RealTimeProtectionEnabled bool
	AntivirusSignatureAge     uint32
	AntivirusSignatureVersion string
}

func readDefender(ctx context.Context, service string) (*Defender, error) {
	rows, err := await(ctx, func() ([]mpComputerStatus, error) {
		var rows []mpComputerStatus
		err := wmiClient.Query("SELECT * FROM MSFT_MpComputerStatus", &rows, nil, `root\Microsoft\Windows\Defender`)
		return rows, err
	})
	if err == nil && len(rows) == 0 {
		err = errors.New("no instance")
	}
	if err != nil {
		if service != "running" {
			// the provider lives in the stopped service and fails to load, yet a stopped Defender
			// is known to protect nothing
			return &Defender{Service: service}, nil
		}
		return nil, fmt.Errorf(`root\Microsoft\Windows\Defender MSFT_MpComputerStatus: %w`, err)
	}
	s := rows[0]
	return &Defender{
		Service:          service,
		Mode:             s.AMRunningMode,
		RealTime:         s.RealTimeProtectionEnabled,
		Antivirus:        s.AntivirusEnabled,
		SignatureAge:     &s.AntivirusSignatureAge,
		SignatureVersion: s.AntivirusSignatureVersion,
	}, nil
}

func evidence(r *check.Recorder) []Hit {
	processes, err := processNames()
	if err != nil {
		r.Add(check.Warn, "inv.files.unavailable", "evidence", EvidenceProcess, "error", err.Error())
	}
	services, err := serviceNames()
	if err != nil {
		r.Add(check.Warn, "inv.files.unavailable", "evidence", EvidenceService, "error", err.Error())
	}
	hits := findEvidence(vendors, expand, isDir, processes, services)
	for _, h := range hits {
		r.Add(check.Pass, "inv.files.found", "vendor", h.Vendor, "kind", h.Kind, "evidence", h.Evidence, "value", h.Value)
	}
	if len(hits) == 0 {
		r.Add(check.Warn, "inv.files.none")
	}
	return hits
}

func expand(p string) string {
	if s, err := registry.ExpandString(p); err == nil {
		return s
	}
	return p
}

func isDir(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && fi.IsDir()
}

func processNames() ([]string, error) {
	snap, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return nil, fmt.Errorf("CreateToolhelp32Snapshot: %w", err)
	}
	defer func() { _ = windows.CloseHandle(snap) }()
	var names []string
	e := windows.ProcessEntry32{Size: uint32(unsafe.Sizeof(windows.ProcessEntry32{}))}
	for err = windows.Process32First(snap, &e); err == nil; err = windows.Process32Next(snap, &e) {
		names = append(names, windows.UTF16ToString(e.ExeFile[:]))
	}
	if !errors.Is(err, windows.ERROR_NO_MORE_FILES) {
		return names, fmt.Errorf("Process32Next: %w", err)
	}
	return names, nil
}

// serviceNames opens the service manager with the enumerate right only: mgr.Connect asks for full
// access, which needs admin
func serviceNames() ([]string, error) {
	h, err := windows.OpenSCManager(nil, nil, windows.SC_MANAGER_ENUMERATE_SERVICE)
	if err != nil {
		return nil, fmt.Errorf("OpenSCManager: %w", err)
	}
	m := &mgr.Mgr{Handle: h}
	defer func() { _ = m.Disconnect() }()
	names, err := m.ListServices()
	if err != nil {
		return nil, fmt.Errorf("EnumServicesStatusEx: %w", err)
	}
	return names, nil
}
