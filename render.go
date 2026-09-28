package namat

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/nawafinity/go-namat/internal/expr"
)

type renderState struct {
	ctx       context.Context
	template  *Template
	rootData  any
	variables map[string]any
	functions map[string]expr.Function
	counter   *renderCounter
}

type renderCounter struct {
	iterations int
}

type textReplacement struct {
	start, end int
	value      string
}

func (s *renderState) child() *renderState {
	variables := make(map[string]any, len(s.variables)+2)
	for key, value := range s.variables {
		variables[key] = value
	}
	return &renderState{
		ctx:       s.ctx,
		template:  s.template,
		rootData:  s.rootData,
		variables: variables,
		functions: s.functions,
		counter:   s.counter,
	}
}

func (s *renderState) processNode(node *xmlNode) error {
	if err := s.ctx.Err(); err != nil {
		return err
	}
	children, err := s.processSequence(node.Children)
	if err != nil {
		return err
	}
	node.Children = children
	if node.is("p") {
		return s.renderParagraph(node)
	}
	return nil
}

func (s *renderState) processSequence(children []*xmlNode) ([]*xmlNode, error) {
	result := make([]*xmlNode, 0, len(children))
	for index := 0; index < len(children); {
		if err := s.ctx.Err(); err != nil {
			return nil, err
		}
		command, structural, err := structuralCommand(children[index], s.template.options)
		if err != nil {
			return nil, err
		}
		if !structural || (command.Type != CommandIf && command.Type != CommandFor) {
			child := children[index]
			if err := s.processNode(child); err != nil {
				return nil, err
			}
			result = append(result, child)
			index++
			continue
		}

		end, elseIndex, err := findStructuralEnd(children, index, command.Type, s.template.options)
		if err != nil {
			return nil, err
		}
		switch command.Type {
		case CommandIf:
			value, err := s.evaluate(command.Expression)
			if err != nil {
				return nil, s.commandError(command, err)
			}
			start, stop := index+1, end
			if expressionTruthy(value) {
				if elseIndex >= 0 {
					stop = elseIndex
				}
			} else if elseIndex >= 0 {
				start = elseIndex + 1
			} else {
				start = end
			}
			segment := cloneNodes(children[start:stop])
			processed, err := s.processSequence(segment)
			if err != nil {
				return nil, err
			}
			result = append(result, processed...)
		case CommandFor:
			value, err := s.evaluate(command.Expression)
			if err != nil {
				return nil, s.commandError(command, err)
			}
			items, err := iterable(value)
			if err != nil {
				return nil, s.commandError(command, err)
			}
			for itemIndex, item := range items {
				s.counter.iterations++
				if s.counter.iterations > s.template.options.MaxIterations {
					return nil, fmt.Errorf("maximum loop iterations exceeded (%d)", s.template.options.MaxIterations)
				}
				childState := s.child()
				childState.variables[command.Variable] = item
				childState.variables["$idx"] = itemIndex
				segment := cloneNodes(children[index+1 : end])
				processed, err := childState.processSequence(segment)
				if err != nil {
					return nil, err
				}
				result = append(result, processed...)
			}
		}
		index = end + 1
	}
	return result, nil
}

func findStructuralEnd(children []*xmlNode, start int, kind CommandType, options Options) (end int, elseIndex int, err error) {
	depth := 0
	elseIndex = -1
	for index := start + 1; index < len(children); index++ {
		command, ok, commandErr := structuralCommand(children[index], options)
		if commandErr != nil {
			return 0, -1, commandErr
		}
		if !ok {
			continue
		}
		switch command.Type {
		case kind:
			depth++
		case CommandElse:
			if kind == CommandIf && depth == 0 {
				if elseIndex >= 0 {
					return 0, -1, fmt.Errorf("IF block contains more than one ELSE")
				}
				elseIndex = index
			}
		case matchingEnd(kind):
			if depth == 0 {
				return index, elseIndex, nil
			}
			depth--
		}
	}
	return 0, -1, fmt.Errorf("%s has no matching end command", kind)
}

func matchingEnd(kind CommandType) CommandType {
	if kind == CommandIf {
		return CommandEndIf
	}
	return CommandEndFor
}

