package engine

import (
	"fmt"
	"strings"
)

type commandSpan struct {
	Start   int
	End     int
	Command Command
}

// normalizeCommandFragments joins a command that Word has split across text
// nodes, including the less common case where the split crosses paragraphs.
// Only text between an unmatched opening delimiter and its closing delimiter
// is moved; ordinary document text keeps its original node and formatting.
func normalizeCommandFragments(root *xmlNode, options Options) {
	nodes := textNodes(root)
	var target *xmlNode
	for _, node := range nodes {
		searchFrom := 0
		for {
			if target != nil {
				end := strings.Index(node.Data[searchFrom:], options.CloseDelimiter)
				if end < 0 {
					target.Data += node.Data[searchFrom:]
					node.Data = node.Data[:searchFrom]
					break
				}
				end += searchFrom + len(options.CloseDelimiter)
				target.Data += node.Data[searchFrom:end]
				node.Data = node.Data[:searchFrom] + node.Data[end:]
				target = nil
				searchFrom = 0
				continue
			}

			start := strings.Index(node.Data[searchFrom:], options.OpenDelimiter)
			if start < 0 {
				break
			}
			start += searchFrom
			bodyStart := start + len(options.OpenDelimiter)
			end := strings.Index(node.Data[bodyStart:], options.CloseDelimiter)
			if end >= 0 {
				searchFrom = bodyStart + end + len(options.CloseDelimiter)
				continue
			}
			target = node
			break
		}
	}
}

func commandsInXML(root *xmlNode, options Options) ([]Command, error) {
	var result []Command
	for _, paragraph := range root.descendants("p") {
		spans, err := commandSpans(textOfParagraph(paragraph), options)
		if err != nil {
			return nil, err
		}
		for _, span := range spans {
			result = append(result, span.Command)
		}
	}
	return result, nil
}

func commandSpans(text string, options Options) ([]commandSpan, error) {
	var result []commandSpan
	position := 0
	for {
		startRelative := strings.Index(text[position:], options.OpenDelimiter)
		if startRelative < 0 {
			return result, nil
		}
		start := position + startRelative
		bodyStart := start + len(options.OpenDelimiter)
		endRelative := strings.Index(text[bodyStart:], options.CloseDelimiter)
		if endRelative < 0 {
			return nil, fmt.Errorf("unterminated command beginning at byte %d", start)
		}
		end := bodyStart + endRelative + len(options.CloseDelimiter)
		command, err := parseCommand(text[bodyStart : bodyStart+endRelative])
		if err != nil {
			return nil, fmt.Errorf("command at byte %d: %w", start, err)
		}
		result = append(result, commandSpan{Start: start, End: end, Command: command})
		position = end
	}
}

func structuralCommand(node *xmlNode, options Options) (Command, bool, error) {
	var paragraphs []*xmlNode
	switch {
	case node.is("p"):
		paragraphs = []*xmlNode{node}
	case node.is("tr"):
		paragraphs = node.descendants("p")
	default:
		return Command{}, false, nil
	}

	var found *Command
	for _, paragraph := range paragraphs {
		text := strings.TrimSpace(textOfParagraph(paragraph))
		spans, err := commandSpans(text, options)
		if err != nil {
			return Command{}, false, err
		}
		if len(spans) != 1 || spans[0].Start != 0 || spans[0].End != len(text) {
			continue
		}
		command := spans[0].Command
		switch command.Type {
		case CommandFor, CommandEndFor, CommandIf, CommandElse, CommandEndIf:
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

func standaloneCommand(node *xmlNode, options Options) (Command, bool, error) {
	if !node.is("p") {
		return Command{}, false, nil
	}
	text := strings.TrimSpace(textOfParagraph(node))
	spans, err := commandSpans(text, options)
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
			case CommandIf, CommandFor:
				stack = append(stack, command)
			case CommandElse:
				if len(stack) == 0 || stack[len(stack)-1].Type != CommandIf {
					return fmt.Errorf("ELSE without matching IF")
				}
			case CommandEndIf:
				if len(stack) == 0 || stack[len(stack)-1].Type != CommandIf {
					return fmt.Errorf("END-IF without matching IF")
				}
				stack = stack[:len(stack)-1]
			case CommandEndFor:
				if len(stack) == 0 || stack[len(stack)-1].Type != CommandFor {
					return fmt.Errorf("END-FOR without matching FOR")
				}
				startVariable := strings.TrimPrefix(stack[len(stack)-1].Variable, "$")
				if command.Variable != "" && command.Variable != startVariable {
					return fmt.Errorf("END-FOR %s does not match FOR %s", command.Variable, startVariable)
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
