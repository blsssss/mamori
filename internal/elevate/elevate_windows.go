// Package elevate starts a copy of the program with administrator rights through UAC.
package elevate

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

var procShellExecuteExW = windows.NewLazySystemDLL("shell32.dll").NewProc("ShellExecuteExW")

const (
	seeMaskNoCloseProcess = 0x00000040
	seeMaskNoAsync        = 0x00000100
	swHide                = 0
)

// SHELLEXECUTEINFOW, x/sys/windows only wraps ShellExecute, which gives no process handle
type shellExecuteInfo struct {
	Size          uint32
	Mask          uint32
	Hwnd          windows.Handle
	Verb          *uint16
	File          *uint16
	Parameters    *uint16
	Directory     *uint16
	Show          int32
	InstApp       windows.Handle
	IDList        uintptr
	Class         *uint16
	KeyClass      windows.Handle
	HotKey        uint32
	IconOrMonitor windows.Handle
	Process       windows.Handle
}

func IsElevated() bool {
	return windows.GetCurrentProcessToken().IsElevated()
}

// RunSelf runs the current executable elevated with args, waits for it and returns its exit code.
// ErrCancelled means the user declined the UAC prompt.
func RunSelf(ctx context.Context, args ...string) (int, error) {
	exe, err := os.Executable()
	if err != nil {
		return -1, err
	}
	quoted := make([]string, len(args))
	for i, a := range args {
		quoted[i] = windows.EscapeArg(a)
	}
	verb, _ := windows.UTF16PtrFromString("runas")
	file, err := windows.UTF16PtrFromString(exe)
	if err != nil {
		return -1, err
	}
	params, err := windows.UTF16PtrFromString(strings.Join(quoted, " "))
	if err != nil {
		return -1, err
	}
	info := shellExecuteInfo{
		Mask:       seeMaskNoCloseProcess | seeMaskNoAsync,
		Verb:       verb,
		File:       file,
		Parameters: params,
		Show:       swHide,
	}
	info.Size = uint32(unsafe.Sizeof(info))
	if r, _, e := procShellExecuteExW.Call(uintptr(unsafe.Pointer(&info))); r == 0 {
		if errors.Is(e, windows.ERROR_CANCELLED) {
			return -1, ErrCancelled
		}
		return -1, fmt.Errorf("ShellExecuteEx: %w", e)
	}
	if info.Process == 0 {
		return -1, errors.New("ShellExecuteEx returned no process handle")
	}
	defer func() { _ = windows.CloseHandle(info.Process) }()
	for {
		ev, err := windows.WaitForSingleObject(info.Process, 200)
		if err != nil {
			return -1, err
		}
		if ev == windows.WAIT_OBJECT_0 {
			break
		}
		// the copy is not killed: it may hold a temporary firewall rule, and only its own deferred
		// cleanup removes it
		if ctx.Err() != nil {
			return -1, ctx.Err()
		}
	}
	var code uint32
	if err := windows.GetExitCodeProcess(info.Process, &code); err != nil {
		return -1, err
	}
	return int(code), nil
}
