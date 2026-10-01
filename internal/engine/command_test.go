package engine

import (
	"strings"
	"testing"
)

func TestParseEveryV1CommandType(t *testing.T) {
	tests := []struct {
		source     string
		wantType   CommandType
		variable   string
		expression string
	}{
		{source: "image", wantType: CommandInsert, expression: "image"},
		{source: "!active", wantType: CommandInsert, expression: "!active"},
		{source: "#let total = sum(items, 'amount')", wantType: CommandLet, variable: "total", expression: "sum(items, 'amount')"},
		{source: "#if customer.active", wantType: CommandIf, expression: "customer.active"},
		{source: "#else", wantType: CommandElse},
		{source: "/if", wantType: CommandEndIf},
		{source: "#each items as item", wantType: CommandEach, variable: "item", expression: "items"},
		{source: "/each", wantType: CommandEndEach},
		{source: "@image company.logo", wantType: CommandImage, expression: "company.logo"},
		{source: "@link project.link", wantType: CommandLink, expression: "project.link"},
		{source: "@html fragment", wantType: CommandHTML, expression: "fragment"},
		{source: "@raw-xml trusted", wantType: CommandRawXML, expression: "trusted"},
	}
	for _, test := range tests {
		t.Run(test.source, func(t *testing.T) {
			command, err := parseCommand(test.source)
			if err != nil {
				t.Fatal(err)
			}
			if command.Type != test.wantType || command.Variable != test.variable || command.Expression != test.expression {
				t.Fatalf("command = %#v", command)
			}
		})
	}
}

func TestParseCommandRejectsMalformedV1Syntax(t *testing.T) {
	invalid := []string{
		"", "#if", "#else value", "/if value", "/each value",
		"#each items", "#each items as 1bad", "#let", "#let value", "#let 1bad = value",
		"@image", "@link", "@html", "@raw-xml", "@unknown value", "#unknown value", "/unknown",
	}
	for _, source := range invalid {
		t.Run(source, func(t *testing.T) {
			if _, err := parseCommand(source); err == nil {
				t.Fatal("expected syntax error")
			}
		})
	}
	if _, err := parseCommand(string([]byte{0xff})); err == nil || !strings.Contains(err.Error(), "UTF-8") {
		t.Fatalf("invalid UTF-8 error = %v", err)
	}
}

func TestCommandParserIgnoresKeywordsInsideNestedExpressions(t *testing.T) {
	command, err := parseCommand("#each filter(items, ' as ') as item")
	if err != nil {
		t.Fatal(err)
	}
	if command.Expression != "filter(items, ' as ')" || command.Variable != "item" {
		t.Fatalf("command = %#v", command)
	}
	command, err = parseCommand("#let value = choose('=', source)")
	if err != nil || command.Expression != "choose('=', source)" {
		t.Fatalf("command = %#v, %v", command, err)
	}
}
