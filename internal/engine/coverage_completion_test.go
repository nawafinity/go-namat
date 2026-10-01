package engine

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/nawafinity/go-namat/internal/value"
)

func TestBuiltinFunctionsBehaviorAndErrors(t *testing.T) {
	functions := builtinFunctions()
	call := func(name string, args ...any) (any, error) {
		t.Helper()
		return functions[name].Call(context.Background(), args...)
	}
	functionFor := map[string]string{
		"trim": "trim", "upper": "upper", "lower": "lower", "default missing": "default", "default null": "default", "default value": "default",
		"exists missing": "exists", "exists null": "exists", "exists value": "exists", "empty missing": "empty", "empty null": "empty", "empty typed nil pointer": "empty",
		"empty pointer": "empty", "empty string": "empty", "empty array": "empty", "empty slice": "empty", "empty map": "empty", "nonempty string": "empty",
		"nonempty slice": "empty", "nonempty map": "empty", "empty scalar": "empty", "join": "join", "join array": "join", "join pointer": "join", "join empty": "join",
	}

	for _, test := range []struct {
		name string
		args []any
		want any
	}{
		{"trim", []any{"  hello \n"}, "hello"},
		{"upper", []any{"Hello"}, "HELLO"},
		{"lower", []any{"Hello"}, "hello"},
		{"default missing", []any{value.MissingValue{}, "fallback"}, "fallback"},
		{"default null", []any{nil, "fallback"}, "fallback"},
		{"default value", []any{"present", "fallback"}, "present"},
		{"exists missing", []any{value.MissingValue{}}, false},
		{"exists null", []any{nil}, false},
		{"exists value", []any{0}, true},
		{"empty missing", []any{value.MissingValue{}}, true},
		{"empty null", []any{nil}, true},
		{"empty typed nil pointer", []any{(*string)(nil)}, true},
		{"empty pointer", []any{func() *string { s := ""; return &s }()}, true},
		{"empty string", []any{""}, true},
		{"empty array", []any{[0]int{}}, true},
		{"empty slice", []any{[]int{}}, true},
		{"empty map", []any{map[string]int{}}, true},
		{"nonempty string", []any{"x"}, false},
		{"nonempty slice", []any{[]int{1}}, false},
		{"nonempty map", []any{map[string]int{"x": 1}}, false},
		{"join", []any{[]any{"a", 2, true}, ","}, "a,2,true"},
		{"join array", []any{[2]string{"a", "b"}, "/"}, "a/b"},
		{"join pointer", []any{func() *[]string { values := []string{"a", "b"}; return &values }(), "-"}, "a-b"},
		{"join empty", []any{[]string{}, ","}, ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := call(functionFor[test.name], test.args...)
			if err != nil || !reflect.DeepEqual(got, test.want) {
				t.Fatalf("builtin result = %#v, %v; want %#v", got, err, test.want)
			}
		})
	}
	if _, err := call("len", 1); err == nil {
		t.Fatal("len accepted a scalar")
	}
	if _, err := call("empty", 1); err == nil || !strings.Contains(err.Error(), "empty does not support") {
		t.Fatalf("empty scalar error = %v", err)
	}
	if _, err := call("string", value.MissingValue{}); err == nil {
		t.Fatal("string accepted a missing value")
	}
	if _, err := call("join", []any{value.MissingValue{}}, ","); err == nil {
		t.Fatal("join accepted a missing list element")
	}
}

