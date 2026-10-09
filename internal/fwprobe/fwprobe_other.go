//go:build !windows

package fwprobe

import (
	"context"

	"github.com/blsssss/mamori/internal/check"
)

func Run(_ context.Context, _ check.Options, emit check.Emit) check.Result {
	return check.Start(check.Firewall, emit).Finish(check.Skip, "common.unsupported_os")
}

func RunRuleTest(_ context.Context, _ check.Options, emit check.Emit) check.Result {
	return check.Start(check.Firewall, emit).Finish(check.Skip, "common.unsupported_os")
}
