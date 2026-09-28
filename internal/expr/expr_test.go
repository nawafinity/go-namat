package expr

import (
	"fmt"
	"testing"
)

func TestExpressionLanguage(t *testing.T) {
	root := map[string]any{
		"record": map[string]any{"name": "Example Record", "score": 91.5},
		"items":  []any{map[string]any{"name": "Item"}},
	}
	functions := map[string]Function{
		"wrap": func(args ...any) (any, error) { return fmt.Sprintf("[%v]", args[0]), nil },
	}
	tests := []struct {
		expression string
		want       any
	}{
		{"record.name", "Example Record"},
		{"record.score >= 90 && record.score < 100", true},
		{"items[0].name", "Item"},
		{"missing?.value ?? 'N/A'", "N/A"},
		{"wrap(record.name)", "[Example Record]"},
		{"`Score: ${record.score}`", "Score: 91.5"},
		{"$idx + 1", float64(3)},
		{"record.score >= 90 ? 'high' : 'low'", "high"},
		{"[record.name, items[0].name][1]", "Item"},
		{"({ url: 'https://example.com', label: record.name }).label", "Example Record"},
		{"record.name.slice(0, 7)", "Example"},
		{"record.name.length > 5", true},
		{"['a', 'b', 'c'].join('-')", "a-b-c"},
		{"items.includes(items[0])", true},
	}
	for _, test := range tests {
		program, err := Compile(test.expression)
		if err != nil {
			t.Fatalf("compile %q: %v", test.expression, err)
		}
		got, err := program.Eval(&Context{Root: root, Variables: map[string]any{"$idx": 2}, Functions: functions})
		if err != nil {
			t.Fatalf("evaluate %q: %v", test.expression, err)
		}
		if fmt.Sprint(got) != fmt.Sprint(test.want) {
			t.Errorf("%s: got %#v, want %#v", test.expression, got, test.want)
		}
	}
}

func TestInvalidExpression(t *testing.T) {
	if _, err := Compile("record.[name"); err == nil {
		t.Fatal("expected syntax error")
	}
}

func TestCallsVariadicGoFunctionFromData(t *testing.T) {
	program, err := Compile("join('a', 'b', 'c')")
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	root := map[string]any{
		"join": func(values ...string) string {
			result := ""
			for _, value := range values {
				result += value
			}
			return result
		},
	}
	got, err := program.Eval(&Context{Root: root})
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if got != "abc" {
		t.Fatalf("got %q, want %q", got, "abc")
	}
}

func TestExpressionOperatorsAndBuiltInMembers(t *testing.T) {
	type sample struct {
		DisplayName string `json:"display_name"`
	}
	root := map[string]any{
		"text":   " Example ",
		"values": []int{1, 2, 3},
		"model":  &sample{DisplayName: "Visible"},
	}
	tests := []struct {
		name       string
		expression string
		want       any
	}{
		{name: "unary not", expression: "!false", want: true},
		{name: "unary minus", expression: "-5", want: float64(-5)},
		{name: "unary plus", expression: "+5", want: float64(5)},
		{name: "subtraction", expression: "10 - 3", want: float64(7)},
		{name: "multiplication", expression: "4 * 2", want: float64(8)},
		{name: "division", expression: "9 / 3", want: float64(3)},
		{name: "modulo", expression: "7 % 4", want: float64(3)},
		{name: "string concatenation", expression: "'item-' + 2", want: "item-2"},
		{name: "strict equality", expression: "2 === 2", want: true},
		{name: "inequality", expression: "2 !== 3", want: true},
		{name: "less than", expression: "2 < 3", want: true},
		{name: "less or equal", expression: "2 <= 2", want: true},
		{name: "greater than", expression: "3 > 2", want: true},
		{name: "greater or equal", expression: "3 >= 3", want: true},
		{name: "null coalescing", expression: "null ?? 'fallback'", want: "fallback"},
		{name: "logical and", expression: "true && 'yes'", want: "yes"},
		{name: "logical or", expression: "false || 'yes'", want: "yes"},
		{name: "false ternary branch", expression: "false ? 'yes' : 'no'", want: "no"},
		{name: "string index", expression: "text[1]", want: "E"},
		{name: "map index", expression: "model['display_name']", want: "Visible"},
		{name: "JSON struct tag", expression: "model.display_name", want: "Visible"},
		{name: "trim", expression: "text.trim()", want: "Example"},
		{name: "uppercase", expression: "text.trim().toUpperCase()", want: "EXAMPLE"},
		{name: "lowercase", expression: "text.trim().lower()", want: "example"},
		{name: "contains", expression: "text.contains('amp')", want: true},
		{name: "includes alias", expression: "text.includes('amp')", want: true},
		{name: "starts with", expression: "text.trim().startsWith('Ex')", want: true},
		{name: "ends with", expression: "text.trim().endsWith('ple')", want: true},
		{name: "negative slice", expression: "text.trim().slice(-3)", want: "ple"},
		{name: "array length", expression: "values.length", want: 3},
		{name: "array default join", expression: "values.join()", want: "1,2,3"},
		{name: "array contains false", expression: "values.contains(9)", want: false},
		{name: "object length", expression: "({a: 1, b: 2}).length", want: 2},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			program, err := Compile(test.expression)
			if err != nil {
				t.Fatalf("Compile: %v", err)
			}
			got, err := program.Eval(&Context{Root: root})
			if err != nil {
				t.Fatalf("Eval: %v", err)
			}
			if fmt.Sprint(got) != fmt.Sprint(test.want) {
				t.Fatalf("got %#v, want %#v", got, test.want)
			}
		})
	}
}

func TestExpressionFailureModes(t *testing.T) {
	tests := []string{
		"1 / 0",
		"1 % 0",
		"null.value",
		"({a: 1}).missing",
		"1()",
		"'x'.trim(1)",
		"'x'.slice('bad')",
		"[1, 2].join(',', ';')",
	}
	for _, expression := range tests {
		t.Run(expression, func(t *testing.T) {
			program, err := Compile(expression)
			if err == nil {
				_, err = program.Eval(&Context{})
			}
			if err == nil {
				t.Fatal("expected expression failure")
			}
		})
	}
	var program *Program
	if _, err := program.Eval(nil); err == nil {
		t.Fatal("nil program evaluated successfully")
	}
}

func TestExpressionPreservesMultilingualUnicode(t *testing.T) {
	const value = "\u0645\u0631\u062d\u0628\u0627 / \u65e5\u672c\u8a9e / \U0001f30d"
	program, err := Compile("value")
	if err != nil {
		t.Fatal(err)
	}
	got, err := program.Eval(&Context{Root: map[string]any{"value": value}})
	if err != nil {
		t.Fatal(err)
	}
	if got != value {
		t.Fatalf("Unicode value changed: got %q, want %q", got, value)
	}
	const identifier = "\u0642\u064a\u0645\u0629"
	program, err = Compile(identifier)
	if err != nil {
		t.Fatalf("compile Unicode identifier: %v", err)
	}
	got, err = program.Eval(&Context{Root: map[string]any{identifier: value}})
	if err != nil || got != value {
		t.Fatalf("Unicode identifier evaluation = %#v, %v", got, err)
	}
}
