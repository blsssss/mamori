package main

import (
	"context"

	"github.com/blsssss/mamori/internal/avprobe"
	"github.com/blsssss/mamori/internal/check"
	"github.com/blsssss/mamori/internal/fwprobe"
	"github.com/blsssss/mamori/internal/netcheck"
	"github.com/blsssss/mamori/internal/secprod"
)

type checkFunc func(context.Context, check.Options, check.Emit) check.Result

var checks = map[check.ID]checkFunc{
	check.Internet:  netcheck.Run,
	check.Inventory: secprod.Run,
	check.Firewall:  fwprobe.Run,
	check.Antivirus: avprobe.Run,
}
