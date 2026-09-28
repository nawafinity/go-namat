package namat

import (
	"context"
	"fmt"
	"time"

	"github.com/nawafinity/go-namat/internal/expr"
)

// Function is a host function exposed to template expressions.
type Function func(args ...any) (any, error)

// QueryResolver resolves a QUERY command into the root data used by a render.
// It runs once before document commands are evaluated.
type QueryResolver func(ctx context.Context, query string) (any, error)

// ErrorHandler may replace the output of a failed value command. Returning a
// non-nil error aborts rendering.
type ErrorHandler func(command string, err error) (replacement any, returnedErr error)

// Options controls compilation and rendering.
type Options struct {
	OpenDelimiter        string
	CloseDelimiter       string
	LiteralXMLDelimiter  string
	Functions            map[string]Function
	QueryResolver        QueryResolver
	ErrorHandler         ErrorHandler
	CollectErrors        bool
	RejectNullish        bool
	AllowObjectResults   bool
	FixSmartQuotes       bool
	DisableLineBreaks    bool
	AllowRawXML          bool
	AllowedLinkSchemes   []string
	MaxIterations        int
	MaxTemplateBytes     int64
	MaxPartBytes         int64
	MaxUncompressedBytes int64
	MaxPackageParts      int
	MaxOutputBytes       int64
	CompressionLevel     int
	Timeout              time.Duration
}

func (o Options) normalized() Options {
	if o.Functions != nil {
		functions := make(map[string]Function, len(o.Functions))
		for name, function := range o.Functions {
			functions[name] = function
		}
		o.Functions = functions
	}
	if o.AllowedLinkSchemes != nil {
		o.AllowedLinkSchemes = append([]string(nil), o.AllowedLinkSchemes...)
	}
	if o.OpenDelimiter == "" {
		o.OpenDelimiter = "[["
	}
	if o.CloseDelimiter == "" {
		o.CloseDelimiter = "]]"
	}
	if o.MaxIterations <= 0 {
		o.MaxIterations = 1_000_000
	}
	if o.LiteralXMLDelimiter == "" {
		o.LiteralXMLDelimiter = "||"
	}
	if o.MaxOutputBytes <= 0 {
		o.MaxOutputBytes = 512 << 20
	}
	if o.MaxTemplateBytes <= 0 {
		o.MaxTemplateBytes = 256 << 20
	}
	if o.MaxPartBytes <= 0 {
		o.MaxPartBytes = 256 << 20
	}
	if o.MaxUncompressedBytes <= 0 {
		o.MaxUncompressedBytes = 1 << 30
	}
	if o.MaxPackageParts <= 0 {
		o.MaxPackageParts = 10_000
	}
	if o.CompressionLevel <= 0 {
		o.CompressionLevel = 1
	}
	if o.CompressionLevel > 9 {
		o.CompressionLevel = 9
	}
	if len(o.AllowedLinkSchemes) == 0 {
		o.AllowedLinkSchemes = []string{"http", "https", "mailto"}
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
