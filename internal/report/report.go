// Package report combines the results of all checks into one summary.
package report

import (
	"time"

	"github.com/blsssss/mamori/internal/check"
	"github.com/blsssss/mamori/internal/sysinfo"
	"github.com/blsssss/mamori/internal/version"
)

type Summary struct {
	Version   string         `json:"version"`
	Commit    string         `json:"commit,omitempty"`
	System    sysinfo.Info   `json:"system"`
	Generated time.Time      `json:"generated"`
	Status    check.Status   `json:"status"`
	Code      string         `json:"code"`
	Results   []check.Result `json:"results"`
}

func Summarize(sys sysinfo.Info, results []check.Result) Summary {
	s := Summary{
		Version:   version.Version,
		Commit:    version.Commit,
		System:    sys,
		Generated: time.Now(),
		Results:   results,
	}
	s.Status, s.Code = verdict(results)
	return s
}

// verdict is about protection: a working connection does not make up for a broken firewall, and
// a missing connection alone does not make the machine unprotected
func verdict(results []check.Result) (check.Status, string) {
	status := check.Pass
	ran := 0
	for _, r := range results {
		if r.Status == check.Skip {
			continue
		}
		ran++
		s := r.Status
		if r.Check == check.Internet && s == check.Fail {
			s = check.Warn
		}
		status = check.Worse(status, s)
	}
	switch {
	case ran == 0:
		return check.Skip, "report.none"
	case status == check.Pass && ran < len(results):
		return check.Warn, "report.partial"
	case status == check.Pass:
		return check.Pass, "report.protected"
	case status == check.Warn:
		return check.Warn, "report.degraded"
	case status == check.Error:
		return check.Error, "report.error"
	default:
		return check.Fail, "report.unprotected"
	}
}
