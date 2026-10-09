// Package check holds the result model shared by every check and the recorder that builds it.
package check

import (
	"context"
	"time"
)

type ID string

const (
	Internet  ID = "internet"
	Inventory ID = "inventory"
	Firewall  ID = "firewall"
	Antivirus ID = "antivirus"
)

var All = []ID{Internet, Inventory, Firewall, Antivirus}

// Status values are ordered by severity, see Worse
type Status string

const (
	Pass  Status = "pass"
	Skip  Status = "skip"
	Warn  Status = "warn"
	Fail  Status = "fail"
	Error Status = "error"
)

var severity = map[Status]int{Pass: 0, Skip: 1, Warn: 2, Fail: 3, Error: 4}

func Worse(a, b Status) Status {
	if severity[b] > severity[a] {
		return b
	}
	return a
}

// Finding is one observation of a check. Code is a stable key that the interface translates,
// Params fill its placeholders and carry raw values such as OS error text.
type Finding struct {
	Code   string            `json:"code"`
	Status Status            `json:"status"`
	Params map[string]string `json:"params,omitempty"`
}

type Result struct {
	Check     ID                `json:"check"`
	Status    Status            `json:"status"`
	Code      string            `json:"code"`
	Params    map[string]string `json:"params,omitempty"`
	Findings  []Finding         `json:"findings"`
	Data      any               `json:"data,omitempty"`
	Started   time.Time         `json:"started"`
	ElapsedMs int64             `json:"elapsedMs"`
}

type Emit func(ID, Finding)

// Elevate runs one probe in an elevated copy of the program and returns its result
type Elevate func(ctx context.Context, probe string) (Result, error)

type Options struct {
	// hosts or URLs that the local firewall policy is expected to block
	PolicyTargets []string
	EicarWait     time.Duration
	Elevated      bool
	Elevate       Elevate
}

func DefaultOptions() Options {
	return Options{EicarWait: 15 * time.Second}
}

type Recorder struct {
	res  Result
	emit Emit
}

func Start(id ID, emit Emit) *Recorder {
	return &Recorder{
		res:  Result{Check: id, Status: Pass, Findings: []Finding{}, Started: time.Now()},
		emit: emit,
	}
}

// Add records a finding; params are key, value pairs
func (r *Recorder) Add(status Status, code string, params ...string) {
	r.AddFinding(Finding{Code: code, Status: status, Params: pairs(params)})
}

func (r *Recorder) AddFinding(f Finding) {
	r.res.Findings = append(r.res.Findings, f)
	if r.emit != nil {
		r.emit(r.res.Check, f)
	}
}

func (r *Recorder) SetData(v any) { r.res.Data = v }

// Finish sets the verdict. The check decides it from its findings: a failed probe with a working
// fallback is a finding, not a failed check.
func (r *Recorder) Finish(status Status, code string, params ...string) Result {
	r.res.Code = code
	r.res.Params = pairs(params)
	r.res.Status = status
	r.res.ElapsedMs = time.Since(r.res.Started).Milliseconds()
	return r.res
}

// Worst is the most severe status among the findings, Pass when there are none
func (r *Recorder) Worst() Status {
	s := Pass
	for _, f := range r.res.Findings {
		s = Worse(s, f.Status)
	}
	return s
}

func pairs(kv []string) map[string]string {
	if len(kv) == 0 {
		return nil
	}
	m := make(map[string]string, len(kv)/2)
	for i := 0; i+1 < len(kv); i += 2 {
		m[kv[i]] = kv[i+1]
	}
	return m
}
