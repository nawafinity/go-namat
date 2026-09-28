package expr

import (
	"errors"
	"fmt"
	"math"
	"reflect"
	"strings"
	"testing"
)

type failingNode struct{ err error }

func (n failingNode) eval(*Context) (any, error) { return nil, n.err }

type stringValue string

func (v stringValue) String() string { return string(v) }

func TestInternalNodeErrorPropagation(t *testing.T) {
	wantErr := errors.New("sentinel")
	fail := failingNode{err: wantErr}
	ctx := &Context{}
	tests := []struct {
		name string
		node node
	}{
		{"array item", arrayNode{values: []node{fail}}},
		{"object value", objectNode{entries: []objectEntry{{key: "key", value: fail}}}},
		{"conditional condition", conditionalNode{condition: fail, whenTrue: literalNode{}, whenFalse: literalNode{}}},
		{"member target", memberNode{target: fail, key: literalNode{value: "key"}}},
		{"member key", memberNode{target: literalNode{value: map[string]any{}}, key: fail}},
		{"call callee", callNode{callee: fail}},
		{"call argument", callNode{callee: literalNode{value: Function(func(...any) (any, error) { return nil, nil })}, args: []node{fail}}},
		{"unary value", unaryNode{op: "!", value: fail}},
		{"binary left", binaryNode{op: "+", left: fail, right: literalNode{value: 1}}},
		{"binary right", binaryNode{op: "+", left: literalNode{value: 1}, right: fail}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := test.node.eval(ctx)
			if !errors.Is(err, wantErr) {
				t.Fatalf("error = %v, want sentinel", err)
			}
		})
	}
}

func TestIdentifierAndMemberEdgeCases(t *testing.T) {
	fn := Function(func(...any) (any, error) { return "function", nil })
	ctx := &Context{
		Root:      map[string]any{"root": "root value"},
		Variables: map[string]any{"variable": "variable value"},
		Functions: map[string]Function{"function": fn},
	}
	for name, want := range map[string]any{
		"variable": "variable value",
		"function": fn,
		"root":     "root value",
	} {
		got, err := (identifierNode{name: name}).eval(ctx)
		if err != nil {
			t.Fatalf("identifier %q: %v", name, err)
		}
		if name == "function" {
			if reflect.ValueOf(got).Pointer() != reflect.ValueOf(want).Pointer() {
				t.Fatalf("identifier %q returned a different function", name)
			}
		} else if got != want {
			t.Fatalf("identifier %q = %#v, want %#v", name, got, want)
		}
	}

	_, err := (identifierNode{name: "missing"}).eval(ctx)
	var unknown unknownIdentifierError
	if !errors.As(err, &unknown) || unknown.Error() != "namat expression: unknown identifier \"missing\"" {
		t.Fatalf("unknown identifier error = %v", err)
	}

	tests := []struct {
		name    string
		member  memberNode
		want    any
		wantErr bool
	}{
		{
			name:   "optional unknown target",
			member: memberNode{target: identifierNode{name: "missing"}, key: literalNode{value: "key"}, optional: true},
		},
		{
			name:   "optional nil target",
			member: memberNode{target: literalNode{value: nil}, key: literalNode{value: "key"}, optional: true},
		},
		{
			name:    "required nil target",
			member:  memberNode{target: literalNode{value: nil}, key: literalNode{value: "key"}},
			wantErr: true,
		},
		{
			name:   "optional missing property",
			member: memberNode{target: literalNode{value: map[string]any{}}, key: literalNode{value: "key"}, optional: true},
		},
		{
			name:    "required missing property",
			member:  memberNode{target: literalNode{value: map[string]any{}}, key: literalNode{value: "key"}},
			wantErr: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := test.member.eval(ctx)
			if test.wantErr && err == nil {
				t.Fatal("expected an error")
			}
			if !test.wantErr && (err != nil || got != test.want) {
				t.Fatalf("got %#v, %v; want %#v", got, err, test.want)
			}
		})
	}
}

func TestProgramAcceptsNilContext(t *testing.T) {
	program, err := Compile("true")
	if err != nil {
		t.Fatal(err)
	}
	got, err := program.Eval(nil)
	if err != nil || got != true {
		t.Fatalf("Eval(nil) = %#v, %v", got, err)
	}
}

