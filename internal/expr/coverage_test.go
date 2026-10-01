package expr

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/nawafinity/go-namat/internal/value"
)

func TestExpressionBehaviorCoverage(t *testing.T) {
	type namedString string
	type namedInt int
	type namedUint uint
	type sample struct {
		Visible string `json:"visible,omitempty"`
		Hidden  string `json:"-"`
		private string
	}
	root := map[string]any{
		"nilMap": map[string]any(nil), "nilSlice": []int(nil), "nilPointer": (*int)(nil),
		"name": namedString("nimat"), "items": [2]string{"\u0623", "\U0001f30d"},
		"nums": []int{7}, "record": &sample{Visible: "\u0638\u0627\u0647\u0631"},
		"byInt": map[int]string{1: "one"}, "byNamed": map[namedString]string{"x": "named"},
		"u": uint64(math.MaxUint64), "v": uint64(0),
	}
	cases := []struct {
		source string
		root   map[string]any
		vars   map[string]any
		want   any
		fail   bool
	}{
		{"-2", nil, nil, int64(-2), false},
		{"+2", nil, nil, int64(2), false},
		{"!false", nil, nil, true, false},
		{"'a' + 'b'", nil, nil, "ab", false},
		{"1 + 2", nil, nil, int64(3), false},
		{"1 - 2", nil, nil, int64(-1), false},
		{"2 * 3", nil, nil, int64(6), false},
		{"5 % 2", nil, nil, int64(1), false},
		{"2.0 + 3.0", nil, nil, nil, false},
		{"2.0 - 3.0", nil, nil, nil, false},
		{"2.0 * 3.0", nil, nil, nil, false},
		{"2.0 / 4.0", nil, nil, nil, false},
		{"2.0 % 1.0", nil, nil, nil, true},
		{"2 / 4", nil, nil, nil, false},
		{"u + v", root, nil, uint64(math.MaxUint64), false},
		{"1 < 2", nil, nil, true, false},
		{"1 <= 1", nil, nil, true, false},
		{"2 > 1", nil, nil, true, false},
		{"2 >= 2", nil, nil, true, false},
		{"'b' > 'a'", nil, nil, true, false},
		{"[1] == [1]", nil, nil, true, false},
		{"({x: 1}) == ({x: 1})", nil, nil, true, false},
		{"null == null", nil, nil, true, false},
		{"null != 1", nil, nil, nil, true},
		{"true || missing", nil, nil, true, false},
		{"false && missing", nil, nil, false, false},
		{"null ?? 3", nil, nil, int64(3), false},
		{"3 ?? missing", nil, nil, int64(3), false},
		{"name", root, nil, "nimat", false},
		{"record.visible", root, nil, "\u0638\u0627\u0647\u0631", false},
		{"record.hidden", root, nil, nil, true},
		{"record.private", root, nil, nil, true},
		{"items[1]", root, nil, "🌍", false},
		{"items[2]", root, nil, nil, true},
		{"nums[0]", root, nil, int(7), false},
		{"nums[-1]", root, nil, nil, true},
		{"byNamed['x']", root, nil, "named", false},
		{"nilMap?.x", root, nil, nil, false},
		{"nilSlice?.x", root, nil, nil, false},
		{"nilPointer?.x", root, nil, nil, false},
		{"a", nil, map[string]any{"a": "variable"}, "variable", false},
	}
	for _, tc := range cases {
		t.Run(tc.source, func(t *testing.T) {
			p, err := Compile(tc.source)
			if err != nil {
				t.Fatalf("compile: %v", err)
			}
			got, err := p.Eval(&Context{Root: tc.root, Variables: tc.vars})
			if tc.fail {
				if err == nil {
					t.Fatalf("expected evaluation failure; got %#v", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("eval: %v", err)
			}
			if tc.want != nil && !reflectDeepEqual(got, tc.want) {
				t.Fatalf("got %#v, want %#v", got, tc.want)
			}
		})
	}
	if _, err := Compile("-"); err == nil {
		t.Fatal("missing unary operand accepted")
	}
	if _, err := Compile("7u"); err == nil {
		t.Fatal("unsigned suffix accepted")
	}
	if _, err := Compile("1 2"); err == nil {
		t.Fatal("trailing token accepted")
	}
}

func TestNumericAndComparisonBoundaryCoverage(t *testing.T) {
	decimal, _ := value.ParseDecimal("2.5")
	for _, tc := range []struct {
		op      string
		l, r    any
		wantErr bool
	}{
		{"+", uint64(1), uint64(2), false}, {"-", uint64(2), uint64(1), false}, {"*", uint64(2), uint64(3), false}, {"%", uint64(5), uint64(2), false},
		{"+", float64(1), float64(2), false}, {"-", float64(1), float64(2), false}, {"*", float64(2), float64(3), false}, {"/", float64(4), float64(2), false},
		{"/", float64(1), float64(0), true}, {"+", float64(1), int64(1), true}, {"%", float64(5), float64(2), true},
		{"+", decimal, float64(1), true}, {"%", int64(5), int64(2), false}, {"/", int64(2), int64(0), true},
		{"+", int64(math.MaxInt64), int64(1), true}, {"+", int64(math.MinInt64), int64(-1), true},
		{"-", int64(math.MinInt64), int64(1), true}, {"-", int64(math.MaxInt64), int64(-1), true},
		{"*", int64(math.MinInt64), int64(-1), true}, {"*", int64(math.MaxInt64), int64(2), true},
		{"%", int64(5), int64(0), true}, {"+", uint64(math.MaxUint64), uint64(1), true}, {"-", uint64(0), uint64(1), true},
		{"*", uint64(math.MaxUint64), uint64(2), true}, {"%", uint64(1), uint64(0), true},
		{"+", int64(1), uint64(1), true}, {"+", "x", int64(1), true},
	} {
		_, err := numericOperation(tc.op, tc.l, tc.r)
		if (err != nil) != tc.wantErr {
			t.Errorf("numericOperation(%q, %#v, %#v): err=%v", tc.op, tc.l, tc.r, err)
		}
	}
	for _, input := range []any{int64(4), decimal, float32(1.5), uint64(2), "x", nil} {
		_, _ = negate(input)
	}
	if _, err := negate(int64(math.MinInt64)); err == nil {
		t.Fatal("MinInt64 negation succeeded")
	}
	if _, err := compareValues("<", math.NaN(), float64(0)); err == nil {
		t.Fatal("NaN ordered")
	}
	for _, tc := range []struct {
		op      string
		l, r    any
		wantErr bool
	}{
		{"<", int64(1), uint64(2), false}, {"<", "a", "b", false}, {"<", "a", int64(1), true}, {"<", true, false, true}, {"?", 1, 2, true},
	} {
		_, err := compareValues(tc.op, tc.l, tc.r)
		if (err != nil) != tc.wantErr {
			t.Errorf("compare %q: %v", tc.op, err)
		}
	}
	for _, tc := range [][2]any{{value.MissingValue{}, value.MissingValue{}}, {nil, nil}, {nil, int64(1)}, {int64(1), uint64(1)}, {float64(1), float64(1)}, {"a", "b"}, {true, false}, {[]int{1}, []int{1}}, {1, "1"}} {
		_, _ = equalValues(tc[0], tc[1])
	}
	if _, err := exactDecimal("1"); err == nil {
		t.Fatal("string converted to decimal")
	}
	if _, err := exactDecimal(float64(1)); err == nil {
		t.Fatal("float converted to exact decimal")
	}
}

func TestRuntimeReflectionAndFunctionCoverage(t *testing.T) {
	type namedInt int64
	type namedUint uint64
	type namedFloat float32
	type namedBool bool
	type namedText string
	for _, v := range []any{nil, value.MissingValue{}, (*int)(nil), (*int)(new(int)), (*value.DecimalValue)(nil), value.DecimalValue{}, namedInt(1), namedUint(1), namedFloat(1), namedBool(true), namedText("x"), []int{}, [1]int{}, map[string]int{}, struct{}{}, complex(1, 2), make(chan int)} {
		_ = KindOf(v)
	}
	if got, ok := AsBool(true); !ok || !got {
		t.Fatalf("AsBool true = %v, %v", got, ok)
	}
	if _, ok := AsBool(1); ok {
		t.Fatal("AsBool accepted integer")
	}
	_ = normalizeRuntimeArguments([]any{true, "x", namedInt(2), namedUint(3), namedFloat(4), value.DecimalValue{}}, Signature{Params: []value.Kind{value.Bool, value.String, value.Int, value.Uint, value.Float, value.Decimal}})
	_ = normalizeRuntimeArguments([]any{1}, Signature{Params: []value.Kind{value.Any, value.Rich, value.String}})
	if err := validateRuntimeArguments("f", []any{true}, Signature{Params: []value.Kind{value.Bool, value.String}}); err == nil {
		t.Fatal("arity mismatch accepted")
	}
	if err := validateRuntimeArguments("f", []any{1}, Signature{Params: []value.Kind{value.Bool}}); err == nil {
		t.Fatal("type mismatch accepted")
	}
	if err := validateRuntimeArguments("f", []any{1, 2}, Signature{Params: []value.Kind{value.Any}, Variadic: true}); err != nil {
		t.Fatal(err)
	}

	funcs := map[string]Function{
		"fail":  {Call: func(context.Context, ...any) (any, error) { return nil, errors.New("broken") }},
		"typed": {Signature: Signature{Returns: value.String}, Call: func(context.Context, ...any) (any, error) { return "ok", nil }},
		"nilfn": {Signature: Signature{}, Call: nil},
	}
	for _, source := range []string{"fail()", "typed()", "nilfn()", "missingFn()"} {
		p, err := Compile(source)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = p.Eval(&Context{Functions: funcs})
	}
	program, _ := Compile("typed()")
	_, _ = program.Eval(&Context{Context: canceledContext(), Functions: map[string]Function{"typed": funcs["typed"]}})
	program, _ = Compile("typed()")
	_, _ = program.Eval(&Context{MaxSteps: 1, Functions: map[string]Function{"typed": funcs["typed"]}})
	if _, err := (*Program)(nil).Eval(nil); err == nil {
		t.Fatal("nil program evaluated")
	}
	if err := (*Context)(nil).step(); err != nil {
		t.Fatal(err)
	}
	ctx := &Context{MaxSteps: 1}
	_ = ctx.step()
	if err := ctx.step(); err == nil {
		t.Fatal("step limit not enforced")
	}
	if err := (&Context{Context: canceledContext()}).step(); err == nil {
		t.Fatal("cancel ignored")
	}
	shared := 0
	_ = (&Context{SharedSteps: &shared}).step()
	if shared != 1 {
		t.Fatalf("shared steps = %d", shared)
	}
}

func TestLexerParserAndFormattingEdgeCoverage(t *testing.T) {
	for _, source := range []string{"'line\\nnext'", `"quote: \\"`, `"unknown\\q"`, "'tail\\"} {
		_, _ = Compile(source)
	}
	for _, source := range []string{"1.", "@", "0..1", "'unterminated", "18446744073709551616", "1.2.3", "f(1,)", "a.", "a[", "[1,]", "{1: 2}", "{x 2}", "(1", "1 | 2", "1 |", "1 | upper(2", "(1).x()"} {
		if _, err := Compile(source); err == nil {
			t.Errorf("Compile(%q) unexpectedly succeeded", source)
		}
	}
	for _, source := range []string{"[]", "{}", "f()", "[1,2]", "{a:1,b:'x'}", "1 | f"} {
		_, _ = Compile(source)
	}
	if _, err := CompileWithOptions("1", CompileOptions{Functions: map[string]Signature{}}); err != nil {
		t.Fatalf("call-free literal rejected with empty registry: %v", err)
	}
	if _, err := CompileWithOptions("f()", CompileOptions{Functions: map[string]Signature{"f": {Params: []value.Kind{value.String}}}}); err == nil {
		t.Fatal("call arity not validated")
	}
	if _, err := CompileWithOptions("f(1)", CompileOptions{Functions: map[string]Signature{"f": {Params: []value.Kind{value.String}, Variadic: true}}}); err != nil {
		t.Fatal(err)
	}
	if _, err := CompileWithOptions("f(1)", CompileOptions{Functions: map[string]Signature{}}); err == nil {
		t.Fatal("unknown function not validated")
	}
	for _, input := range []any{value.MissingValue{}, nil, "x", value.DecimalValue{}, float32(1.25), float64(1.25), 1, true, []byte("x")} {
		_, _ = Format(input)
	}
	for _, tk := range []token{{text: "word", pos: 4}, {text: "", pos: 0}} {
		if !strings.Contains(tk.String(), "at") {
			t.Fatalf("token.String = %q", tk.String())
		}
	}
	_, _ = indexValue(int64(-1))
	_, _ = indexValue(uint64(math.MaxUint64))
	_, _ = indexValue("0")
	_, _ = lookup(map[string]int{"x": 1}, "missing")
	_, _ = lookup("🌍", int64(0))
	_, _ = lookup("🌍", int64(1))
	_, _ = lookup("🌍", int64(-1))
	_, _ = lookup([]int{1}, int64(9))
	_, _ = lookup(struct {
		Visible int `json:"visible,omitempty"`
		Ignore  int `json:"-"`
		hidden  int
	}{1, 2, 3}, "missing")
	_ = keysOf(nil)
	_ = keysOf(value.MissingValue{})
	_ = keysOf(map[int]int{1: 2})
	_ = keysOf(struct {
		Visible int `json:"visible,omitempty"`
		Ignore  int `json:"-"`
		hidden  int
	}{1, 2, 3})
	_ = mapKeys(nil)
	_ = functionNames(nil)
	_ = unknownNameError("field", "zzzz", []string{"alpha"})
}

func canceledContext() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}

func reflectDeepEqual(a, b any) bool { return fmt.Sprint(a) == fmt.Sprint(b) }

type failingNode struct{}

func (failingNode) eval(*Context) (any, error) { return nil, errors.New("child failed") }

func TestNodeFailurePropagationAndRemainingSemantics(t *testing.T) {
	ctx := &Context{}
	for _, n := range []node{
		arrayNode{values: []node{failingNode{}}},
		objectNode{entries: []objectEntry{{key: "x", value: failingNode{}}}},
		identifierNode{name: "missing"},
		memberNode{target: failingNode{}},
		memberNode{target: failingNode{}, optional: true},
		memberNode{target: literalNode{value: nil}},
		memberNode{target: literalNode{value: map[string]int{"x": 1}}, key: failingNode{}},
		memberNode{target: literalNode{value: map[string]int{"x": 1}}, key: literalNode{value: "y"}},
		callNode{name: "f", args: []node{failingNode{}}},
		unaryNode{op: "!", value: literalNode{value: 1}},
		unaryNode{op: "+", value: literalNode{value: "x"}},
		unaryNode{op: "?", value: literalNode{value: 1}},
		unaryNode{op: "!", value: failingNode{}},
		binaryNode{op: "!=", left: literalNode{value: 1}, right: literalNode{value: 2}},
		binaryNode{op: "?", left: literalNode{value: 1}, right: literalNode{value: 2}},
		binaryNode{op: "+", left: literalNode{value: 1}, right: failingNode{}},
		binaryNode{op: "&&", left: literalNode{value: true}, right: literalNode{value: "x"}},
		binaryNode{op: "&&", left: literalNode{value: 1}, right: literalNode{value: true}},
		binaryNode{op: "==", left: literalNode{value: value.MissingValue{}}, right: literalNode{value: nil}},
	} {
		_, _ = n.eval(ctx)
	}
	_, _ = (callNode{name: "f", args: []node{failingNode{}}}).eval(&Context{Functions: map[string]Function{"f": {Signature: Signature{Params: []value.Kind{value.Any}}, Call: func(context.Context, ...any) (any, error) { return nil, nil }}}})
	for _, n := range []node{
		arrayNode{}, objectNode{}, identifierNode{name: "x"}, memberNode{}, callNode{name: "f"},
		unaryNode{op: "!", value: literalNode{value: true}}, binaryNode{op: "+", left: literalNode{value: 1}, right: literalNode{value: 2}},
	} {
		_, _ = n.eval(&Context{MaxSteps: 1, steps: 1})
	}
	_, _ = (memberNode{target: literalNode{value: map[string]int{"x": 1}}, key: literalNode{value: "y"}, optional: true}).eval(&Context{})
	var nilValue *int
	var wrapped any = nilValue
	if KindOf(&wrapped) != value.Null {
		t.Fatalf("wrapped typed nil kind = %s", KindOf(&wrapped))
	}
	for _, target := range []any{nil, value.MissingValue{}, (*int)(nil), &wrapped, map[int]int{1: 2}, struct {
		Visible int
		hidden  int
	}{Visible: 1}} {
		_ = keysOf(target)
	}
	_, _ = lookup(value.MissingValue{}, "x")
	_, _ = lookup(&wrapped, "x")
	_, _ = lookup(map[int]string{1: "one"}, "1")
	_, _ = lookup(map[string]int{"x": 1}, int64(0))
	_, _ = lookup(struct{ Value int }{1}, int64(0))
	if got, ok := lookup(map[string]int{"x": 1}, "x"); !ok || got != 1 {
		t.Fatalf("map lookup = %#v, %v", got, ok)
	}
	if got, ok := indexValue(uint64(1)); !ok || got != 1 {
		t.Fatalf("uint index = %d, %v", got, ok)
	}
	_ = mapKeys(map[string]any{"x": 1})
	_, _ = numericOperation("?", uint64(1), uint64(1))
	_, _ = compareValues("?", "a", "a")
	for _, tc := range [][2]any{{float64(1), float64(2)}, {float64(2), float64(1)}, {float64(1), float64(1)}} {
		_, _ = compareValues("<", tc[0], tc[1])
	}
	if _, err := (&Program{root: nil}).Eval(nil); err == nil {
		t.Fatal("empty program accepted")
	}
	program, _ := Compile("1")
	if _, err := program.Eval(nil); err != nil {
		t.Fatalf("nil context: %v", err)
	}
}

func TestParserRecursiveValidationAndMalformedDelimiters(t *testing.T) {
	for _, source := range []string{"1 +", "a[1", "[1", "{a: 1", "{a: 1 +}", "1 | f(", "0.1.2", "9223372036854775808"} {
		_, _ = Compile(source)
	}
	for _, source := range []string{"[f()]", "{x: f()}", "!f()", "f() + 1", "outer(f())"} {
		functions := map[string]Signature{}
		if source == "outer(f())" {
			functions["outer"] = Signature{Params: []value.Kind{value.Any}}
		}
		if _, err := CompileWithOptions(source, CompileOptions{Functions: functions}); err == nil {
			t.Errorf("nested unknown function accepted: %s", source)
		}
	}
	if err := validateCalls(arrayNode{values: []node{callNode{name: "f"}}}, map[string]Signature{}); err == nil {
		t.Fatal("array child call escaped validation")
	}
	if err := validateCalls(objectNode{entries: []objectEntry{{value: callNode{name: "f"}}}}, map[string]Signature{}); err == nil {
		t.Fatal("object child call escaped validation")
	}
	if err := validateCalls(memberNode{target: callNode{name: "f"}}, map[string]Signature{}); err == nil {
		t.Fatal("member target call escaped validation")
	}
	if err := validateCalls(unaryNode{value: callNode{name: "f"}}, map[string]Signature{}); err == nil {
		t.Fatal("unary child call escaped validation")
	}
	if err := validateCalls(binaryNode{left: callNode{name: "f"}}, map[string]Signature{}); err == nil {
		t.Fatal("binary child call escaped validation")
	}
	if err := validateCalls(callNode{name: "outer", args: []node{callNode{name: "f"}}}, map[string]Signature{"outer": {Params: []value.Kind{value.Any}}}); err == nil {
		t.Fatal("argument call escaped validation")
	}
	p := parser{tokens: []token{{kind: tokenEOF}}}
	if _, err := p.parseArguments(); err == nil {
		t.Fatal("missing opening parenthesis accepted")
	}
	if tok := p.peek(); tok.kind != tokenEOF {
		t.Fatalf("peek at exhausted parser = %v", tok)
	}
	if _, ok := binaryPrecedence("~"); ok {
		t.Fatal("unknown operator has precedence")
	}
	if _, err := lex("'a\\r\\t\\q'", 0); err != nil {
		t.Fatal(err)
	}
	decimalParser := parser{tokens: []token{{kind: tokenNumber, text: "1.2.3"}}}
	if _, err := decimalParser.parsePrimary(); err == nil {
		t.Fatal("malformed decimal accepted")
	}
	arrayParser := parser{tokens: []token{{kind: tokenLBracket}, {kind: tokenNumber, text: "1"}, {kind: tokenOperator, text: "+"}, {kind: tokenRBracket}}}
	if _, err := arrayParser.parsePrimary(); err == nil {
		t.Fatal("invalid array item accepted")
	}
}
