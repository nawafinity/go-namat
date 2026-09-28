package namat

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
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

// CommandType identifies a template command.
type CommandType string

const (
	// CommandInsert inserts an expression result as text.
	CommandInsert CommandType = "INS"
	// CommandExec evaluates an assignment without visible output.
	CommandExec CommandType = "EXEC"
	// CommandSet is an explicit assignment command.
	CommandSet CommandType = "SET"
	// CommandIf starts a conditional block.
	CommandIf CommandType = "IF"
	// CommandElse separates conditional branches.
	CommandElse CommandType = "ELSE"
	// CommandEndIf ends a conditional block.
	CommandEndIf CommandType = "END-IF"
	// CommandFor starts a collection loop.
	CommandFor CommandType = "FOR"
	// CommandEndFor ends a collection loop.
	CommandEndFor CommandType = "END-FOR"
	// CommandImage inserts an inline drawing.
	CommandImage CommandType = "IMAGE"
	// CommandLink inserts an external hyperlink.
	CommandLink CommandType = "LINK"
	// CommandHTML inserts an HTML altChunk in the main document.
	CommandHTML CommandType = "HTML"
	// CommandRawXML inserts trusted OOXML when explicitly enabled.
	CommandRawXML CommandType = "RAW-XML"
	// CommandQuery asks the host application to resolve root data.
	CommandQuery CommandType = "QUERY"
	// CommandAlias defines a reusable command.
	CommandAlias CommandType = "ALIAS"
	// CommandAliasRef invokes a previously defined alias.
	CommandAliasRef CommandType = "ALIAS-REF"
)

// Command is a parsed command found in a template.
type Command struct {
	// Raw is the command text without delimiters.
	Raw string
	// Type identifies the parsed command kind.
	Type CommandType
	// Expression contains the command expression or query text.
	Expression string
	// Variable contains a loop variable or alias name when applicable.
	Variable string
}

// Image describes an inline image returned by an IMAGE expression.
type Image struct {
	// Data contains encoded PNG, JPEG, GIF, or SVG bytes.
	Data []byte
	// Extension identifies the image format without requiring a leading dot.
	Extension string
	// Width is the rendered width in centimeters.
	Width float64
	// Height is the rendered height in centimeters.
	Height float64
	// Alt is the accessibility description stored in drawing properties.
	Alt string
	// Rotation is measured clockwise in degrees.
	Rotation float64
	// Caption is optional text emitted below the inline image.
	Caption string
	// Thumbnail is an optional raster fallback for an SVG image.
	Thumbnail *Image
}

// Link describes an external hyperlink returned by a LINK expression.
type Link struct {
	// URL is an absolute external URL whose scheme must be allowed by Options.
	URL string
	// Label is the visible link text; an empty label falls back to URL.
	Label string
	// Tooltip is optional hover text stored in the hyperlink element.
	Tooltip string
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
	Part    string
	Command string
	Err     error
}

func (e *Error) Error() string {
	switch {
	case e.Part != "" && e.Command != "":
		return fmt.Sprintf("namat: %s: command %q: %v", e.Part, e.Command, e.Err)
	case e.Part != "":
		return fmt.Sprintf("namat: %s: %v", e.Part, e.Err)
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
