// Package engine implements Namat's DOCX template compiler and renderer.
// The public API is exposed by the module-root namat package.
package engine

import (
	"context"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"
	"sync"

	"github.com/nawafinity/go-namat/internal/expr"
)

// Template is an immutable compiled DOCX template. It is safe to render from
// multiple goroutines.
type Template struct {
	options  Options
	pkg      *docxPackage
	parts    map[string]*xmlNode
	cache    sync.Map
	commands []Command
}

// Compile validates and compiles a DOCX template.
func Compile(template []byte, options Options) (*Template, error) {
	options = options.normalized()
	if options.LanguageVersion != "v1" {
		return nil, fmt.Errorf("namat: unsupported language version %q; only v1 is available", options.LanguageVersion)
	}
	if int64(len(template)) > options.MaxTemplateBytes {
		return nil, fmt.Errorf("namat: %w: template exceeds MaxTemplateBytes (%d)", ErrSecurityLimit, options.MaxTemplateBytes)
	}
	if options.OpenDelimiter == options.CloseDelimiter {
		return nil, fmt.Errorf("namat: open and close delimiters must differ")
	}
	pkg, err := readPackageWithLimits(template, options.MaxPartBytes, options.MaxUncompressedBytes, options.MaxPackageParts)
	if err != nil {
		return nil, err
	}
	compiled := &Template{options: options, pkg: pkg, parts: make(map[string]*xmlNode)}
	partCommands := make(map[string][]Command)
	var validationErrors []error
	for _, name := range pkg.Order {
		part := pkg.Parts[name]
		if !isTemplateXMLPart(name, part.Data, options.OpenDelimiter) {
			continue
		}
		root, err := parseXML(part.Data)
		if err != nil {
			return nil, &Error{Part: name, Err: fmt.Errorf("parse XML: %w", err)}
		}
		normalizeCommandFragments(root, options)
		commands, err := commandsInXML(root, options)
		if err != nil {
			return nil, commandSyntaxError(name, err)
		}
		if len(commands) == 0 {
			continue
		}
		compiled.parts[name] = root
		partCommands[name] = commands
		compiled.commands = append(compiled.commands, commands...)
	}
	for _, name := range pkg.Order {
		root, ok := compiled.parts[name]
		if !ok {
			continue
		}
		commands := partCommands[name]
		for _, command := range commands {
			if err := compiled.validateCommand(command); err != nil {
				wrapped := &Error{Part: name, Paragraph: command.Location.Paragraph, CommandIndex: command.Location.Ordinal, Start: command.Location.Start, End: command.Location.End, Command: command.Raw, Err: fmt.Errorf("%w: %v", ErrCommandSyntax, err)}
				if !options.CollectErrors {
					return nil, wrapped
				}
				validationErrors = append(validationErrors, wrapped)
			}
		}
		if err := validateStructure(root, options); err != nil {
			wrapped := &Error{Part: name, Err: fmt.Errorf("%w: %v", ErrCommandSyntax, err)}
			if !options.CollectErrors {
				return nil, wrapped
			}
			validationErrors = append(validationErrors, wrapped)
		}
	}
	if len(validationErrors) > 0 {
		return nil, &MultiError{Errors: validationErrors}
	}
	return compiled, nil
}

// CompileReader reads and compiles a DOCX template. Input is bounded by
// Options.MaxTemplateBytes.
func CompileReader(reader io.Reader, options Options) (*Template, error) {
	if reader == nil {
		return nil, fmt.Errorf("namat: template reader is nil")
	}
	normalized := options.normalized()
	content, err := io.ReadAll(io.LimitReader(reader, normalized.MaxTemplateBytes+1))
	if err != nil {
		return nil, fmt.Errorf("namat: read template: %w", err)
	}
	return Compile(content, normalized)
}

// CreateReport compiles and renders a template in one call.
func CreateReport(ctx context.Context, template []byte, data any, options Options) ([]byte, error) {
	compiled, err := Compile(template, options)
	if err != nil {
		return nil, err
	}
	return compiled.Render(ctx, data)
}

// CreateReportReader compiles a template from a reader and renders it.
func CreateReportReader(ctx context.Context, template io.Reader, data any, options Options) ([]byte, error) {
	compiled, err := CompileReader(template, options)
	if err != nil {
		return nil, err
	}
	return compiled.Render(ctx, data)
}

