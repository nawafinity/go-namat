// Package namat renders data into Microsoft Word DOCX templates using a pure
// Go engine and a bounded expression language.
package namat

import (
	"context"
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
	aliases  map[string]Command
	queries  []string
	commands []Command
}

// Compile validates and compiles a DOCX template.
func Compile(template []byte, options Options) (*Template, error) {
	options = options.normalized()
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
	compiled := &Template{options: options, pkg: pkg, parts: make(map[string]*xmlNode), aliases: make(map[string]Command)}
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
			return nil, &Error{Part: name, Err: err}
		}
		for _, command := range commands {
			if command.Type == CommandAlias {
				if _, exists := compiled.aliases[command.Variable]; exists {
					return nil, &Error{Part: name, Command: command.Raw, Err: fmt.Errorf("duplicate alias %q", command.Variable)}
				}
				resolved, err := parseCommand(command.Expression)
				if err != nil {
					return nil, &Error{Part: name, Command: command.Raw, Err: err}
				}
				compiled.aliases[command.Variable] = resolved
			}
			if command.Type == CommandQuery {
				compiled.queries = append(compiled.queries, command.Expression)
			}
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
				wrapped := &Error{Part: name, Command: command.Raw, Err: fmt.Errorf("%w: %v", ErrCommandSyntax, err)}
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
	if len(t.queries) > 1 {
		return nil, fmt.Errorf("namat: template contains more than one QUERY command")
	}
	if len(t.queries) == 1 {
		if t.options.QueryResolver == nil {
			return nil, fmt.Errorf("namat: template contains QUERY but no QueryResolver was configured")
		}
		resolved, err := t.options.QueryResolver(ctx, t.queries[0])
		if err != nil {
			return nil, fmt.Errorf("namat: resolve query: %w", err)
		}
		data = resolved
	}

	pkg := t.pkg.clone()
	resourceCounter := maxDrawingIdentifier(t.parts)
	state := &renderState{
		ctx:       ctx,
		template:  t,
		rootData:  data,
		variables: make(map[string]any),
		functions: t.options.expressionFunctions(),
		counter:   &renderCounter{resources: resourceCounter},
		pkg:       pkg,
	}
	for _, name := range pkg.Order {
		source, ok := t.parts[name]
		if !ok {
			continue
		}
		root := source.clone()
		state.partName = name
		if err := state.processNode(root); err != nil {
			return nil, &Error{Part: name, Err: err}
		}
		content, err := root.bytes()
		if err != nil {
			return nil, &Error{Part: name, Err: fmt.Errorf("serialize XML: %w", err)}
		}
		pkg.Parts[name].Data = content
	}
	result, err := pkg.bytesWithCompression(t.options.CompressionLevel)
	if err != nil {
		return nil, err
	}
	if int64(len(result)) > t.options.MaxOutputBytes {
		return nil, fmt.Errorf("namat: %w: rendered document exceeds MaxOutputBytes (%d)", ErrSecurityLimit, t.options.MaxOutputBytes)
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
	_, err = writer.Write(report)
	return err
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
	pkg, err := readPackage(template)
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
			return nil, &Error{Part: name, Err: err}
		}
		result = append(result, commands...)
	}
	return result, nil
}

func (t *Template) validateCommand(command Command) error {
	if command.Type == CommandAliasRef {
		resolved, err := t.resolveCommand(command)
		if err != nil {
			return err
		}
		return t.validateCommand(resolved)
	}
	switch command.Type {
	case CommandElse, CommandEndIf, CommandEndFor, CommandQuery, CommandAlias:
		return nil
	case CommandExec, CommandSet:
		_, expression, err := parseAssignment(command.Expression)
		if err != nil {
			return err
		}
		_, err = t.program(expression)
		return err
	case CommandImage, CommandLink, CommandHTML, CommandRawXML:
		// Rich-content expressions are compiled here and materialized during rendering.
		_, err := t.program(command.Expression)
		return err
	default:
		_, err := t.program(command.Expression)
		return err
	}
}

func (t *Template) resolveCommand(command Command) (Command, error) {
	if command.Type != CommandAliasRef {
		return command, nil
	}
	resolved, ok := t.aliases[command.Variable]
	if !ok {
		return Command{}, fmt.Errorf("unknown alias %q", command.Variable)
	}
	resolved.Raw = command.Raw
	return resolved, nil
}

func (t *Template) program(source string) (*expr.Program, error) {
	if t.options.FixSmartQuotes {
		source = normalizeSmartQuotes(source)
	}
	if cached, ok := t.cache.Load(source); ok {
		return cached.(*expr.Program), nil
	}
	program, err := expr.Compile(source)
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
		return v.Len(), nil
	default:
		return 0, fmt.Errorf("len does not support %T", value)
	}
}

func formatValue(value any) string {
	if value == nil {
		return ""
	}
	switch v := value.(type) {
	case string:
		return v
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case float32:
		return strconv.FormatFloat(float64(v), 'f', -1, 32)
	default:
		return fmt.Sprint(v)
	}
}

func parseAssignment(source string) (string, string, error) {
	index := strings.Index(source, "=")
	if index <= 0 || index == len(source)-1 {
		return "", "", fmt.Errorf("assignment syntax is name = expression")
	}
	name := strings.TrimSpace(source[:index])
	value := strings.TrimSpace(source[index+1:])
	if !validName(strings.TrimPrefix(name, "$")) {
		return "", "", fmt.Errorf("invalid assignment target %q", name)
	}
	return name, value, nil
}
