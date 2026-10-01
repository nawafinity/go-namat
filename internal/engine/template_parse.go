package engine

import (
	"errors"
	"fmt"
	"strings"
)

type commandSyntaxLocationError struct {
	Location CommandLocation
	Err      error
}

func (e *commandSyntaxLocationError) Error() string { return e.Err.Error() }
func (e *commandSyntaxLocationError) Unwrap() error { return e.Err }

type commandSpan struct {
	Start   int
	End     int
	Command Command
}

type commandRange struct {
	start     int
	bodyStart int
	bodyEnd   int
	end       int
	closed    bool
}

// scanCommandRanges is the single delimiter scanner used before and after
// Word-run normalization. A close delimiter is recognized only outside quoted
// strings and at expression nesting depth zero.
func scanCommandRanges(text, open, close string) []commandRange {
	var result []commandRange
	for position := 0; position < len(text); {
		relative := strings.Index(text[position:], open)
		if relative < 0 {
			break
		}
		start := position + relative
		bodyStart := start + len(open)
		quote := byte(0)
		escaped := false
		depth := 0
		bodyEnd := len(text)
		end := len(text)
		closed := false
		for index := bodyStart; index < len(text); index++ {
			current := text[index]
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
			if current == '\'' || current == '"' || current == '`' {
				quote = current
				continue
			}
			if depth == 0 && strings.HasPrefix(text[index:], close) {
				bodyEnd = index
				end = index + len(close)
				closed = true
				break
			}
			switch current {
			case '(', '[', '{':
				depth++
			case ')', ']', '}':
				if depth > 0 {
					depth--
				}
			}
		}
		result = append(result, commandRange{start: start, bodyStart: bodyStart, bodyEnd: bodyEnd, end: end, closed: closed})
		if !closed {
			break
		}
		position = end
	}
	return result
}

// normalizeCommandFragments joins a command that Word has split across text
// nodes, including the less common case where the split crosses paragraphs.
// Only text between an unmatched opening delimiter and its closing delimiter
// is moved; ordinary document text keeps its original node and formatting.
func normalizeCommandFragments(root *xmlNode, options Options) {
	nodes := textNodes(root)
	if len(nodes) == 0 {
		return
	}

	type nodeSpan struct {
		start int
		end   int
	}
	spans := make([]nodeSpan, len(nodes))
	var joined strings.Builder
	for index, node := range nodes {
		spans[index].start = joined.Len()
		joined.WriteString(node.Data)
		spans[index].end = joined.Len()
	}
	text := joined.String()

	var ranges []commandRange
	for _, scanned := range scanCommandRanges(text, options.OpenDelimiter, options.CloseDelimiter) {
		start, end := scanned.start, scanned.end
		startNode, endNode := -1, -1
		for index, span := range spans {
			if startNode < 0 && start >= span.start && start < span.end {
				startNode = index
			}
			last := end - 1
			if endNode < 0 && last >= span.start && last < span.end {
				endNode = index
			}
		}
		if startNode >= 0 && endNode >= 0 && startNode != endNode {
			ranges = append(ranges, scanned)
		}
	}
	if len(ranges) == 0 {
		return
	}

	for index, node := range nodes {
		span := spans[index]
		var rebuilt strings.Builder
		position := span.start
		for _, command := range ranges {
			if command.end <= span.start || command.start >= span.end {
				continue
			}
			before := command.start
			if position < before {
				rebuilt.WriteString(text[position:before])
			}
			if command.start >= span.start && command.start < span.end {
				rebuilt.WriteString(text[command.start:command.end])
			}
			if command.end > position {
				position = command.end
				if position > span.end {
					position = span.end
				}
			}
		}
		if position < span.end {
			rebuilt.WriteString(text[position:span.end])
		}
		node.Data = rebuilt.String()
	}
}

func commandsInXML(root *xmlNode, options Options) ([]Command, error) {
	var result []Command
	ordinal := 0
	for paragraphIndex, paragraph := range root.descendants("p") {
		spans, err := commandSpans(textOfParagraph(paragraph), options)
		if err != nil {
			var located *commandSyntaxLocationError
			if errors.As(err, &located) {
				located.Location.Paragraph = paragraphIndex + 1
				located.Location.Ordinal = ordinal + 1
			}
			return nil, err
		}
		for _, span := range spans {
			command := span.Command
			command.Location = CommandLocation{Paragraph: paragraphIndex + 1, Ordinal: ordinal + 1, Start: span.Start, End: span.End}
			paragraph.commandLocations = append(paragraph.commandLocations, command.Location)
			result = append(result, command)
			ordinal++
		}
	}
	return result, nil
}

