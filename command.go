package namat

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

// CommandType identifies a template command.
type CommandType string

const (
	CommandInsert   CommandType = "INS"
	CommandExec     CommandType = "EXEC"
	CommandSet      CommandType = "SET"
	CommandIf       CommandType = "IF"
	CommandElse     CommandType = "ELSE"
	CommandEndIf    CommandType = "END-IF"
	CommandFor      CommandType = "FOR"
	CommandEndFor   CommandType = "END-FOR"
	CommandImage    CommandType = "IMAGE"
	CommandLink     CommandType = "LINK"
	CommandHTML     CommandType = "HTML"
	CommandRawXML   CommandType = "RAW-XML"
	CommandQuery    CommandType = "QUERY"
	CommandAlias    CommandType = "ALIAS"
	CommandAliasRef CommandType = "ALIAS-REF"
)

// Command is a parsed command found in a template.
type Command struct {
	Raw        string
	Type       CommandType
	Expression string
	Variable   string
}

func parseCommand(raw string) (Command, error) {
	if !utf8.ValidString(raw) {
		return Command{}, fmt.Errorf("command is not valid UTF-8")
	}
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
	if strings.HasPrefix(upper, string(CommandElse)+" ") || strings.HasPrefix(upper, string(CommandEndIf)+" ") {
		return Command{}, fmt.Errorf("%s does not accept arguments", strings.Fields(upper)[0])
	}
	if strings.HasPrefix(upper, string(CommandEndFor)) {
		variable := strings.TrimPrefix(strings.TrimSpace(trimmed[len(CommandEndFor):]), "$")
		if variable != "" && !validName(variable) {
			return Command{}, fmt.Errorf("invalid END-FOR variable %q", variable)
		}
		return Command{Raw: raw, Type: CommandEndFor, Variable: variable}, nil
	}
	if strings.HasPrefix(upper, string(CommandFor)+" ") {
		rest := strings.TrimSpace(trimmed[len(CommandFor):])
		parts := splitKeyword(rest, " IN ")
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return Command{}, fmt.Errorf("FOR syntax is FOR name IN expression")
		}
		variable := strings.TrimPrefix(strings.TrimSpace(parts[0]), "$")
		if !validName(variable) {
			return Command{}, fmt.Errorf("invalid FOR variable %q", variable)
		}
		return Command{Raw: raw, Type: CommandFor, Variable: "$" + variable, Expression: strings.TrimSpace(parts[1])}, nil
	}
	if strings.HasPrefix(upper, string(CommandAlias)+" ") {
		rest := strings.TrimSpace(trimmed[len(CommandAlias):])
		separator := strings.IndexAny(rest, " \t\r\n")
		if separator <= 0 || strings.TrimSpace(rest[separator:]) == "" {
			return Command{}, fmt.Errorf("ALIAS syntax is ALIAS name command")
		}
		name := strings.TrimSpace(rest[:separator])
		if !validName(name) {
			return Command{}, fmt.Errorf("invalid alias name %q", name)
		}
		return Command{Raw: raw, Type: CommandAlias, Variable: name, Expression: strings.TrimSpace(rest[separator:])}, nil
	}
	if strings.HasPrefix(trimmed, "*") {
		name := strings.TrimSpace(trimmed[1:])
		if !validName(name) {
			return Command{}, fmt.Errorf("alias reference syntax is *name")
		}
		return Command{Raw: raw, Type: CommandAliasRef, Variable: name}, nil
	}
	for _, kind := range []CommandType{CommandInsert, CommandExec, CommandSet, CommandIf, CommandImage, CommandLink, CommandHTML, CommandRawXML, CommandQuery} {
		prefix := string(kind) + " "
		if strings.HasPrefix(upper, prefix) {
			return Command{Raw: raw, Type: kind, Expression: strings.TrimSpace(trimmed[len(prefix):])}, nil
		}
	}
	if strings.HasPrefix(trimmed, "=") {
		return Command{Raw: raw, Type: CommandInsert, Expression: strings.TrimSpace(trimmed[1:])}, nil
	}
	if strings.HasPrefix(trimmed, "!") {
		return Command{Raw: raw, Type: CommandExec, Expression: strings.TrimSpace(trimmed[1:])}, nil
	}
	return Command{Raw: raw, Type: CommandInsert, Expression: trimmed}, nil
}

func validName(value string) bool {
	for index, r := range []rune(value) {
		if index == 0 {
			if r != '_' && !unicode.IsLetter(r) {
				return false
			}
			continue
		}
		if r != '_' && !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return false
		}
	}
	return value != ""
}

func splitKeyword(value, keyword string) []string {
	for index := range value {
		end := index + len(keyword)
		if end <= len(value) && strings.EqualFold(value[index:end], keyword) {
			return []string{value[:index], value[end:]}
		}
	}
	return nil
}
