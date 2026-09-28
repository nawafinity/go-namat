package namat

import (
	"strings"
	"testing"
)

func TestParseEveryCommandType(t *testing.T) {
	tests := []struct {
		source   string
		wantType CommandType
	}{
		{source: "value", wantType: CommandInsert},
		{source: "INS value", wantType: CommandInsert},
		{source: "= value", wantType: CommandInsert},
		{source: "EXEC local = value", wantType: CommandExec},
		{source: "! local = value", wantType: CommandExec},
		{source: "SET local = value", wantType: CommandSet},
		{source: "IF visible", wantType: CommandIf},
		{source: "ELSE", wantType: CommandElse},
		{source: "END-IF", wantType: CommandEndIf},
		{source: "FOR item IN items", wantType: CommandFor},
		{source: "END-FOR item", wantType: CommandEndFor},
		{source: "IMAGE image", wantType: CommandImage},
		{source: "LINK link", wantType: CommandLink},
		{source: "HTML html", wantType: CommandHTML},
		{source: "RAW-XML xml", wantType: CommandRawXML},
		{source: "QUERY synthetic query", wantType: CommandQuery},
		{source: "ALIAS display INS value", wantType: CommandAlias},
		{source: "*display", wantType: CommandAliasRef},
	}
	for _, test := range tests {
		t.Run(test.source, func(t *testing.T) {
			command, err := parseCommand(test.source)
			if err != nil {
				t.Fatalf("parseCommand: %v", err)
			}
			if command.Type != test.wantType {
				t.Fatalf("type = %s, want %s", command.Type, test.wantType)
			}
		})
	}
}

func TestParseCommandRejectsMalformedSyntax(t *testing.T) {
	invalid := []string{
		"",
		"INS",
		"EXEC",
		"SET",
		"IF",
		"IMAGE",
		"LINK",
		"HTML",
		"RAW-XML",
		"QUERY",
		"ELSE value",
		"END-IF value",
		"END-FOR 1invalid",
		"FOR item items",
		"FOR 1invalid IN items",
		"ALIAS invalid",
		"*invalid name",
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
	if command, err := parseCommand("END-FORWARD"); err != nil || command.Type != CommandInsert {
		t.Fatalf("END-FORWARD should be a normal insertion, got %#v, %v", command, err)
	}
}
