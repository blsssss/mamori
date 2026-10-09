package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/blsssss/mamori/internal/check"
	"github.com/blsssss/mamori/internal/elevate"
	"github.com/blsssss/mamori/internal/report"
	"github.com/blsssss/mamori/internal/sysinfo"
	"github.com/blsssss/mamori/internal/version"
)

type App struct {
	ctx     context.Context
	running sync.Mutex
	mu      sync.Mutex
	cancel  context.CancelFunc
}

type AppInfo struct {
	Version  string       `json:"version"`
	Commit   string       `json:"commit"`
	Elevated bool         `json:"elevated"`
	System   sysinfo.Info `json:"system"`
}

type RunOptions struct {
	PolicyTargets  []string `json:"policyTargets"`
	EicarWaitSec   int      `json:"eicarWaitSec"`
	AllowElevation bool     `json:"allowElevation"`
}

type FindingEvent struct {
	Check   check.ID      `json:"check"`
	Finding check.Finding `json:"finding"`
}

func NewApp() *App { return &App{} }

func (a *App) startup(ctx context.Context) { a.ctx = ctx }

func (a *App) Info() AppInfo {
	return AppInfo{
		Version:  version.Version,
		Commit:   version.Commit,
		Elevated: elevate.IsElevated(),
		System:   sysinfo.Get(),
	}
}

// Run executes one check and streams its findings as "finding" events while it runs
func (a *App) Run(id string, opts RunOptions) (check.Result, error) {
	run, ok := checks[check.ID(id)]
	if !ok {
		return check.Result{}, fmt.Errorf("unknown check %q", id)
	}
	if !a.running.TryLock() {
		return check.Result{}, errors.New("another check is running")
	}
	defer a.running.Unlock()

	ctx, cancel := context.WithTimeout(a.ctx, 3*time.Minute)
	a.setCancel(cancel)
	defer func() {
		cancel()
		a.setCancel(nil)
	}()

	o := check.DefaultOptions()
	o.Elevated = elevate.IsElevated()
	for _, t := range opts.PolicyTargets {
		if t = strings.TrimSpace(t); t != "" {
			o.PolicyTargets = append(o.PolicyTargets, t)
		}
	}
	if opts.EicarWaitSec > 0 {
		o.EicarWait = time.Duration(opts.EicarWaitSec) * time.Second
	}
	if opts.AllowElevation && !o.Elevated {
		o.Elevate = elevatedProbe
	}
	emit := func(id check.ID, f check.Finding) {
		runtime.EventsEmit(a.ctx, "finding", FindingEvent{Check: id, Finding: f})
	}
	return run(ctx, o, emit), nil
}

func (a *App) Cancel() {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.cancel != nil {
		a.cancel()
	}
}

func (a *App) setCancel(c context.CancelFunc) {
	a.mu.Lock()
	a.cancel = c
	a.mu.Unlock()
}

func (a *App) Summarize(results []check.Result) report.Summary {
	return report.Summarize(sysinfo.Get(), results)
}

// SaveReport asks where to save and writes content there; an empty path means the user cancelled
func (a *App) SaveReport(defaultName, content string) (string, error) {
	path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		DefaultFilename: defaultName,
		Filters: []runtime.FileFilter{
			{DisplayName: "Report (*.txt, *.html, *.json)", Pattern: "*.txt;*.html;*.json"},
		},
	})
	if err != nil || path == "" {
		return "", err
	}
	return path, os.WriteFile(path, []byte(content), 0o644)
}

func (a *App) Quit() { runtime.Quit(a.ctx) }
