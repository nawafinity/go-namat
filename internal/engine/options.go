package engine

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
	// OpenDelimiter starts a template command. The default is "[[".
	OpenDelimiter string
	// CloseDelimiter ends a template command. The default is "]]".
	CloseDelimiter string
	// LiteralXMLDelimiter surrounds trusted OOXML fragments. The default is "||".
	LiteralXMLDelimiter string
	// Functions exposes explicitly registered Go functions to expressions.
	Functions map[string]Function
	// QueryResolver resolves the optional template-level QUERY command.
	QueryResolver QueryResolver
	// ErrorHandler may recover from a failed value command.
	ErrorHandler ErrorHandler
	// CollectErrors returns independent compile errors together.
	CollectErrors bool
	// RejectNullish rejects nil insertion results instead of writing empty text.
	RejectNullish bool
	// AllowObjectResults permits maps and structs to be formatted as text.
	AllowObjectResults bool
	// FixSmartQuotes normalizes typographic quotes before parsing expressions.
	FixSmartQuotes bool
	// DisableLineBreaks leaves newline characters as text instead of Word breaks.
	DisableLineBreaks bool
	// AllowRawXML enables RAW-XML and literal XML delimiters for trusted input.
	AllowRawXML bool
	// AllowedLinkSchemes is the allowlist used by LINK commands.
	AllowedLinkSchemes []string
	// MaxIterations limits aggregate loop iterations during one render.
	MaxIterations int
	// MaxTemplateBytes limits the compressed input template size.
	MaxTemplateBytes int64
	// MaxPartBytes limits one uncompressed ZIP part.
	MaxPartBytes int64
	// MaxUncompressedBytes limits aggregate uncompressed package content.
	MaxUncompressedBytes int64
	// MaxPackageParts limits the number of ZIP entries.
	MaxPackageParts int
	// MaxOutputBytes limits the final compressed document size.
	MaxOutputBytes int64
	// CompressionLevel selects ZIP deflate level 1 through 9.
	CompressionLevel int
	// Timeout bounds one render when the parent context has no earlier deadline.
	Timeout time.Duration
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