// Render produces a DOCX report using the supplied data.
func (t *Template) Render(parent context.Context, data any) ([]byte, error) {
	ctx, cancel := renderContext(parent, t.options.Timeout)
	defer cancel()
	pkg := t.pkg.clone()
	resourceCounter := maxDrawingIdentifier(t.parts)
	counter := &renderCounter{resources: resourceCounter}
	for _, name := range pkg.Order {
		source, ok := t.parts[name]
		if !ok {
			continue
		}
		root := source.clone()
		state := &renderState{
			ctx:       ctx,
			template:  t,
			rootData:  data,
			variables: make(map[string]any),
			declared:  make(map[string]struct{}),
			functions: t.options.expressionFunctions(),
			counter:   counter,
			pkg:       pkg,
			partName:  name,
		}
		if err := state.processNode(root); err != nil {
			var commandErr *Error
			if errors.As(err, &commandErr) && commandErr.Part == "" {
				copy := *commandErr
				copy.Part = name
				return nil, &copy
			}
			return nil, &Error{Part: name, Err: err}
		}
		pkg.Parts[name].Data = root.bytes()
	}
	if err := pkg.validateLimits(t.options.MaxPartBytes, t.options.MaxUncompressedBytes, t.options.MaxPackageParts); err != nil {
		return nil, err
	}
	result, err := pkg.bytesWithCompressionLimit(ctx, t.options.CompressionLevel, t.options.MaxOutputBytes)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func maxDrawingIdentifier(parts map[string]*xmlNode) int {
	maximum := 0
	for _, root := range parts {
		for _, node := range root.descendants("docPr") {
			value, err := strconv.Atoi(attributeValue(node, "id"))
			if err == nil && value > maximum {
				maximum = value
			}
		}
		for _, node := range root.descendants("cNvPr") {
			value, err := strconv.Atoi(attributeValue(node, "id"))
			if err == nil && value > maximum {
				maximum = value
			}
		}
	}
	return maximum
}

// RenderTo writes a rendered DOCX to writer.
func (t *Template) RenderTo(ctx context.Context, writer io.Writer, data any) error {
	if writer == nil {
		return fmt.Errorf("namat: output writer is nil")
	}
	report, err := t.Render(ctx, data)
	if err != nil {
		return err
	}
	written, err := writer.Write(report)
	if err != nil {
		return err
	}
	if written != len(report) {
		return io.ErrShortWrite
	}
	return nil
}

// Commands returns a copy of all commands in package order.
func (t *Template) Commands() []Command {
	if t == nil {
		return nil
	}
	return append([]Command(nil), t.commands...)
}

// ListCommands returns commands in document order from all templated parts.
func ListCommands(template []byte, options Options) ([]Command, error) {
	options = options.normalized()
	if int64(len(template)) > options.MaxTemplateBytes {
		return nil, fmt.Errorf("namat: %w: template exceeds MaxTemplateBytes (%d)", ErrSecurityLimit, options.MaxTemplateBytes)
	}
	pkg, err := readPackageWithLimits(template, options.MaxPartBytes, options.MaxUncompressedBytes, options.MaxPackageParts)
	if err != nil {
		return nil, err
	}
	var result []Command
	for _, name := range pkg.Order {
		part := pkg.Parts[name]
		if !isTemplateXMLPart(name, part.Data, options.OpenDelimiter) {
			continue
		}
		root, err := parseXML(part.Data)
		if err != nil {
			return nil, &Error{Part: name, Err: err}
		}
		normalizeCommandFragments(root, options)
		commands, err := commandsInXML(root, options)
		if err != nil {
			return nil, commandSyntaxError(name, err)
		}
		if len(commands) == 0 {
			continue
		}
		result = append(result, commands...)
	}
	return result, nil
}

func commandSyntaxError(part string, err error) error {
	result := &Error{Part: part, Err: fmt.Errorf("%w: %v", ErrCommandSyntax, err)}
	var located *commandSyntaxLocationError
	if errors.As(err, &located) {
		result.Paragraph = located.Location.Paragraph
		result.CommandIndex = located.Location.Ordinal
		result.Start = located.Location.Start
		result.End = located.Location.End
	}
	return result
}

func (t *Template) validateCommand(command Command) error {
	switch command.Type {
	case CommandElse, CommandEndIf, CommandEndEach:
		return nil
	case CommandLet, CommandImage, CommandLink, CommandHTML, CommandRawXML:
		// Rich-content expressions are compiled here and materialized during rendering.
		_, err := t.program(command.Expression)
		return err
	default:
		_, err := t.program(command.Expression)
		return err
	}
}

func (t *Template) program(source string) (*expr.Program, error) {
	if t.options.FixSmartQuotes {
		source = normalizeSmartQuotes(source)
	}
	if cached, ok := t.cache.Load(source); ok {
		return cached.(*expr.Program), nil
	}
	program, err := expr.CompileWithOptions(source, expr.CompileOptions{
		MaxBytes:  t.options.MaxExpressionBytes,
		MaxTokens: t.options.MaxExpressionTokens,
		MaxDepth:  t.options.MaxExpressionDepth,
		Functions: t.options.expressionSignatures(),
	})
	if err != nil {
		return nil, err
	}
	actual, _ := t.cache.LoadOrStore(source, program)
	return actual.(*expr.Program), nil
}

func normalizeSmartQuotes(source string) string {
	return strings.NewReplacer("‘", "'", "’", "'", "“", "\"", "”", "\"").Replace(source)
}

func collectionLength(value any) (int, error) {
	if value == nil {
		return 0, nil
	}
	v := reflect.ValueOf(value)
	for v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface {
		if v.IsNil() {
			return 0, nil
		}
		v = v.Elem()
	}
	switch v.Kind() {
	case reflect.Array, reflect.Slice, reflect.Map, reflect.String:
		if v.Kind() == reflect.String {
			return len([]rune(v.String())), nil
		}
		return v.Len(), nil
	default:
		return 0, fmt.Errorf("len does not support %T", value)
	}
}

func formatValue(value any) string {
	formatted, err := expr.Format(value)
	if err != nil {
		return ""
	}
	return formatted
}
