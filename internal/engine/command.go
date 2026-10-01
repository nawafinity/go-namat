package engine

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

// CommandType identifies a v1 template command.
type CommandType string

const (
	CommandInsert  CommandType = "insert"
	CommandLet     CommandType = "let"
	CommandIf      CommandType = "if"
	CommandElse    CommandType = "else"
	CommandEndIf   CommandType = "/if"
	CommandEach    CommandType = "each"
	CommandEndEach CommandType = "/each"
	CommandImage   CommandType = "@image"
	CommandLink    CommandType = "@link"
	CommandHTML    CommandType = "@html"
	CommandRawXML  CommandType = "@raw-xml"
)

// Command is a parsed command found in a template.
type Command struct {
	Raw        string
	Type       CommandType
	Expression string
	Variable   string
	Location   CommandLocation
}

// CommandLocation identifies an author-facing position inside one OOXML part.
type CommandLocation struct {
	Paragraph int
	Ordinal   int
	Start     int
	End       int
}

func parseCommand(raw string) (Command, error) {
	if !utf8.ValidString(raw) {
		return Command{}, fmt.Errorf("command is not valid UTF-8")
	}
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return Command{}, fmt.Errorf("empty command")
	}

	switch trimmed {
	case "#else":
		return Command{Raw: raw, Type: CommandElse}, nil
	case "/if":
		return Command{Raw: raw, Type: CommandEndIf}, nil
	case "/each":
		return Command{Raw: raw, Type: CommandEndEach}, nil
	}

	if strings.HasPrefix(trimmed, "#if") {
		expression, err := commandArgument(trimmed, "#if")
		if err != nil {
			return Command{}, err
		}
		return Command{Raw: raw, Type: CommandIf, Expression: expression}, nil
	}
	if strings.HasPrefix(trimmed, "#each") {
		rest, err := commandArgument(trimmed, "#each")
		if err != nil {
			return Command{}, err
		}
		expression, variable, ok := splitTopLevelKeyword(rest, "as")
		if !ok || expression == "" || !validName(variable) {
			return Command{}, fmt.Errorf("#each syntax is #each expression as name")
		}
		return Command{Raw: raw, Type: CommandEach, Expression: expression, Variable: variable}, nil
	}
	if strings.HasPrefix(trimmed, "#let") {
		rest, err := commandArgument(trimmed, "#let")
		if err != nil {
			return Command{}, err
		}
		name, expression, ok := splitTopLevelAssignment(rest)
		if !ok || !validName(name) || expression == "" {
			return Command{}, fmt.Errorf("#let syntax is #let name = expression")
		}
		return Command{Raw: raw, Type: CommandLet, Expression: expression, Variable: name}, nil
	}

	for _, kind := range []CommandType{CommandImage, CommandLink, CommandHTML, CommandRawXML} {
		prefix := string(kind)
		if strings.HasPrefix(trimmed, prefix) {
			expression, err := commandArgument(trimmed, prefix)
			if err != nil {
				return Command{}, err
			}
			return Command{Raw: raw, Type: kind, Expression: expression}, nil
		}
	}

	if strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "/") || strings.HasPrefix(trimmed, "@") {
		return Command{}, fmt.Errorf("unknown template directive %q", firstWord(trimmed))
	}
	return Command{Raw: raw, Type: CommandInsert, Expression: trimmed}, nil
}

func commandArgument(source, prefix string) (string, error) {
	if source == prefix {
		return "", fmt.Errorf("%s requires an expression", prefix)
	}
	if len(source) <= len(prefix) || !unicode.IsSpace(rune(source[len(prefix)])) {
		return "", fmt.Errorf("unknown template directive %q", firstWord(source))
	}
	argument := strings.TrimSpace(source[len(prefix):])
	if argument == "" {
		return "", fmt.Errorf("%s requires an expression", prefix)
	}
	return argument, nil
}

func firstWord(value string) string {
	if index := strings.IndexFunc(value, unicode.IsSpace); index >= 0 {
		return value[:index]
	}
	return value
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

func splitTopLevelKeyword(value, keyword string) (string, string, bool) {
	quote := rune(0)
	escaped := false
	depth := 0
	runes := []rune(value)
	for index, current := range runes {
		if quote != 0 {
			if escaped {
				escaped = false
				continue
			}
			if current == '\\' {
				escaped = true
				continue
			}
			if current == quote {
				quote = 0
			}
			continue
		}
		switch current {
		case '\'', '"', '`':
			quote = current
		case '(', '[', '{':
			depth++
		case ')', ']', '}':
			if depth > 0 {
				depth--
			}
		default:
			if depth != 0 || !unicode.IsSpace(current) {
				continue
			}
			start := index
			for start < len(runes) && unicode.IsSpace(runes[start]) {
				start++
			}
			end := start + len([]rune(keyword))
			if end > len(runes) || string(runes[start:end]) != keyword {
				continue
			}
			if end < len(runes) && !unicode.IsSpace(runes[end]) {
				continue
			}
			left := strings.TrimSpace(string(runes[:index]))
			right := strings.TrimSpace(string(runes[end:]))
			return left, right, true
		}
	}
	return "", "", false
}

func splitTopLevelAssignment(value string) (string, string, bool) {
	quote := rune(0)
	escaped := false
	depth := 0
	for index, current := range value {
		if quote != 0 {
			if escaped {
				escaped = false
				continue
			}
			if current == '\\' {
				escaped = true
				continue
			}
			if current == quote {
				quote = 0
			}
			continue
		}
		switch current {
		case '\'', '"', '`':
			quote = current
		case '(', '[', '{':
			depth++
		case ')', ']', '}':
			if depth > 0 {
				depth--
			}
		case '=':
			if depth == 0 {
				return strings.TrimSpace(value[:index]), strings.TrimSpace(value[index+1:]), true
			}
		}
	}
	return "", "", false
}
