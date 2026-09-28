// Package namat renders data into Microsoft Word DOCX templates using a pure
// Go engine and a bounded expression language.
package namat

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"

	"github.com/nawafinity/go-namat/internal/expr"
)

// Template is an immutable compiled DOCX template. It is safe to render from
// multiple goroutines.
type Template struct {
	options Options
	pkg     *docxPackage
	parts   map[string]*xmlNode
	cache   sync.Map
}

// Compile validates and compiles a DOCX template.
func Compile(template []byte, options Options) (*Template, error) {
	options = options.normalized()
	if options.OpenDelimiter == options.CloseDelimiter {
		return nil, fmt.Errorf("namat: open and close delimiters must differ")
	}
	pkg, err := readPackage(template)
	if err != nil {
		return nil, err
	}
	compiled := &Template{options: options, pkg: pkg, parts: make(map[string]*xmlNode)}
	for name, part := range pkg.Parts {
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
			if err := compiled.validateCommand(command); err != nil {
				return nil, &Error{Part: name, Command: command.Raw, Err: err}
			}
		}
		if err := validateStructure(root, options); err != nil {
			return nil, &Error{Part: name, Err: err}
		}
		compiled.parts[name] = root
	}
	return compiled, nil
}

// CreateReport compiles and renders a template in one call.
func CreateReport(ctx context.Context, template []byte, data any, options Options) ([]byte, error) {
	compiled, err := Compile(template, options)
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
	state := &renderState{
		ctx:       ctx,
		template:  t,
		rootData:  data,
		variables: make(map[string]any),
		functions: t.options.expressionFunctions(),
		counter:   &renderCounter{},
	}
	for _, name := range pkg.Order {
		source, ok := t.parts[name]
		if !ok {
			continue
		}
		root := source.clone()
		if err := state.processNode(root); err != nil {
			return nil, &Error{Part: name, Err: err}
		}
		content, err := root.bytes()
		if err != nil {
			return nil, &Error{Part: name, Err: fmt.Errorf("serialize XML: %w", err)}
		}
		pkg.Parts[name].Data = content
	}
	return pkg.bytes()
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
	switch command.Type {
	case CommandElse, CommandEndIf, CommandEndFor:
		return nil
	case CommandExec, CommandSet:
		_, expression, err := parseAssignment(command.Expression)
		if err != nil {
			return err
		}
		_, err = t.program(expression)
		return err
	case CommandImage, CommandLink, CommandHTML, CommandRawXML:
		// These are reserved now and implemented in subsequent compatibility layers.
		_, err := t.program(command.Expression)
		return err
	default:
		_, err := t.program(command.Expression)
		return err
	}
}

func (t *Template) program(source string) (*expr.Program, error) {
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
	if name == "" || strings.ContainsAny(name, " .[]()") {
		return "", "", fmt.Errorf("invalid assignment target %q", name)
	}
	return name, value, nil
}
