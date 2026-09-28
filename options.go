package namat

import (
	"context"
	"fmt"
	"time"

	"github.com/nawafinity/go-namat/internal/expr"
)

// Function is a host function exposed to template expressions.
type Function func(args ...any) (any, error)

// ErrorHandler may replace the output of a failed value command. Returning a
// non-nil error aborts rendering.
type ErrorHandler func(command string, err error) (replacement any, returnedErr error)

// Options controls compilation and rendering.
type Options struct {
	OpenDelimiter  string
	CloseDelimiter string
	Functions      map[string]Function
	ErrorHandler   ErrorHandler
	RejectNullish  bool
	MaxIterations  int
	Timeout        time.Duration
}

func (o Options) normalized() Options {
	if o.OpenDelimiter == "" {
		o.OpenDelimiter = "[["
	}
	if o.CloseDelimiter == "" {
		o.CloseDelimiter = "]]"
	}
	if o.MaxIterations <= 0 {
		o.MaxIterations = 1_000_000
	}
	if o.Timeout <= 0 {
		o.Timeout = 30 * time.Second
	}
	return o
}

func (o Options) expressionFunctions() map[string]expr.Function {
	functions := builtinFunctions()
	for name, fn := range o.Functions {
		function := fn
		functions[name] = func(args ...any) (any, error) { return function(args...) }
	}
	return functions
}

func builtinFunctions() map[string]expr.Function {
	return map[string]expr.Function{
		"len": func(args ...any) (any, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("len expects one argument")
			}
			return collectionLength(args[0])
		},
		"string": func(args ...any) (any, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("string expects one argument")
			}
			return formatValue(args[0]), nil
		},
	}
}

func renderContext(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if parent == nil {
		parent = context.Background()
	}
	return context.WithTimeout(parent, timeout)
}
