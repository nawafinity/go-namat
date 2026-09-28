package namat

import (
	"fmt"
	"strings"
)

// CommandType identifies a template command.
type CommandType string

const (
	CommandInsert CommandType = "INS"
	CommandExec   CommandType = "EXEC"
	CommandSet    CommandType = "SET"
	CommandIf     CommandType = "IF"
	CommandElse   CommandType = "ELSE"
	CommandEndIf  CommandType = "END-IF"
	CommandFor    CommandType = "FOR"
	CommandEndFor CommandType = "END-FOR"
	CommandImage  CommandType = "IMAGE"
	CommandLink   CommandType = "LINK"
	CommandHTML   CommandType = "HTML"
	CommandRawXML CommandType = "RAW-XML"
)

// Command is a parsed command found in a template.
type Command struct {
	Raw        string
	Type       CommandType
	Expression string
	Variable   string
}

func parseCommand(raw string) (Command, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return Command{}, fmt.Errorf("empty command")
	}
	upper := strings.ToUpper(trimmed)
	for _, simple := range []CommandType{CommandElse, CommandEndIf} {
		if upper == string(simple) {
			return Command{Raw: raw, Type: simple}, nil
		}
	}
	if strings.HasPrefix(upper, string(CommandEndFor)) {
		return Command{Raw: raw, Type: CommandEndFor, Variable: strings.TrimSpace(trimmed[len(CommandEndFor):])}, nil
	}
	if strings.HasPrefix(upper, string(CommandFor)+" ") {
		rest := strings.TrimSpace(trimmed[len(CommandFor):])
		parts := splitKeyword(rest, " IN ")
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return Command{}, fmt.Errorf("FOR syntax is FOR name IN expression")
		}
		return Command{Raw: raw, Type: CommandFor, Variable: "$" + strings.TrimPrefix(strings.TrimSpace(parts[0]), "$"), Expression: strings.TrimSpace(parts[1])}, nil
	}
	for _, kind := range []CommandType{CommandInsert, CommandExec, CommandSet, CommandIf, CommandImage, CommandLink, CommandHTML, CommandRawXML} {
		prefix := string(kind) + " "
		if strings.HasPrefix(upper, prefix) {
			return Command{Raw: raw, Type: kind, Expression: strings.TrimSpace(trimmed[len(prefix):])}, nil
		}
	}
	if strings.HasPrefix(trimmed, "=") {
		return Command{Raw: raw, Type: CommandInsert, Expression: strings.TrimSpace(trimmed[1:])}, nil
	}
	return Command{Raw: raw, Type: CommandInsert, Expression: trimmed}, nil
}

func splitKeyword(value, keyword string) []string {
	upper := strings.ToUpper(value)
	index := strings.Index(upper, keyword)
	if index < 0 {
		return nil
	}
	return []string{value[:index], value[index+len(keyword):]}
}
