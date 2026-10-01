package engine

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/nawafinity/go-namat/internal/expr"
	"github.com/nawafinity/go-namat/internal/value"
)

// FunctionSpec describes and implements a host function.
type FunctionSpec struct {
	Params      []value.Kind
	Variadic    bool
	Returns     value.Kind
	Description string
	Pure        bool
	Call        func(context.Context, ...any) (any, error)
}

// ErrorHandler may replace the output of a failed value command. Returning a
// non-nil error aborts rendering.
type ErrorHandler func(command string, err error) (replacement any, returnedErr error)

// Options controls compilation and rendering.
type Options struct {
	// LanguageVersion selects the template contract. Empty means "v1".
	LanguageVersion string
	// OpenDelimiter starts a template command. The default is "[[".
	OpenDelimiter string
	// CloseDelimiter ends a template command. The default is "]]".
	CloseDelimiter string
	// Functions exposes explicitly registered Go functions to expressions.
	Functions map[string]FunctionSpec
	// ErrorHandler may recover from a failed value command.
	ErrorHandler ErrorHandler
	// CollectErrors returns independent compile errors together.
	CollectErrors bool
	// FixSmartQuotes normalizes typographic quotes before parsing expressions.
	FixSmartQuotes bool
	// DisableLineBreaks leaves newline characters as text instead of Word breaks.
	DisableLineBreaks bool
	// AllowRawXML enables the explicit @raw-xml directive for trusted input.
	AllowRawXML bool
	// AllowedLinkSchemes is the allowlist used by LINK commands.
	AllowedLinkSchemes []string
	// MaxIterations limits aggregate loop iterations during one render.
	MaxIterations       int
	MaxExpressionBytes  int
	MaxExpressionTokens int
	MaxExpressionDepth  int
	MaxEvaluationSteps  int
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
	if o.LanguageVersion == "" {
		o.LanguageVersion = "v1"
	}
	if o.Functions != nil {
		functions := make(map[string]FunctionSpec, len(o.Functions))
		for name, function := range o.Functions {
			function.Params = append([]value.Kind(nil), function.Params...)
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
	if o.MaxExpressionBytes <= 0 {
		o.MaxExpressionBytes = 16 << 10
	}
	if o.MaxExpressionTokens <= 0 {
		o.MaxExpressionTokens = 2_048
	}
	if o.MaxExpressionDepth <= 0 {
		o.MaxExpressionDepth = 128
	}
	if o.MaxEvaluationSteps <= 0 {
		o.MaxEvaluationSteps = 100_000
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
	for name, spec := range o.Functions {
		functions[name] = expr.Function{
			Signature: expr.Signature{Params: append([]value.Kind(nil), spec.Params...), Variadic: spec.Variadic, Returns: spec.Returns, Description: spec.Description, Pure: spec.Pure},
			Call:      spec.Call,
		}
	}
	return functions
}

func (o Options) expressionSignatures() map[string]expr.Signature {
	functions := o.expressionFunctions()
	result := make(map[string]expr.Signature, len(functions))
	for name, function := range functions {
		result[name] = function.Signature
	}
	return result
}

func builtinFunctions() map[string]expr.Function {
	return map[string]expr.Function{
		"len": builtin([]value.Kind{value.Any}, value.Int, func(_ context.Context, args ...any) (any, error) {
			length, err := collectionLength(args[0])
			return int64(length), err
		}),
		"string": builtin([]value.Kind{value.Any}, value.String, func(_ context.Context, args ...any) (any, error) {
			return expr.Format(args[0])
		}),
		"trim": builtin([]value.Kind{value.String}, value.String, func(_ context.Context, args ...any) (any, error) {
			return strings.TrimSpace(args[0].(string)), nil
		}),
		"upper": builtin([]value.Kind{value.String}, value.String, func(_ context.Context, args ...any) (any, error) {
			return strings.ToUpper(args[0].(string)), nil
		}),
		"lower": builtin([]value.Kind{value.String}, value.String, func(_ context.Context, args ...any) (any, error) {
			return strings.ToLower(args[0].(string)), nil
		}),
		"default": builtin([]value.Kind{value.Any, value.Any}, value.Any, func(_ context.Context, args ...any) (any, error) {
			if expr.KindOf(args[0]) == value.Missing || expr.KindOf(args[0]) == value.Null {
				return args[1], nil
			}
			return args[0], nil
		}),
		"exists": builtin([]value.Kind{value.Any}, value.Bool, func(_ context.Context, args ...any) (any, error) {
			kind := expr.KindOf(args[0])
			return kind != value.Missing && kind != value.Null, nil
		}),
		"empty": builtin([]value.Kind{value.Any}, value.Bool, func(_ context.Context, args ...any) (any, error) {
			if expr.KindOf(args[0]) == value.Missing || expr.KindOf(args[0]) == value.Null {
				return true, nil
			}
			reflected := reflect.ValueOf(args[0])
			for reflected.IsValid() && (reflected.Kind() == reflect.Pointer || reflected.Kind() == reflect.Interface) {
				if reflected.IsNil() {
					return true, nil
				}
				reflected = reflected.Elem()
			}
			switch reflected.Kind() {
			case reflect.String, reflect.Array, reflect.Slice, reflect.Map:
				return reflected.Len() == 0, nil
			default:
				return false, fmt.Errorf("empty does not support %T", args[0])
			}
		}),
		"join": builtin([]value.Kind{value.List, value.String}, value.String, func(_ context.Context, args ...any) (any, error) {
			reflected := reflect.ValueOf(args[0])
			for reflected.Kind() == reflect.Pointer || reflected.Kind() == reflect.Interface {
				reflected = reflected.Elem()
			}
			items := make([]string, reflected.Len())
			for index := range items {
				formatted, err := expr.Format(reflected.Index(index).Interface())
				if err != nil {
					return nil, err
				}
				items[index] = formatted
			}
			return strings.Join(items, args[1].(string)), nil
		}),
	}
}

func builtin(params []value.Kind, returns value.Kind, call func(context.Context, ...any) (any, error)) expr.Function {
	return expr.Function{Signature: expr.Signature{Params: params, Returns: returns}, Call: call}
}

func renderContext(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if parent == nil {
		parent = context.Background()
	}
	return context.WithTimeout(parent, timeout)
}
