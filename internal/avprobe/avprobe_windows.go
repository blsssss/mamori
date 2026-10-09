package avprobe

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/blsssss/mamori/internal/check"
)

const (
	eicarPoll = 250 * time.Millisecond
	eicarFile = "eicar-test.txt"
)

var (
	amsiDLL              = windows.NewLazySystemDLL("amsi.dll")
	procAmsiInitialize   = amsiDLL.NewProc("AmsiInitialize")
	procAmsiOpenSession  = amsiDLL.NewProc("AmsiOpenSession")
	procAmsiScanString   = amsiDLL.NewProc("AmsiScanString")
	procAmsiCloseSession = amsiDLL.NewProc("AmsiCloseSession")
	procAmsiUninitialize = amsiDLL.NewProc("AmsiUninitialize")
)

func Run(ctx context.Context, opts check.Options, emit check.Emit) check.Result {
	r := check.Start(check.Antivirus, emit)
	wait := opts.EicarWait
	if wait <= 0 {
		wait = check.DefaultOptions().EicarWait
	}
	eicar := eicarTest(ctx, r, wait)
	if ctx.Err() != nil {
		return r.Finish(check.Skip, "common.cancelled")
	}
	status, code := verdict(eicar, amsiTest(r))
	return r.Finish(status, code)
}

func eicarTest(ctx context.Context, r *check.Recorder, wait time.Duration) outcome {
	if ctx.Err() != nil {
		return notRun
	}
	dir, err := os.MkdirTemp("", "mamori-av-*")
	if err != nil {
		r.Add(check.Error, "av.eicar.write_failed", "error", err.Error())
		return notRun
	}
	defer cleanup(r, dir)

	path := filepath.Join(dir, eicarFile)
	data := reveal(eicarEnc)
	err = os.WriteFile(path, data, 0o600)
	clear(data)
	if err != nil {
		if blockedByAV(err) {
			r.Add(check.Pass, "av.eicar.blocked_on_write", "error", err.Error())
			return detected
		}
		r.Add(check.Error, "av.eicar.write_failed", "error", err.Error())
		return notRun
	}

	// every poll opens the file again, the open is what triggers an on-access scan
	start := time.Now()
	for {
		content, err := os.ReadFile(path)
		elapsed := seconds(time.Since(start))
		switch readState(content, err) {
		case fileRemoved:
			r.Add(check.Pass, "av.eicar.removed", "seconds", elapsed)
			return detected
		case fileBlocked:
			r.Add(check.Pass, "av.eicar.blocked_on_read", "error", err.Error(), "seconds", elapsed)
			return detected
		case fileAltered:
			r.Add(check.Pass, "av.eicar.altered", "seconds", elapsed)
			return detected
		}
		if time.Since(start) >= wait {
			params := []string{"seconds", elapsed}
			if err != nil {
				params = append(params, "error", err.Error())
			}
			r.Add(check.Fail, "av.eicar.not_detected", params...)
			return notDetected
		}
		select {
		case <-ctx.Done():
			return notRun
		case <-time.After(eicarPoll):
		}
	}
}

// the scanner may still hold the file for a moment while it quarantines it, hence the retries
func cleanup(r *check.Recorder, dir string) {
	err := os.RemoveAll(dir)
	for i := 0; err != nil && i < 3; i++ {
		time.Sleep(eicarPoll)
		err = os.RemoveAll(dir)
	}
	if err != nil {
		r.Add(check.Warn, "av.eicar.cleanup_failed", "path", dir, "error", err.Error())
	}
}

func amsiTest(r *check.Recorder) outcome {
	result, err := amsiScan()
	if err != nil {
		r.Add(check.Warn, "av.amsi.unavailable", "error", err.Error())
		return notRun
	}
	n := strconv.FormatUint(uint64(result), 10)
	if classifyAMSI(result) == detected {
		r.Add(check.Pass, "av.amsi.detected", "result", n)
		return detected
	}
	r.Add(check.Fail, "av.amsi.not_detected", "result", n)
	return notDetected
}

func amsiScan() (uint32, error) {
	// LazyProc.Call panics on a missing export, Find turns that into an error
	for _, p := range []*windows.LazyProc{procAmsiInitialize, procAmsiOpenSession, procAmsiScanString, procAmsiCloseSession, procAmsiUninitialize} {
		if err := p.Find(); err != nil {
			return 0, err
		}
	}
	app, err := windows.UTF16PtrFromString("mamori")
	if err != nil {
		return 0, err
	}
	sample, err := windows.UTF16PtrFromString(string(reveal(amsiEnc)))
	if err != nil {
		return 0, err
	}

	var amsiCtx uintptr
	hr, _, _ := procAmsiInitialize.Call(uintptr(unsafe.Pointer(app)), uintptr(unsafe.Pointer(&amsiCtx)))
	if err := hresult("AmsiInitialize", hr); err != nil {
		return 0, err
	}
	defer func() { _, _, _ = procAmsiUninitialize.Call(amsiCtx) }()

	var session uintptr
	hr, _, _ = procAmsiOpenSession.Call(amsiCtx, uintptr(unsafe.Pointer(&session)))
	if err := hresult("AmsiOpenSession", hr); err != nil {
		return 0, err
	}
	defer func() { _, _, _ = procAmsiCloseSession.Call(amsiCtx, session) }()

	var result uint32
	hr, _, _ = procAmsiScanString.Call(amsiCtx, uintptr(unsafe.Pointer(sample)), uintptr(unsafe.Pointer(app)), session, uintptr(unsafe.Pointer(&result)))
	if err := hresult("AmsiScanString", hr); err != nil {
		return 0, err
	}
	return result, nil
}

// only the low 32 bits of the return register hold the HRESULT
func hresult(name string, hr uintptr) error {
	code := uint32(hr)
	if int32(code) >= 0 {
		return nil
	}
	// FACILITY_WIN32 wraps a win32 error that has a readable system message
	if code&0xFFFF0000 == 0x80070000 {
		return fmt.Errorf("%s: 0x%08X: %w", name, code, windows.Errno(code&0xFFFF))
	}
	return fmt.Errorf("%s: 0x%08X", name, code)
}

func seconds(d time.Duration) string {
	return strconv.FormatFloat(d.Seconds(), 'f', 1, 64)
}
