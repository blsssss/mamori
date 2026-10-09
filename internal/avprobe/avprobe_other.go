//go:build !windows

package avprobe

import (
	"context"

	"github.com/blsssss/mamori/internal/check"
)

func Run(ctx context.Context, opts check.Options, emit check.Emit) check.Result {
	return check.Start(check.Antivirus, emit).Finish(check.Skip, "common.unsupported_os")
}