func TestBuiltInMemberErrorsAndBounds(t *testing.T) {
	call := func(target any, name string, args ...any) (any, error) {
		t.Helper()
		member, ok := builtinMember(target, name)
		if !ok {
			t.Fatalf("builtinMember(%T, %q) was not found", target, name)
		}
		return member.(Function)(args...)
	}

	errorCases := []struct {
		name   string
		target any
		member string
		args   []any
	}{
		{"trim arguments", " value ", "trim", []any{1}},
		{"upper arguments", "value", "upper", []any{1}},
		{"lower arguments", "value", "lower", []any{1}},
		{"contains arguments", "value", "contains", nil},
		{"startsWith arguments", "value", "startsWith", nil},
		{"endsWith arguments", "value", "endsWith", nil},
		{"slice no arguments", "value", "slice", nil},
		{"slice too many arguments", "value", "slice", []any{0, 1, 2}},
		{"slice invalid start", "value", "slice", []any{"bad"}},
		{"slice invalid end", "value", "slice", []any{0, "bad"}},
		{"join arguments", []int{1}, "join", []any{",", ";"}},
		{"includes arguments", []int{1}, "includes", nil},
	}
	for _, test := range errorCases {
		t.Run(test.name, func(t *testing.T) {
			if _, err := call(test.target, test.member, test.args...); err == nil {
				t.Fatal("expected an argument validation error")
			}
		})
	}

	if got, err := call([]int{1, 2}, "join", "-"); err != nil || got != "1-2" {
		t.Fatalf("join = %#v, %v", got, err)
	}
	if got, err := call([]int{1, 2}, "includes", 2); err != nil || got != true {
		t.Fatalf("includes = %#v, %v", got, err)
	}
	if got, ok := builtinMember(map[string]int{"a": 1}, "length"); !ok || got != 1 {
		t.Fatalf("map length = %#v, %v", got, ok)
	}
	var nilPointer *string
	value := "value"
	if got, ok := builtinMember(&value, "length"); !ok || got != 5 {
		t.Fatalf("pointer string length = %#v, %v", got, ok)
	}
	for _, test := range []struct {
		target any
		name   string
	}{
		{nil, "length"},
		{nilPointer, "length"},
		{struct{}{}, "length"},
		{"value", "missing"},
	} {
		if _, ok := builtinMember(test.target, test.name); ok {
			t.Fatalf("unexpected member %q on %T", test.name, test.target)
		}
	}

	bounds := []struct {
		start, end, length int
		wantStart, wantEnd int
	}{
		{-2, -1, 5, 3, 4},
		{-9, 2, 5, 0, 2},
		{9, 10, 5, 5, 5},
		{4, 2, 5, 4, 4},
		{1, 9, 5, 1, 5},
	}
	for _, test := range bounds {
		start, end := normalizeSliceBounds(test.start, test.end, test.length)
		if start != test.wantStart || end != test.wantEnd {
			t.Fatalf("normalizeSliceBounds(%d, %d, %d) = %d, %d; want %d, %d", test.start, test.end, test.length, start, end, test.wantStart, test.wantEnd)
		}
	}
}

func TestDirectOperatorFailureBranches(t *testing.T) {
	ctx := &Context{}
	for _, test := range []struct {
		name string
		node node
	}{
		{"unary minus type", unaryNode{op: "-", value: literalNode{value: "x"}}},
		{"unary plus type", unaryNode{op: "+", value: literalNode{value: "x"}}},
		{"unary unsupported", unaryNode{op: "~", value: literalNode{value: 1}}},
		{"binary numeric type", binaryNode{op: "-", left: literalNode{value: "x"}, right: literalNode{value: 1}}},
		{"binary unsupported", binaryNode{op: "**", left: literalNode{value: 2}, right: literalNode{value: 3}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := test.node.eval(ctx); err == nil {
				t.Fatal("expected an operator error")
			}
		})
	}

	shortCircuits := []struct {
		name string
		node binaryNode
		want any
	}{
		{"coalescing", binaryNode{op: "??", left: literalNode{value: "left"}, right: failingNode{err: errors.New("right evaluated")}}, "left"},
		{"or", binaryNode{op: "||", left: literalNode{value: "left"}, right: failingNode{err: errors.New("right evaluated")}}, "left"},
		{"and", binaryNode{op: "&&", left: literalNode{value: false}, right: failingNode{err: errors.New("right evaluated")}}, false},
	}
	for _, test := range shortCircuits {
		got, err := test.node.eval(ctx)
		if err != nil || got != test.want {
			t.Fatalf("%s = %#v, %v; want %#v", test.name, got, err, test.want)
		}
	}

	for _, op := range []string{">", ">=", "<", "<="} {
		if _, err := compare(op, "b", "a"); err != nil {
			t.Fatalf("string compare %q: %v", op, err)
		}
	}
	if _, err := compare("?", "a", "b"); err == nil {
		t.Fatal("unsupported comparison succeeded")
	}
}

