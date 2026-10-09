package report

import (
	"testing"

	"github.com/blsssss/mamori/internal/check"
)

func res(id check.ID, s check.Status) check.Result { return check.Result{Check: id, Status: s} }

func TestVerdict(t *testing.T) {
	cases := []struct {
		name    string
		results []check.Result
		status  check.Status
		code    string
	}{
		{"none", nil, check.Skip, "report.none"},
		{"all skipped", []check.Result{res(check.Firewall, check.Skip)}, check.Skip, "report.none"},
		{"all pass", []check.Result{res(check.Internet, check.Pass), res(check.Inventory, check.Pass), res(check.Firewall, check.Pass), res(check.Antivirus, check.Pass)}, check.Pass, "report.protected"},
		{"some skipped", []check.Result{res(check.Internet, check.Pass), res(check.Firewall, check.Skip)}, check.Warn, "report.partial"},
		{"offline only", []check.Result{res(check.Internet, check.Fail), res(check.Firewall, check.Pass), res(check.Antivirus, check.Pass)}, check.Warn, "report.degraded"},
		{"firewall broken", []check.Result{res(check.Internet, check.Pass), res(check.Firewall, check.Fail), res(check.Antivirus, check.Pass)}, check.Fail, "report.unprotected"},
		{"antivirus error", []check.Result{res(check.Antivirus, check.Error), res(check.Firewall, check.Fail)}, check.Error, "report.error"},
	}
	for _, c := range cases {
		s, code := verdict(c.results)
		if s != c.status || code != c.code {
			t.Errorf("%s: got %s %s, want %s %s", c.name, s, code, c.status, c.code)
		}
	}
}
