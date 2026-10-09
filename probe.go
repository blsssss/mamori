package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/blsssss/mamori/internal/check"
	"github.com/blsssss/mamori/internal/elevate"
	"github.com/blsssss/mamori/internal/fwprobe"
	"github.com/blsssss/mamori/internal/report"
	"github.com/blsssss/mamori/internal/sysinfo"
)

// runProbe is the headless mode: mamori --probe <check|all|firewall-rule> [-out file] [-policy a,b].
// No window is created, so it serves both the elevated copy and the end-to-end tests.
func runProbe(args []string) int {
	if len(args) == 0 {
		return 2
	}
	name := args[0]
	fs := flag.NewFlagSet("probe", flag.ContinueOnError)
	out := fs.String("out", "", "write the JSON result to this file instead of stdout")
	policy := fs.String("policy", "", "comma separated hosts or URLs the firewall policy must block")
	wait := fs.Duration("eicar-wait", 0, "how long to wait for the antivirus to react to the test file")
	if err := fs.Parse(args[1:]); err != nil {
		return 2
	}
	opts := check.DefaultOptions()
	opts.Elevated = elevate.IsElevated()
	if *policy != "" {
		opts.PolicyTargets = strings.Split(*policy, ",")
	}
	if *wait > 0 {
		opts.EicarWait = *wait
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	var v any
	switch name {
	case "all":
		results := make([]check.Result, 0, len(check.All))
		for _, id := range check.All {
			results = append(results, checks[id](ctx, opts, nil))
		}
		v = report.Summarize(sysinfo.Get(), results)
	case fwprobe.RuleProbe:
		v = fwprobe.RunRuleTest(ctx, opts, nil)
	default:
		run, ok := checks[check.ID(name)]
		if !ok {
			return 2
		}
		v = run(ctx, opts, nil)
	}
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return 3
	}
	if *out == "" {
		_, err = os.Stdout.Write(append(data, '\n'))
	} else {
		err = os.WriteFile(*out, data, 0o600)
	}
	if err != nil {
		return 3
	}
	return 0
}

func elevatedProbe(ctx context.Context, probe string) (check.Result, error) {
	dir, err := os.MkdirTemp("", "mamori-")
	if err != nil {
		return check.Result{}, err
	}
	defer os.RemoveAll(dir)
	out := filepath.Join(dir, "result.json")
	code, err := elevate.RunSelf(ctx, "--probe", probe, "-out", out)
	if err != nil {
		return check.Result{}, err
	}
	if code != 0 {
		return check.Result{}, fmt.Errorf("elevated probe exited with code %d", code)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		return check.Result{}, err
	}
	var res check.Result
	if err := json.Unmarshal(data, &res); err != nil {
		return check.Result{}, err
	}
	return res, nil
}
