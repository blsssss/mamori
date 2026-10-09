//go:build !windows

package secprod

import (
	"context"
	"errors"

	"github.com/blsssss/mamori/internal/check"
)

func Run(_ context.Context, _ check.Options, emit check.Emit) check.Result {
	return check.Start(check.Inventory, emit).Finish(check.Skip, "common.unsupported_os")
}

func Products(context.Context, string) ([]Product, error) { return nil, errors.ErrUnsupported }

func Firewall(context.Context) (WindowsFirewall, error) {
	return WindowsFirewall{}, errors.ErrUnsupported
}
