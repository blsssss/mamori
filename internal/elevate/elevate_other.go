//go:build !windows

package elevate

import (
	"context"
	"errors"
)

func IsElevated() bool { return false }

func RunSelf(context.Context, ...string) (int, error) {
	return -1, errors.New("elevation is only supported on windows")
}