func (s *renderState) renderParagraph(paragraph *xmlNode) error {
	text := textOfParagraph(paragraph)
	spans, err := commandSpans(text, s.template.options)
	if err != nil {
		return err
	}
	if len(spans) == 0 {
		return nil
	}
	replacements := make([]textReplacement, 0, len(spans))
	for _, span := range spans {
		command := span.Command
		var value any
		switch command.Type {
		case CommandExec, CommandSet:
			name, expression, err := parseAssignment(command.Expression)
			if err != nil {
				return s.commandError(command, err)
			}
			value, err = s.evaluate(expression)
			if err != nil {
				return s.commandError(command, err)
			}
			s.variables[name] = value
			value = ""
		case CommandInsert:
			value, err = s.evaluate(command.Expression)
			if err != nil {
				if s.template.options.ErrorHandler == nil {
					return s.commandError(command, err)
				}
				value, err = s.template.options.ErrorHandler(command.Raw, err)
				if err != nil {
					return s.commandError(command, err)
				}
			}
			if value == nil && s.template.options.RejectNullish {
				return s.commandError(command, fmt.Errorf("expression returned null"))
			}
		case CommandImage, CommandLink, CommandHTML, CommandRawXML:
			return s.commandError(command, fmt.Errorf("%s rendering is not implemented yet", command.Type))
		case CommandIf, CommandElse, CommandEndIf, CommandFor, CommandEndFor:
			// Structural commands are removed by sequence processing. Reaching this
			// path means the marker is inline, which will be supported separately.
			return s.commandError(command, fmt.Errorf("inline structural commands are not supported yet"))
		default:
			return s.commandError(command, fmt.Errorf("unsupported command type %s", command.Type))
		}
		replacements = append(replacements, textReplacement{start: span.Start, end: span.End, value: formatValue(value)})
	}
	return replaceParagraphText(paragraph, replacements)
}

func (s *renderState) evaluate(source string) (any, error) {
	program, err := s.template.program(source)
	if err != nil {
		return nil, err
	}
	return program.Eval(&expr.Context{Root: s.rootData, Variables: s.variables, Functions: s.functions})
}

func (s *renderState) commandError(command Command, err error) error {
	return &Error{Command: command.Raw, Err: err}
}

func replaceParagraphText(paragraph *xmlNode, replacements []textReplacement) error {
	nodes := textNodes(paragraph)
	if len(nodes) == 0 {
		return fmt.Errorf("command paragraph contains no text nodes")
	}
	for i := len(replacements) - 1; i >= 0; i-- {
		replacement := replacements[i]
		startNode, startOffset, ok := locateTextOffset(nodes, replacement.start, true)
		if !ok {
			return fmt.Errorf("cannot locate command start in Word runs")
		}
		endNode, endOffset, ok := locateTextOffset(nodes, replacement.end, false)
		if !ok {
			return fmt.Errorf("cannot locate command end in Word runs")
		}
		if startNode == endNode {
			nodes[startNode].Data = nodes[startNode].Data[:startOffset] + replacement.value + nodes[startNode].Data[endOffset:]
			continue
		}
		nodes[startNode].Data = nodes[startNode].Data[:startOffset] + replacement.value
		for nodeIndex := startNode + 1; nodeIndex < endNode; nodeIndex++ {
			nodes[nodeIndex].Data = ""
		}
		nodes[endNode].Data = nodes[endNode].Data[endOffset:]
	}
	return nil
}

func locateTextOffset(nodes []*xmlNode, offset int, boundaryToNext bool) (int, int, bool) {
	position := 0
	for index, node := range nodes {
		next := position + len(node.Data)
		if offset < next || (offset == next && (!boundaryToNext || index == len(nodes)-1)) {
			return index, offset - position, true
		}
		position = next
	}
	return 0, 0, false
}

func cloneNodes(nodes []*xmlNode) []*xmlNode {
	result := make([]*xmlNode, len(nodes))
	for i, node := range nodes {
		result[i] = node.clone()
	}
	return result
}

func iterable(value any) ([]any, error) {
	if value == nil {
		return nil, nil
	}
	v := reflect.ValueOf(value)
	for v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface {
		if v.IsNil() {
			return nil, nil
		}
		v = v.Elem()
	}
	switch v.Kind() {
	case reflect.Array, reflect.Slice:
		items := make([]any, v.Len())
		for i := 0; i < v.Len(); i++ {
			items[i] = v.Index(i).Interface()
		}
		return items, nil
	default:
		return nil, fmt.Errorf("FOR expects an array or slice, got %T", value)
	}
}

func expressionTruthy(value any) bool {
	if value == nil {
		return false
	}
	switch v := value.(type) {
	case bool:
		return v
	case string:
		return strings.TrimSpace(v) != ""
	case float64:
		return v != 0
	case float32:
		return v != 0
	case int:
		return v != 0
	default:
		return true
	}
}