func paragraphCommandSpans(paragraph *xmlNode, text string, options Options) ([]commandSpan, error) {
	spans, err := commandSpans(text, options)
	if err != nil {
		return nil, err
	}
	for index := range spans {
		if index < len(paragraph.commandLocations) {
			spans[index].Command.Location = paragraph.commandLocations[index]
		}
	}
	return spans, nil
}

func commandSpans(text string, options Options) ([]commandSpan, error) {
	var result []commandSpan
	for _, scanned := range scanCommandRanges(text, options.OpenDelimiter, options.CloseDelimiter) {
		if !scanned.closed {
			return nil, &commandSyntaxLocationError{Location: CommandLocation{Start: scanned.start, End: len(text)}, Err: fmt.Errorf("unterminated command beginning at byte %d", scanned.start)}
		}
		command, err := parseCommand(text[scanned.bodyStart:scanned.bodyEnd])
		if err != nil {
			return nil, &commandSyntaxLocationError{Location: CommandLocation{Start: scanned.start, End: scanned.end}, Err: fmt.Errorf("command at byte %d: %w", scanned.start, err)}
		}
		result = append(result, commandSpan{Start: scanned.start, End: scanned.end, Command: command})
	}
	return result, nil
}

func structuralCommand(node *xmlNode, options Options) (Command, bool, error) {
	var paragraphs []*xmlNode
	switch {
	case node.is("p"):
		paragraphs = []*xmlNode{node}
	case node.is("tr"):
		paragraphs = rowParagraphs(node)
	default:
		return Command{}, false, nil
	}

	var found *Command
	for _, paragraph := range paragraphs {
		text := strings.TrimSpace(textOfParagraph(paragraph))
		spans, err := paragraphCommandSpans(paragraph, text, options)
		if err != nil {
			return Command{}, false, err
		}
		if len(spans) != 1 || spans[0].Start != 0 || spans[0].End != len(text) {
			continue
		}
		command := spans[0].Command
		switch command.Type {
		case CommandEach, CommandEndEach, CommandIf, CommandElse, CommandEndIf:
			if found != nil {
				return Command{}, false, fmt.Errorf("multiple structural commands in one block")
			}
			found = &command
		}
	}
	if found == nil {
		return Command{}, false, nil
	}
	return *found, true, nil
}

// rowParagraphs returns paragraphs owned by row itself. Descendant rows belong
// to nested tables and must be evaluated independently; treating their marker
// paragraphs as markers on the outer row makes valid nested loops ambiguous.
func rowParagraphs(row *xmlNode) []*xmlNode {
	var paragraphs []*xmlNode
	var walk func(*xmlNode)
	walk = func(node *xmlNode) {
		for _, child := range node.Children {
			if child.is("tr") {
				continue
			}
			if child.is("p") {
				paragraphs = append(paragraphs, child)
				continue
			}
			walk(child)
		}
	}
	walk(row)
	return paragraphs
}

func standaloneCommand(node *xmlNode, options Options) (Command, bool, error) {
	if !node.is("p") {
		return Command{}, false, nil
	}
	text := strings.TrimSpace(textOfParagraph(node))
	spans, err := paragraphCommandSpans(node, text, options)
	if err != nil {
		return Command{}, false, err
	}
	if len(spans) != 1 || spans[0].Start != 0 || spans[0].End != len(text) {
		return Command{}, false, nil
	}
	return spans[0].Command, true, nil
}

func validateStructure(root *xmlNode, options Options) error {
	return validateNodeStructure(root, options)
}

func validateNodeStructure(parent *xmlNode, options Options) error {
	stack := make([]Command, 0)
	for _, child := range parent.Children {
		command, ok, err := structuralCommand(child, options)
		if err != nil {
			return err
		}
		if ok {
			switch command.Type {
			case CommandIf, CommandEach:
				stack = append(stack, command)
			case CommandElse:
				if len(stack) == 0 || stack[len(stack)-1].Type != CommandIf {
					return fmt.Errorf("#else without matching #if")
				}
			case CommandEndIf:
				if len(stack) == 0 || stack[len(stack)-1].Type != CommandIf {
					return fmt.Errorf("/if without matching #if")
				}
				stack = stack[:len(stack)-1]
			case CommandEndEach:
				if len(stack) == 0 || stack[len(stack)-1].Type != CommandEach {
					return fmt.Errorf("/each without matching #each")
				}
				stack = stack[:len(stack)-1]
			}
			continue
		}
		if err := validateNodeStructure(child, options); err != nil {
			return err
		}
	}
	if len(stack) > 0 {
		return fmt.Errorf("%s has no matching end command", stack[len(stack)-1].Type)
	}
	return nil
}