func TestCommandScannerAndTopLevelSplittingEdges(t *testing.T) {
	if left, right, ok := splitTopLevelKeyword("items as", "as"); !ok || left != "items" || right != "" {
		t.Fatalf("keyword with empty right side = %q, %q, %v", left, right, ok)
	}
	if _, _, ok := splitTopLevelKeyword("items asleep", "as"); ok {
		t.Fatal("keyword matched a prefix of a word")
	}
	if _, _, ok := splitTopLevelKeyword("items other as item", "as"); !ok {
		t.Fatal("top-level keyword after a different word was not found")
	}
	if left, right, ok := splitTopLevelKeyword("call('a \\\" as \\\" b') as item", "as"); !ok || left != `call('a \" as \" b')` || right != "item" {
		t.Fatalf("escaped quoted keyword = %q, %q, %v", left, right, ok)
	}
	if left, right, ok := splitTopLevelAssignment(`call("a\\\"=b") = value`); !ok || left != `call("a\\\"=b")` || right != "value" {
		t.Fatalf("assignment with escaped equals = %q, %q, %v", left, right, ok)
	}
	if left, right, ok := splitTopLevelAssignment("call(1 = 2)"); ok || left != "" || right != "" {
		t.Fatalf("nested assignment separator = %q, %q, %v", left, right, ok)
	}

	for _, source := range []string{`[[ value ]]tail`, `[[fn("]] ")]]`, `[[fn("x\" ]] y")]]`} {
		ranges := scanCommandRanges(source, "[[", "]]")
		if len(ranges) != 1 || !ranges[0].closed {
			t.Fatalf("scanCommandRanges(%q) = %#v", source, ranges)
		}
	}
	if ranges := scanCommandRanges("no command", "[[", "]]"); len(ranges) != 0 {
		t.Fatalf("plain text ranges = %#v", ranges)
	}
	if ranges := scanCommandRanges("[[fn([1, 2])]]", "[[", "]]"); len(ranges) != 1 || !ranges[0].closed {
		t.Fatalf("nested delimiters ranges = %#v", ranges)
	}
	if err := (&Error{Err: errors.New("failure")}).Error(); err != "namat: failure" {
		t.Fatalf("location-free error = %q", err)
	}
	if err := (&commandSyntaxLocationError{Err: errors.New("syntax")}).Unwrap(); err == nil || err.Error() != "syntax" {
		t.Fatalf("command syntax unwrap = %v", err)
	}
	for _, test := range []struct{ source, prefix string }{
		{"#if", "#if"},
		{"#eachx value", "#each"},
		{"#let ", "#let"},
	} {
		if _, err := commandArgument(test.source, test.prefix); err == nil {
			t.Fatalf("commandArgument(%q, %q) succeeded", test.source, test.prefix)
		}
	}
	if _, err := parseCommand("#let a-b = 1"); err == nil {
		t.Fatal("invalid later variable-name rune succeeded")
	}
	if _, err := parseCommand("#each"); err == nil {
		t.Fatal("each without an argument succeeded")
	}
}

