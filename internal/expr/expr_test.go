package expr

import (
	"fmt"
	"testing"
)

func TestExpressionLanguage(t *testing.T) {
	root := map[string]any{
		"agency": map[string]any{"name": "هيئة البيانات", "score": 91.5},
		"items":  []any{map[string]any{"name": "API"}},
	}
	functions := map[string]Function{
		"wrap": func(args ...any) (any, error) { return fmt.Sprintf("[%v]", args[0]), nil },
	}
	tests := []struct {
		expression string
		want       any
	}{
		{"agency.name", "هيئة البيانات"},
		{"agency.score >= 90 && agency.score < 100", true},
		{"items[0].name", "API"},
		{"missing?.value ?? 'N/A'", "N/A"},
		{"wrap(agency.name)", "[هيئة البيانات]"},
		{"`Score: ${agency.score}`", "Score: 91.5"},
		{"$idx + 1", float64(3)},
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
	if _, err := Compile("agency.[name"); err == nil {
		t.Fatal("expected syntax error")
	}
}

func TestCallsVariadicGoFunctionFromData(t *testing.T) {
	program, err := Compile("join('أ', 'ب', 'ج')")
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
	if got != "أبج" {
		t.Fatalf("got %q, want %q", got, "أبج")
	}
}