func TestTemplateInterpolationInternals(t *testing.T) {
	ctx := &Context{Root: map[string]any{"name": "Namat"}}
	for _, test := range []struct {
		name    string
		text    string
		want    string
		wantErr bool
	}{
		{"plain text", "plain", "plain", false},
		{"interpolation", "Hello ${name}!", "Hello Namat!", false},
		{"compile error", "${@}", "", true},
		{"evaluation error", "${missing}", "", true},
		{"unterminated interpolation", "${name", "", true},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := (templateNode{text: test.text}).eval(ctx)
			if test.wantErr {
				if err == nil {
					t.Fatal("expected an error")
				}
				return
			}
			if err != nil || got != test.want {
				t.Fatalf("got %#v, %v; want %q", got, err, test.want)
			}
		})
	}

	for _, test := range []struct {
		text  string
		start int
		want  int
	}{
		{"{nested}}", 0, 8},
		{"'quoted } text'}", 0, 15},
		{"\"escaped \\\" } text\"}", 0, 19},
		{string(rune(96)) + "quoted } text" + string(rune(96)) + "}", 0, 15},
	} {
		got, err := findTemplateEnd(test.text, test.start)
		if err != nil || got != test.want {
			t.Fatalf("findTemplateEnd(%q) = %d, %v; want %d", test.text, got, err, test.want)
		}
	}
	if _, err := findTemplateEnd("{nested", 0); err == nil {
		t.Fatal("unterminated nested interpolation succeeded")
	}
}

func TestLookupAndReflectionEdges(t *testing.T) {
	type namedKey string
	type sample struct {
		Visible string
		hidden  string
	}
	value := "value"
	var nilPointer *string
	var nilInterface any
	pointerToNilInterface := &nilInterface
	tests := []struct {
		name   string
		target any
		key    any
		want   any
		ok     bool
	}{
		{"nil target", nil, "key", nil, false},
		{"nil pointer", nilPointer, "key", nil, false},
		{"pointer to nil interface", pointerToNilInterface, "key", nil, false},
		{"pointer", &value, 0, "v", true},
		{"typed map key", map[namedKey]string{"key": "value"}, namedKey("key"), "value", true},
		{"string map fallback", map[namedKey]string{"key": "value"}, "key", "value", true},
		{"missing map key", map[string]string{}, "key", nil, false},
		{"struct exact", sample{Visible: "yes"}, "Visible", "yes", true},
		{"struct folded", sample{Visible: "yes"}, "visible", "yes", true},
		{"unexported field", sample{hidden: "no"}, "hidden", nil, false},
		{"slice index", []string{"a"}, "0", "a", true},
		{"slice invalid index", []string{"a"}, "bad", nil, false},
		{"slice negative index", []string{"a"}, -1, nil, false},
		{"slice large index", []string{"a"}, 1, nil, false},
		{"string index", "go", 1, "o", true},
		{"string invalid index", "go", "bad", nil, false},
		{"string negative index", "go", -1, nil, false},
		{"string large index", "go", 2, nil, false},
		{"unsupported target", 1, "key", nil, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, ok := lookup(test.target, test.key)
			if ok != test.ok || !reflect.DeepEqual(got, test.want) {
				t.Fatalf("lookup = %#v, %v; want %#v, %v", got, ok, test.want, test.ok)
			}
		})
	}

	noResults := func() {}
	if got, err := callReflect(noResults, nil); err != nil || got != nil {
		t.Fatalf("no-result function = %#v, %v", got, err)
	}
	wantErr := errors.New("function error")
	returnsError := func() (int, error) { return 0, wantErr }
	if _, err := callReflect(returnsError, nil); !errors.Is(err, wantErr) {
		t.Fatalf("returned error = %v", err)
	}
	for _, test := range []struct {
		name   string
		callee any
		args   []any
	}{
		{"not callable", 1, nil},
		{"wrong fixed count", func(int) {}, nil},
		{"wrong variadic count", func(string, ...int) {}, nil},
		{"wrong type", func(int) {}, []any{struct{}{}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := callReflect(test.callee, test.args); err == nil {
				t.Fatal("expected a reflection call error")
			}
		})
	}

	variadic := func(prefix string, values ...int) string { return fmt.Sprintf("%s:%d", prefix, len(values)) }
	if got, err := callReflect(variadic, []any{"x"}); err != nil || got != "x:0" {
		t.Fatalf("zero variadic arguments = %#v, %v", got, err)
	}
	convertible := func(value int) int { return value }
	if got, err := callReflect(convertible, []any{float64(3)}); err != nil || got != 3 {
		t.Fatalf("convertible argument = %#v, %v", got, err)
	}
	nilArgument := func(value *string) bool { return value == nil }
	if got, err := callReflect(nilArgument, []any{nil}); err != nil || got != true {
		t.Fatalf("nil argument = %#v, %v", got, err)
	}
}

