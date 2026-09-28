package expr

import (
	"fmt"
	"testing"
)

func TestExpressionLanguage(t *testing.T) {
	root := map[string]any{
		"record": map[string]any{"name": "قيمة عربية", "score": 91.5},
		"items":  []any{map[string]any{"name": "Item"}},
	}
	functions := map[string]Function{
		"wrap": func(args ...any) (any, error) { return fmt.Sprintf("[%v]", args[0]), nil },
	}
	tests := []struct {
		expression string
		want       any
	}{
		{"record.name", "قيمة عربية"},
		{"record.score >= 90 && record.score < 100", true},
		{"items[0].name", "Item"},
		{"missing?.value ?? 'N/A'", "N/A"},
		{"wrap(record.name)", "[قيمة عربية]"},
		{"`Score: ${record.score}`", "Score: 91.5"},
		{"$idx + 1", float64(3)},
		{"record.score >= 90 ? 'مرتفع' : 'منخفض'", "مرتفع"},
		{"[record.name, items[0].name][1]", "Item"},
		{"({ url: 'https://example.com', label: record.name }).label", "قيمة عربية"},
		{"record.name.slice(0, 4)", "قيمة"},
		{"record.name.length > 5", true},
		{"['أ', 'ب', 'ج'].join('-')", "أ-ب-ج"},
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
