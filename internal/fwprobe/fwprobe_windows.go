package fwprobe

import (
	"context"
	"os"

	"github.com/blsssss/mamori/internal/check"
	"github.com/blsssss/mamori/internal/secprod"
)

func Run(ctx context.Context, opts check.Options, emit check.Emit) check.Result {
	r := newRecorder(emit)
	w, werr := secprod.Firewall(ctx)
	products, perr := secprod.Products(ctx, secprod.KindFirewall)
	r.add(stateFindings(w, werr, products, perr)...)

	policy := policyTest(ctx, opts.PolicyTargets, connect)
	if ctx.Err() != nil {
		return r.finish(ctx, verdict)
	}
	r.add(policy...)

	switch name := thirdPartyOnly(w, werr, products); {
	case name != "":
		r.add(finding(check.Warn, "fw.rule.third_party", "name", name))
	case opts.Elevated:
		liveRuleTest(ctx, r)
	case opts.Elevate != nil:
		res, err := opts.Elevate(ctx, RuleProbe)
		if ctx.Err() == nil {
			r.add(elevatedFindings(res, err)...)
		}
	default:
		r.add(finding(check.Skip, "fw.rule.needs_admin"))
	}
	return r.finish(ctx, verdict)
}

// RunRuleTest needs administrator rights: it adds a temporary block rule and checks that it holds
func RunRuleTest(ctx context.Context, _ check.Options, emit check.Emit) check.Result {
	r := newRecorder(emit)
	liveRuleTest(ctx, r)
	return r.finish(ctx, ruleVerdict)
}

func liveRuleTest(ctx context.Context, r *recorder) {
	exe, err := os.Executable()
	if err != nil {
		r.add(finding(check.Error, "fw.rule.add_failed", "error", err.Error()))
		return
	}
	err = withCOM(func() error {
		fw, err := openPolicy()
		if err != nil {
			return err
		}
		defer fw.close()
		ruleTest(ctx, fw, connect, exe, r)
		return nil
	})
	if err != nil {
		r.add(finding(check.Error, "fw.rule.add_failed", "error", err.Error()))
	}
}