func TestRemainingEngineFailurePaths(t *testing.T) {
	// Compile must stop at the first invalid expression unless error collection
	// is explicitly requested.
	badExpression := testDOCX(t, map[string][]byte{"word/document.xml": []byte(wordDocument("<w:p><w:r><w:t>[[1 +]]</w:t></w:r></w:p>"))})
	if _, err := Compile(badExpression, Options{}); err == nil {
		t.Fatal("invalid expression compiled")
	}

	document := testDOCX(t, map[string][]byte{"word/document.xml": []byte(wordDocument("<w:p><w:r><w:t>[[missing]]</w:t></w:r></w:p>"))})
	compiled, err := Compile(document, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if err := compiled.RenderTo(context.Background(), &strings.Builder{}, nil); err == nil {
		t.Fatal("RenderTo hid render failure")
	}

	if got := formatValue(value.MissingValue{}); got != "" {
		t.Fatalf("formatValue unsupported result = %q", got)
	}

	state := richTestState()
	if _, err := state.htmlNode(value.MissingValue{}); err == nil || !strings.Contains(err.Error(), "HTML value") {
		t.Fatalf("HTML formatting error = %v", err)
	}

	state = renderTestState(Options{}, nil)
	state.ctx = context.Background()
	state.counter.evaluationSteps = state.template.options.MaxEvaluationSteps
	if _, err := state.evaluate("true"); err == nil {
		t.Fatal("evaluation step exhaustion succeeded")
	}
}

func TestSequenceDeclarationAndRawValueFailures(t *testing.T) {
	state := renderTestState(Options{}, nil)
	if err := state.declare("answer", 42); err != nil {
		t.Fatal(err)
	}
	if err := state.declare("answer", 43); err == nil {
		t.Fatal("duplicate declaration succeeded")
	}
	if err := state.declareLoopValues("item", 1, map[string]any{"index": int64(0)}); err != nil {
		t.Fatal(err)
	}
	if err := state.declareLoopValues("answer", 1, map[string]any{}); err == nil {
		t.Fatal("loop item redeclaration succeeded")
	}

	for _, test := range []struct {
		name       string
		paragraphs []string
		data       any
		allowRaw   bool
	}{
		{"let expression", []string{"[[#let answer = missing]]"}, nil, false},
		{"let duplicate", []string{"[[#let answer = 1]]", "[[#let answer = 2]]"}, nil, false},
		{"raw XML missing value", []string{"[[@raw-xml value]]"}, map[string]any{"value": value.MissingValue{}}, true},
		{"loop redeclares loop metadata", []string{"[[#each values as loop]]", "body", "[[/each]]"}, map[string]any{"values": []int{1}}, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			state := renderTestState(Options{AllowRawXML: test.allowRaw}, test.data)
			children := make([]*xmlNode, 0, len(test.paragraphs))
			for _, paragraph := range test.paragraphs {
				children = append(children, commandParagraph(t, paragraph))
			}
			if _, err := state.processSequence(children); err == nil {
				t.Fatal("expected sequence failure")
			}
		})
	}
}

func TestErrorHandlerCannotReplaceObjectWithMissing(t *testing.T) {
	state := renderTestState(Options{ErrorHandler: func(string, error) (any, error) {
		return value.MissingValue{}, nil
	}}, map[string]any{"object": map[string]any{"field": "value"}})
	paragraph := commandParagraph(t, "[[object]]")
	if err := state.renderParagraph(paragraph); err == nil || !strings.Contains(err.Error(), "missing value cannot be rendered") {
		t.Fatalf("missing value error after object recovery = %v", err)
	}
}

type thresholdContext struct {
	context.Context
	calls int
	stop  int
}

func (c *thresholdContext) Err() error {
	c.calls++
	if c.calls >= c.stop {
		return context.Canceled
	}
	return nil
}

func (c *thresholdContext) Deadline() (time.Time, bool) { return time.Time{}, false }
func (c *thresholdContext) Done() <-chan struct{}       { return nil }
func (c *thresholdContext) Value(any) any               { return nil }

func TestEvaluateDetectsCancellationAfterEval(t *testing.T) {
	state := renderTestState(Options{}, nil)
	ctx := &thresholdContext{Context: context.Background(), stop: 3}
	state.ctx = ctx
	if _, err := state.evaluate("true"); !errors.Is(err, context.Canceled) {
		t.Fatalf("post-evaluation cancellation = %v after %d context checks", err, ctx.calls)
	}
}

func TestSplitTextMarkupRejectsNodeWithoutText(t *testing.T) {
	if _, err := splitTextMarkup(elementNode("t"), "first\nsecond"); err == nil {
		t.Fatal("splitting a text value into a node without text succeeded")
	}
	if nodes, err := splitTextMarkup(mustParseElement(t, "<w:t>before</w:t>"), "before\nafter"); err != nil || len(nodes) != 3 {
		t.Fatalf("split text markup = %#v, %v", nodes, err)
	}
	if nodes, err := splitTextMarkup(mustParseElement(t, "<w:t>before</w:t>"), "before\n\nafter"); err != nil || len(nodes) != 4 {
		t.Fatalf("split text markup with an empty line = %#v, %v", nodes, err)
	}
}
