//go:build !windows

package netcheck

import (
	"context"

	"github.com/blsssss/mamori/internal/check"
)

func Run(_ context.Context, _ check.Options, emit check.Emit) check.Result {
	return check.Start(check.Internet, emit).Finish(check.Skip, "common.unsupported_os")
}
