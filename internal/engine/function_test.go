package engine

import (
	"context"

	"github.com/nawafinity/go-namat/internal/value"
)

func testFunction(call func(context.Context, ...any) (any, error)) FunctionSpec {
	return FunctionSpec{Returns: value.Any, Call: call}
}