func TestLexerEscapesAndFailures(t *testing.T) {
	for source, want := range map[string]string{
		"1.5":              "1.5",
		"'line\\nnext'":    "line\nnext",
		"'line\\rnext'":    "line\rnext",
		"'col\\tvalue'":    "col\tvalue",
		"'slash\\\\quote'": "slash\\quote",
		"'quote\\\"text'":  "quote\"text",
		"'unknown\\x'":     "unknown\\x",
	} {
		tokens, err := lex(source)
		if err != nil {
			t.Fatalf("lex %q: %v", source, err)
		}
		if tokens[0].text != want {
			t.Fatalf("lex %q = %q, want %q", source, tokens[0].text, want)
		}
	}

	templateSource := string(rune(96)) + "template\\" + string(rune(96)) + "value" + string(rune(96))
	tokens, err := lex(templateSource)
	if err != nil || tokens[0].kind != tokenTemplate || tokens[0].text != "template"+string(rune(96))+"value" {
		t.Fatalf("template token = %#v, %v", tokens, err)
	}

	for _, source := range []string{"@", "'unterminated", "'trailing\\", "\u0661"} {
		if _, err := lex(source); err == nil {
			t.Fatalf("lex(%q) unexpectedly succeeded", source)
		}
	}
}

func TestParserFailureCoverage(t *testing.T) {
	invalid := []string{
		"",
		"1 2",
		"1 +",
		"true ?",
		"true ? 1 false",
		"true ? 1 :",
		"!",
		"value.",
		"value[",
		"value[0",
		"value(",
		"value(1",
		"(",
		"(1",
		"[",
		"[1",
		"{1: 2}",
		"{key 2}",
		"{key:",
		"{key: 1",
		"@",
	}
	for _, source := range invalid {
		t.Run(source, func(t *testing.T) {
			if _, err := Compile(source); err == nil {
				t.Fatal("expected a parser error")
			}
		})
	}

	p := parser{}
	if got := p.peek(); got.kind != tokenEOF {
		t.Fatalf("empty parser peek = %#v", got)
	}
	if _, ok := binaryPrecedence("unknown"); ok {
		t.Fatal("unknown operator has precedence")
	}
	if got := (token{text: "value", pos: 3}).String(); got != "\"value\" at 3" {
		t.Fatalf("token string = %q", got)
	}
}

func TestPrimitiveConversionEdges(t *testing.T) {
	falseValues := []any{nil, false, "", 0, math.NaN()}
	for _, value := range falseValues {
		if truthy(value) {
			t.Fatalf("truthy(%#v) = true", value)
		}
	}
	for _, value := range []any{true, "x", 1, struct{}{}} {
		if !truthy(value) {
			t.Fatalf("truthy(%#v) = false", value)
		}
	}

	numbers := []any{int(1), int8(1), int16(1), int32(1), int64(1), uint(1), uint8(1), uint16(1), uint32(1), uint64(1), float32(1), float64(1)}
	for _, value := range numbers {
		if got, ok := number(value); !ok || got != 1 {
			t.Fatalf("number(%T) = %v, %v", value, got, ok)
		}
	}
	if _, ok := number("1"); ok {
		t.Fatal("string unexpectedly converted by number")
	}

	for _, test := range []struct {
		value any
		want  int
		ok    bool
	}{
		{float64(2), 2, true},
		{"3", 3, true},
		{"bad", 0, false},
		{2.5, 0, false},
		{struct{}{}, 0, false},
	} {
		got, ok := integer(test.value)
		if got != test.want || ok != test.ok {
			t.Fatalf("integer(%#v) = %d, %v; want %d, %v", test.value, got, ok, test.want, test.ok)
		}
	}

	if !equal(1, float64(1)) || equal([]int{1}, []int{2}) {
		t.Fatal("numeric or deep equality result was incorrect")
	}
	for value, want := range map[any]string{
		stringValue("custom"): "custom",
		float64(1.25):         "1.25",
		float32(1.5):          "1.5",
		42:                    "42",
	} {
		if got := stringify(value); got != want {
			t.Fatalf("stringify(%#v) = %q, want %q", value, got, want)
		}
	}
	if stringify(nil) != "" || stringify("text") != "text" {
		t.Fatal("nil or string conversion failed")
	}

	var nilMap map[string]int
	var nilSlice []int
	var nilFunc func()
	var nilChan chan int
	for _, value := range []any{nilMap, nilSlice, nilFunc, nilChan} {
		if !isNil(value) {
			t.Fatalf("isNil(%T) = false", value)
		}
	}
	if isNil(strings.NewReader("value")) {
		t.Fatal("non-nil pointer reported nil")
	}

	if got := fmt.Sprint(stringValue("value")); got != "value" {
		t.Fatalf("Stringer sanity check = %q", got)
	}
}
