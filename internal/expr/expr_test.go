package expr

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/nawafinity/go-namat/internal/value"
)

func TestV1ExpressionLanguage(t *testing.T) {
	decimal, _ := value.ParseDecimal("91.5")
	root := map[string]any{
		"record": map[string]any{"name": "Example Record", "score": decimal},
		"items":  []any{map[string]any{"name": "Item"}},
	}
	functions := testFunctions()
	tests := []struct {
		expression string
		want       string
	}{
		{"record.name", "Example Record"},
		{"record.score >= 90 && record.score < 100", "true"},
		{"items[0].name", "Item"},
		{"missing?.value | default('N/A')", "N/A"},
		{"record.name | upper | wrap", "[EXAMPLE RECORD]"},
		{"[record.name, items[0].name][1]", "Item"},
		{"({ url: 'https://example.com', label: record.name }).label", "Example Record"},
		{"0.1 + 0.2", "0.3"},
		{"9007199254740993 == 9007199254740992", "false"},
	}
	for _, test := range tests {
		program, err := CompileWithOptions(test.expression, CompileOptions{Functions: signatures(functions)})
		if err != nil {
			t.Fatalf("compile %q: %v", test.expression, err)
		}
		got, err := program.Eval(&Context{Context: context.Background(), Root: root, Functions: functions, MaxSteps: 1_000})
		if err != nil {
			t.Fatalf("evaluate %q: %v", test.expression, err)
		}
		formatted, err := Format(got)
		if err != nil {
			t.Fatalf("format %q: %v", test.expression, err)
		}
		if formatted != test.want {
			t.Errorf("%s: got %q, want %q", test.expression, formatted, test.want)
		}
	}
}

func TestStrictExpressionFailures(t *testing.T) {
	functions := testFunctions()
	compileFailures := []string{
		"2 === 2",
		"false ? 'a' : 'b'",
		"`value ${x}`",
		"name.trim()",
		"nil",
		"undefined",
		"unknown('x')",
	}
	for _, source := range compileFailures {
		t.Run("compile "+source, func(t *testing.T) {
			if _, err := CompileWithOptions(source, CompileOptions{Functions: signatures(functions)}); err == nil {
				t.Fatal("expected compile error")
			}
		})
	}

	evaluationFailures := []string{
		"'item-' + 2",
		"true && 'yes'",
		"1 == '1'",
		"1 < '2'",
		"1 / 0",
		"1 % 0",
		"null.value",
		"({a: 1}).missing",
	}
	for _, source := range evaluationFailures {
		t.Run("eval "+source, func(t *testing.T) {
			program, err := CompileWithOptions(source, CompileOptions{Functions: signatures(functions)})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := program.Eval(&Context{Functions: functions}); err == nil {
				t.Fatal("expected evaluation error")
			}
		})
	}
}

func TestOptionalAccessPreservesMissing(t *testing.T) {
	program, err := Compile("record?.name")
	if err != nil {
		t.Fatal(err)
	}
	result, err := program.Eval(&Context{Root: map[string]any{}})
	if err != nil {
		t.Fatal(err)
	}
	if KindOf(result) != value.Missing {
		t.Fatalf("kind = %s, want missing", KindOf(result))
	}
	if _, err := Format(result); err == nil {
		t.Fatal("missing value formatted without default")
	}
}

func TestExactStructAndMapNames(t *testing.T) {
	type sample struct {
		DisplayName string `json:"display_name"`
	}
	program, err := Compile("model.display_name")
	if err != nil {
		t.Fatal(err)
	}
	got, err := program.Eval(&Context{Root: map[string]any{"model": sample{DisplayName: "Visible"}}})
	if err != nil || got != "Visible" {
		t.Fatalf("result = %#v, %v", got, err)
	}
	program, _ = Compile("model.DisplayName")
	if _, err := program.Eval(&Context{Root: map[string]any{"model": sample{DisplayName: "Visible"}}}); err == nil {
		t.Fatal("struct field bypassed exact JSON name")
	}
}

