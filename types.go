package namat

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/nawafinity/go-namat/internal/value"
)

// ValueType identifies a strict v1 expression type.
type ValueType = value.Kind

const (
	TypeAny     ValueType = value.Any
	TypeMissing ValueType = value.Missing
	TypeNull    ValueType = value.Null
	TypeBool    ValueType = value.Bool
	TypeString  ValueType = value.String
	TypeInt     ValueType = value.Int
	TypeUint    ValueType = value.Uint
	TypeDecimal ValueType = value.Decimal
	TypeFloat   ValueType = value.Float
	TypeList    ValueType = value.List
	TypeObject  ValueType = value.Object
	TypeRich    ValueType = value.Rich
)

// Decimal is an exact base-10/rational number suitable for money and other
// values that must not pass through float64.
type Decimal = value.DecimalValue

func ParseDecimal(source string) (Decimal, error) { return value.ParseDecimal(source) }

const LanguageVersionV1 = "v1"

// FunctionSpec describes a host function and its compile-time signature.
type FunctionSpec struct {
	Params      []ValueType
	Variadic    bool
	Returns     ValueType
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
	// Functions exposes explicitly registered, typed Go functions.
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
	MaxIterations int
	// MaxExpressionBytes limits the source length of one expression.
	MaxExpressionBytes int
	// MaxExpressionTokens limits lexical complexity of one expression.
	MaxExpressionTokens int
	// MaxExpressionDepth limits parser and AST nesting.
	MaxExpressionDepth int
	// MaxEvaluationSteps limits aggregate expression work during one render.
	MaxEvaluationSteps int
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

// CommandType identifies a template command.
type CommandType string

const (
	CommandInsert  CommandType = "insert"
	CommandLet     CommandType = "let"
	CommandIf      CommandType = "if"
	CommandElse    CommandType = "else"
	CommandEndIf   CommandType = "/if"
	CommandEach    CommandType = "each"
	CommandEndEach CommandType = "/each"
	CommandImage   CommandType = "@image"
	CommandLink    CommandType = "@link"
	CommandHTML    CommandType = "@html"
	CommandRawXML  CommandType = "@raw-xml"
)

// Command is a parsed command found in a template.
type Command struct {
	// Raw is the command text without delimiters.
	Raw string
	// Type identifies the parsed command kind.
	Type CommandType
	// Expression contains the command expression.
	Expression string
	// Variable contains a loop variable or lexical declaration name.
	Variable string
	Location CommandLocation
}

type CommandLocation struct {
	Paragraph int
	Ordinal   int
	Start     int
	End       int
}

// Image describes an inline image returned by an IMAGE expression.
type Image struct {
	// Data contains encoded PNG, JPEG, GIF, or SVG bytes.
	Data []byte `json:"data"`
	// Extension identifies the image format without requiring a leading dot.
	Extension string `json:"extension"`
	// Width is the rendered width in centimeters.
	Width float64 `json:"width"`
	// Height is the rendered height in centimeters.
	Height float64 `json:"height"`
	// Alt is the accessibility description stored in drawing properties.
	Alt string `json:"alt"`
	// Rotation is measured clockwise in degrees.
	Rotation float64 `json:"rotation"`
	// Caption is optional text emitted below the inline image.
	Caption string `json:"caption"`
	// Thumbnail is an optional raster fallback for an SVG image.
	Thumbnail *Image `json:"thumbnail,omitempty"`
}

// Link describes an external hyperlink returned by a LINK expression.
type Link struct {
	// URL is an absolute external URL whose scheme must be allowed by Options.
	URL string `json:"url"`
	// Label is the visible link text; an empty label falls back to URL.
	Label string `json:"label"`
	// Tooltip is optional hover text stored in the hyperlink element.
	Tooltip string `json:"tooltip"`
}

// Metadata contains standard Word core and extended document properties.
// Values such as Pages and Words are cached by Word and are not recalculated
// by Namat.
type Metadata struct {
	Title                string
	Subject              string
	Creator              string
	Keywords             string
	Description          string
	LastModifiedBy       string
	Revision             string
	Category             string
	ContentStatus        string
	Company              string
	Template             string
	Application          string
	AppVersion           string
	Created              *time.Time
	Modified             *time.Time
	LastPrinted          *time.Time
	Pages                int
	Words                int
	Characters           int
	CharactersWithSpaces int
	Lines                int
	Paragraphs           int
}

var (
	// ErrInvalidTemplate identifies an invalid DOCX package or template shape.
	ErrInvalidTemplate = errors.New("invalid template")
	// ErrCommandSyntax identifies a malformed command or expression.
	ErrCommandSyntax = errors.New("command syntax error")
	// ErrCommandExecution identifies a command that failed during rendering.
	ErrCommandExecution = errors.New("command execution error")
	// ErrNullishResult identifies a rejected nil command result.
	ErrNullishResult = errors.New("nullish command result")
	// ErrObjectResult identifies an object accidentally used as text.
	ErrObjectResult = errors.New("object command result")
	// ErrSecurityLimit identifies a configured resource or safety limit.
	ErrSecurityLimit = errors.New("security limit exceeded")
)

// Error describes a template failure and the package part where it occurred.
type Error struct {
	Part         string
	Paragraph    int
	CommandIndex int
	Start        int
	End          int
	Command      string
	Err          error
}

func (e *Error) Error() string {
	location := e.Part
	if e.Paragraph > 0 {
		location += fmt.Sprintf(": paragraph %d", e.Paragraph)
	}
	if e.CommandIndex > 0 {
		location += fmt.Sprintf(": command %d", e.CommandIndex)
	}
	switch {
	case location != "" && e.Command != "":
		return fmt.Sprintf("namat: %s: %q: %v", location, e.Command, e.Err)
	case location != "":
		return fmt.Sprintf("namat: %s: %v", location, e.Err)
	default:
		return fmt.Sprintf("namat: %v", e.Err)
	}
}

func (e *Error) Unwrap() error { return e.Err }

// MultiError contains independent validation errors found when CollectErrors
// is enabled.
type MultiError struct {
	Errors []error
}

func (e *MultiError) Error() string {
	if e == nil || len(e.Errors) == 0 {
		return "namat: no errors"
	}
	parts := make([]string, len(e.Errors))
	for index, err := range e.Errors {
		parts[index] = err.Error()
	}
	return fmt.Sprintf("namat: %d validation errors: %s", len(parts), strings.Join(parts, "; "))
}

// Unwrap allows errors.Is and errors.As to inspect every contained error.
func (e *MultiError) Unwrap() []error {
	if e == nil {
		return nil
	}
	return e.Errors
}
