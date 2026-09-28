// Package namat renders data into Microsoft Word DOCX and DOCM templates using
// a pure Go engine and a bounded expression language.
//
// The package root is a small, stable public facade. Implementation details
// live under internal so applications cannot depend on private OOXML,
// rendering, or expression-engine behavior.
package namat

import (
	"context"
	"errors"
	"io"

	"github.com/nawafinity/go-namat/internal/engine"
)

// Template is an immutable compiled DOCX template. It is safe to render from
// multiple goroutines.
type Template struct {
	compiled *engine.Template
}

// Compile validates and compiles a DOCX template.
func Compile(template []byte, options Options) (*Template, error) {
	compiled, err := engine.Compile(template, options.internal())
	if err != nil {
		return nil, publicError(err)
	}
	return &Template{compiled: compiled}, nil
}

// CompileReader reads and compiles a DOCX template. Input is bounded by
// Options.MaxTemplateBytes.
func CompileReader(reader io.Reader, options Options) (*Template, error) {
	compiled, err := engine.CompileReader(reader, options.internal())
	if err != nil {
		return nil, publicError(err)
	}
	return &Template{compiled: compiled}, nil
}

// CreateReport compiles and renders a template in one call.
func CreateReport(ctx context.Context, template []byte, data any, options Options) ([]byte, error) {
	report, err := engine.CreateReport(ctx, template, data, options.internal())
	return report, publicError(err)
}

// CreateReportReader compiles a template from a reader and renders it.
func CreateReportReader(ctx context.Context, template io.Reader, data any, options Options) ([]byte, error) {
	report, err := engine.CreateReportReader(ctx, template, data, options.internal())
	return report, publicError(err)
}

// Render produces a DOCX report using the supplied data.
func (t *Template) Render(ctx context.Context, data any) ([]byte, error) {
	report, err := t.compiled.Render(ctx, data)
	return report, publicError(err)
}

// RenderTo writes a rendered DOCX to writer.
func (t *Template) RenderTo(ctx context.Context, writer io.Writer, data any) error {
	return publicError(t.compiled.RenderTo(ctx, writer, data))
}

// Commands returns a copy of all commands in package order.
func (t *Template) Commands() []Command {
	if t == nil || t.compiled == nil {
		return nil
	}
	commands := t.compiled.Commands()
	result := make([]Command, len(commands))
	for index, command := range commands {
		result[index] = publicCommand(command)
	}
	return result
}

// ListCommands returns commands in document order from all templated parts.
func ListCommands(template []byte, options Options) ([]Command, error) {
	commands, err := engine.ListCommands(template, options.internal())
	if err != nil {
		return nil, publicError(err)
	}
	result := make([]Command, len(commands))
	for index, command := range commands {
		result[index] = publicCommand(command)
	}
	return result, nil
}

// GetMetadata reads cached metadata fields from a DOCX or DOCM package.
func GetMetadata(document []byte) (Metadata, error) {
	metadata, err := engine.GetMetadata(document)
	if err != nil {
		return Metadata{}, publicError(err)
	}
	return Metadata{
		Title:                metadata.Title,
		Subject:              metadata.Subject,
		Creator:              metadata.Creator,
		Keywords:             metadata.Keywords,
		Description:          metadata.Description,
		LastModifiedBy:       metadata.LastModifiedBy,
		Revision:             metadata.Revision,
		Category:             metadata.Category,
		ContentStatus:        metadata.ContentStatus,
		Company:              metadata.Company,
		Template:             metadata.Template,
		Application:          metadata.Application,
		AppVersion:           metadata.AppVersion,
		Created:              metadata.Created,
		Modified:             metadata.Modified,
		LastPrinted:          metadata.LastPrinted,
		Pages:                metadata.Pages,
		Words:                metadata.Words,
		Characters:           metadata.Characters,
		CharactersWithSpaces: metadata.CharactersWithSpaces,
		Lines:                metadata.Lines,
		Paragraphs:           metadata.Paragraphs,
	}, nil
}

func (options Options) internal() engine.Options {
	var functions map[string]engine.Function
	if options.Functions != nil {
		functions = make(map[string]engine.Function, len(options.Functions))
		for name, function := range options.Functions {
			functions[name] = engine.Function(function)
		}
	}
	return engine.Options{
		OpenDelimiter:        options.OpenDelimiter,
		CloseDelimiter:       options.CloseDelimiter,
		LiteralXMLDelimiter:  options.LiteralXMLDelimiter,
		Functions:            functions,
		QueryResolver:        engine.QueryResolver(options.QueryResolver),
		ErrorHandler:         engine.ErrorHandler(options.ErrorHandler),
		CollectErrors:        options.CollectErrors,
		RejectNullish:        options.RejectNullish,
		AllowObjectResults:   options.AllowObjectResults,
		FixSmartQuotes:       options.FixSmartQuotes,
		DisableLineBreaks:    options.DisableLineBreaks,
		AllowRawXML:          options.AllowRawXML,
		AllowedLinkSchemes:   options.AllowedLinkSchemes,
		MaxIterations:        options.MaxIterations,
		MaxTemplateBytes:     options.MaxTemplateBytes,
		MaxPartBytes:         options.MaxPartBytes,
		MaxUncompressedBytes: options.MaxUncompressedBytes,
		MaxPackageParts:      options.MaxPackageParts,
		MaxOutputBytes:       options.MaxOutputBytes,
		CompressionLevel:     options.CompressionLevel,
		Timeout:              options.Timeout,
	}
}

func publicCommand(command engine.Command) Command {
	return Command{
		Raw:        command.Raw,
		Type:       CommandType(command.Type),
		Expression: command.Expression,
		Variable:   command.Variable,
	}
}

func publicError(err error) error {
	switch value := err.(type) {
	case nil:
		return nil
	case *engine.Error:
		return &Error{Part: value.Part, Command: value.Command, Err: publicError(value.Err)}
	case *engine.MultiError:
		errors := make([]error, len(value.Errors))
		for index, item := range value.Errors {
			errors[index] = publicError(item)
		}
		return &MultiError{Errors: errors}
	default:
		return publicCategory(err)
	}
}

type categorizedError struct {
	err      error
	category error
}

func (e categorizedError) Error() string   { return e.err.Error() }
func (e categorizedError) Unwrap() []error { return []error{e.err, e.category} }

func publicCategory(err error) error {
	categories := []struct {
		internal error
		public   error
	}{
		{engine.ErrInvalidTemplate, ErrInvalidTemplate},
		{engine.ErrCommandSyntax, ErrCommandSyntax},
		{engine.ErrCommandExecution, ErrCommandExecution},
		{engine.ErrNullishResult, ErrNullishResult},
		{engine.ErrObjectResult, ErrObjectResult},
		{engine.ErrSecurityLimit, ErrSecurityLimit},
	}
	for _, category := range categories {
		if errors.Is(err, category.internal) {
			return categorizedError{err: err, category: category.public}
		}
	}
	return err
}