func TestNamedAndPointerScalarKinds(t *testing.T) {
	type label string
	type count int64
	type switchValue bool
	text := label("value")
	number := count(2)
	enabled := switchValue(true)
	if KindOf(&text) != value.String || KindOf(&number) != value.Int || KindOf(&enabled) != value.Bool {
		t.Fatalf("pointer kinds = %s, %s, %s", KindOf(&text), KindOf(&number), KindOf(&enabled))
	}
	formatted, err := Format(&text)
	if err != nil || formatted != "value" {
		t.Fatalf("pointer format = %q, %v", formatted, err)
	}
	program, _ := Compile("number + 1")
	result, err := program.Eval(&Context{Root: map[string]any{"number": &number}})
	if err != nil || result != int64(3) {
		t.Fatalf("pointer arithmetic = %#v, %v", result, err)
	}
	program, _ = Compile("enabled && text == 'value'")
	result, err = program.Eval(&Context{Root: map[string]any{"enabled": &enabled, "text": &text}})
	if err != nil || result != true {
		t.Fatalf("pointer boolean/string evaluation = %#v, %v", result, err)
	}
}

func TestCompileAndEvaluationLimits(t *testing.T) {
	if _, err := CompileWithOptions("1234", CompileOptions{MaxBytes: 3}); err == nil {
		t.Fatal("expression byte limit was not enforced")
	}
	if _, err := CompileWithOptions("1 + 2 + 3", CompileOptions{MaxTokens: 3}); err == nil {
		t.Fatal("token limit was not enforced")
	}
	if _, err := CompileWithOptions(strings.Repeat("(", 20)+"1"+strings.Repeat(")", 20), CompileOptions{MaxDepth: 8}); err == nil {
		t.Fatal("depth limit was not enforced")
	}
	program, _ := Compile("1 + 2")
	if _, err := program.Eval(&Context{MaxSteps: 1}); err == nil {
		t.Fatal("evaluation step limit was not enforced")
	}
}

func TestFunctionSignatureValidation(t *testing.T) {
	functions := testFunctions()
	if _, err := CompileWithOptions("wrap()", CompileOptions{Functions: signatures(functions)}); err == nil {
		t.Fatal("arity mismatch compiled")
	}
	program, err := CompileWithOptions("upper(1)", CompileOptions{Functions: signatures(functions)})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := program.Eval(&Context{Functions: functions}); err == nil || !strings.Contains(err.Error(), "expects string") {
		t.Fatalf("runtime type error = %v", err)
	}
	functions["wrong"] = Function{Signature: Signature{Returns: value.String}, Call: func(context.Context, ...any) (any, error) { return int64(1), nil }}
	program, err = CompileWithOptions("wrong()", CompileOptions{Functions: signatures(functions)})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := program.Eval(&Context{Functions: functions}); err == nil || !strings.Contains(err.Error(), "declared string") {
		t.Fatalf("return type error = %v", err)
	}
}

func TestDiagnosticsSuggestNearbyName(t *testing.T) {
	program, _ := Compile("customer.nmae")
	_, err := program.Eval(&Context{Root: map[string]any{"customer": map[string]any{"name": "A"}}})
	if err == nil || !strings.Contains(err.Error(), `did you mean "name"`) {
		t.Fatalf("error = %v", err)
	}
}

func TestExpressionPreservesMultilingualUnicode(t *testing.T) {
	const identifier = "\u0642\u064a\u0645\u0629"
	const text = "\u0645\u0631\u062d\u0628\u0627 / \u65e5\u672c\u8a9e / \U0001f30d"
	program, err := Compile(identifier)
	if err != nil {
		t.Fatal(err)
	}
	got, err := program.Eval(&Context{Root: map[string]any{identifier: text}})
	if err != nil || got != text {
		t.Fatalf("result = %#v, %v", got, err)
	}
}

func testFunctions() map[string]Function {
	return map[string]Function{
		"default": {
			Signature: Signature{Params: []value.Kind{value.Any, value.Any}, Returns: value.Any},
			Call: func(_ context.Context, args ...any) (any, error) {
				if KindOf(args[0]) == value.Missing || KindOf(args[0]) == value.Null {
					return args[1], nil
				}
				return args[0], nil
			},
		},
		"upper": {
			Signature: Signature{Params: []value.Kind{value.String}, Returns: value.String},
			Call: func(_ context.Context, args ...any) (any, error) {
				return strings.ToUpper(args[0].(string)), nil
			},
		},
		"wrap": {
			Signature: Signature{Params: []value.Kind{value.String}, Returns: value.String},
			Call: func(_ context.Context, args ...any) (any, error) {
				return fmt.Sprintf("[%s]", args[0]), nil
			},
		},
	}
}

func signatures(functions map[string]Function) map[string]Signature {
	result := make(map[string]Signature, len(functions))
	for name, function := range functions {
		result[name] = function.Signature
	}
	return result
}
