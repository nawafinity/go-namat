package engine

import (
	"context"
	"fmt"
	"reflect"

	"github.com/nawafinity/go-namat/internal/expr"
	valuetype "github.com/nawafinity/go-namat/internal/value"
)

type renderState struct {
	ctx       context.Context
	template  *Template
	rootData  any
	variables map[string]any
	declared  map[string]struct{}
	functions map[string]expr.Function
	counter   *renderCounter
	pkg       *docxPackage
	partName  string
}

type renderCounter struct {
	iterations      int
	resources       int
	evaluationSteps int
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
		declared:  make(map[string]struct{}),
		functions: s.functions,
		counter:   s.counter,
		pkg:       s.pkg,
		partName:  s.partName,
	}
}

func (s *renderState) declare(name string, value any) error {
	if _, exists := s.declared[name]; exists {
		return fmt.Errorf("variable %q is already declared in this scope", name)
	}
	s.declared[name] = struct{}{}
	s.variables[name] = value
	return nil
}

func (s *renderState) declareLoopValues(name string, item any, loop map[string]any) error {
	if err := s.declare(name, item); err != nil {
		return err
	}
	return s.declare("loop", loop)
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
		blockCommand, standalone, err := standaloneCommand(children[index], s.template.options)
		if err != nil {
			return nil, err
		}
		if standalone {
			switch blockCommand.Type {
			case CommandLet:
				value, evalErr := s.evaluate(blockCommand.Expression)
				if evalErr != nil {
					return nil, s.commandError(blockCommand, evalErr)
				}
				if declareErr := s.declare(blockCommand.Variable, value); declareErr != nil {
					return nil, s.commandError(blockCommand, declareErr)
				}
				index++
				continue
			case CommandHTML:
				if s.partName != "word/document.xml" {
					return nil, s.commandError(blockCommand, fmt.Errorf("@html is supported only in word/document.xml"))
				}
				value, evalErr := s.evaluate(blockCommand.Expression)
				if evalErr != nil {
					return nil, s.commandError(blockCommand, evalErr)
				}
				node, nodeErr := s.htmlNode(value)
				if nodeErr != nil {
					return nil, s.commandError(blockCommand, nodeErr)
				}
				result = append(result, node)
				index++
				continue
			case CommandRawXML:
				if !s.template.options.AllowRawXML {
					return nil, s.commandError(blockCommand, fmt.Errorf("@raw-xml is disabled; set AllowRawXML to enable it"))
				}
				value, evalErr := s.evaluate(blockCommand.Expression)
				if evalErr != nil {
					return nil, s.commandError(blockCommand, evalErr)
				}
				rawXML, formatErr := expr.Format(value)
				if formatErr != nil {
					return nil, s.commandError(blockCommand, fmt.Errorf("@raw-xml value: %w", formatErr))
				}
				nodes, parseErr := parseXMLFragment(rawXML)
				if parseErr != nil {
					return nil, s.commandError(blockCommand, parseErr)
				}
				result = append(result, nodes...)
				index++
				continue
			}
		}

		command, structural, err := structuralCommand(children[index], s.template.options)
		if err != nil {
			return nil, err
		}
		if !structural || (command.Type != CommandIf && command.Type != CommandEach) {
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
			condition, ok := expr.AsBool(value)
			if !ok {
				return nil, s.commandError(command, fmt.Errorf("#if condition must be bool, got %T", value))
			}
			if condition {
				if elseIndex >= 0 {
					stop = elseIndex
				}
			} else if elseIndex >= 0 {
				start = elseIndex + 1
			} else {
				start = end
			}
			segment := cloneNodes(children[start:stop])
			processed, err := s.child().processSequence(segment)
			if err != nil {
				return nil, err
			}
			result = append(result, processed...)
		case CommandEach:
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
					return nil, fmt.Errorf("%w: maximum loop iterations exceeded (%d)", ErrSecurityLimit, s.template.options.MaxIterations)
				}
				childState := s.child()
				parentLoop, _ := s.variables["loop"]
				loop := map[string]any{
					"index":  int64(itemIndex),
					"number": int64(itemIndex + 1),
					"first":  itemIndex == 0,
					"last":   itemIndex == len(items)-1,
					"parent": parentLoop,
				}
				if err := childState.declareLoopValues(command.Variable, item, loop); err != nil {
					return nil, s.commandError(command, err)
				}
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
					return 0, -1, fmt.Errorf("#if block contains more than one #else")
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
	return CommandEndEach
}

func (s *renderState) renderParagraph(paragraph *xmlNode) error {
	text := textOfParagraph(paragraph)
	spans, err := paragraphCommandSpans(paragraph, text, s.template.options)
	if err != nil {
		return err
	}
	if len(spans) == 0 {
		return nil
	}
	actions := make([]paragraphAction, 0, len(spans))
	for _, span := range spans {
		command := span.Command
		var value any
		switch command.Type {
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
			if kind := expr.KindOf(value); kind == valuetype.Missing || kind == valuetype.Null {
				nullErr := fmt.Errorf("%w: expression returned %s; use default", ErrNullishResult, kind)
				if s.template.options.ErrorHandler == nil {
					return s.commandError(command, nullErr)
				}
				value, err = s.template.options.ErrorHandler(command.Raw, nullErr)
				if err != nil {
					return s.commandError(command, err)
				}
			}
			if isObjectResult(value) {
				objectErr := fmt.Errorf("%w: insertion returned %T", ErrObjectResult, value)
				if s.template.options.ErrorHandler == nil {
					return s.commandError(command, objectErr)
				}
				value, err = s.template.options.ErrorHandler(command.Raw, objectErr)
				if err != nil {
					return s.commandError(command, err)
				}
			}
			formatted, formatErr := expr.Format(value)
			if formatErr != nil {
				return s.commandError(command, formatErr)
			}
			actions = append(actions, paragraphAction{start: span.Start, end: span.End, text: &formatted})
			continue
		case CommandImage:
			value, err = s.evaluate(command.Expression)
			if err != nil {
				return s.commandError(command, err)
			}
			node, nodeErr := s.imageNode(value)
			if nodeErr != nil {
				return s.commandError(command, nodeErr)
			}
			actions = append(actions, paragraphAction{start: span.Start, end: span.End, node: node})
			continue
		case CommandLink:
			value, err = s.evaluate(command.Expression)
			if err != nil {
				return s.commandError(command, err)
			}
			node, nodeErr := s.linkNode(value)
			if nodeErr != nil {
				return s.commandError(command, nodeErr)
			}
			actions = append(actions, paragraphAction{start: span.Start, end: span.End, node: node})
			continue
		case CommandHTML, CommandRawXML:
			return s.commandError(command, fmt.Errorf("%s must occupy its own paragraph", command.Type))
		case CommandLet, CommandIf, CommandElse, CommandEndIf, CommandEach, CommandEndEach:
			// Structural commands are removed by sequence processing. Reaching this
			// path means a directive was placed inline.
			return s.commandError(command, fmt.Errorf("%s must occupy its own paragraph or table row", command.Type))
		}
	}
	return finalizeParagraph(paragraph, actions, s.template.options)
}

func finalizeParagraph(paragraph *xmlNode, actions []paragraphAction, options Options) error {
	if err := applyParagraphActions(paragraph, actions); err != nil {
		return err
	}
	return expandTextMarkup(paragraph, options)
}

func applyParagraphActions(paragraph *xmlNode, actions []paragraphAction) error {
	for index := len(actions) - 1; index >= 0; index-- {
		if err := applyParagraphAction(paragraph, actions[index]); err != nil {
			return err
		}
	}
	return nil
}

func applyParagraphAction(paragraph *xmlNode, action paragraphAction) error {
	if action.node != nil {
		return replaceTextRangeWithNode(paragraph, action.start, action.end, action.node)
	}
	if action.text == nil {
		return fmt.Errorf("paragraph action has neither text nor node")
	}
	return replaceParagraphText(paragraph, []textReplacement{{start: action.start, end: action.end, value: *action.text}})
}

func (s *renderState) evaluate(source string) (any, error) {
	if err := s.ctx.Err(); err != nil {
		return nil, err
	}
	program, err := s.template.program(source)
	if err != nil {
		return nil, err
	}
	value, err := program.Eval(&expr.Context{Context: s.ctx, Root: s.rootData, Variables: s.variables, Functions: s.functions, MaxSteps: s.template.options.MaxEvaluationSteps, SharedSteps: &s.counter.evaluationSteps})
	if err != nil {
		return nil, err
	}
	if err := s.ctx.Err(); err != nil {
		return nil, err
	}
	return value, nil
}

func (s *renderState) commandError(command Command, err error) error {
	return &Error{Command: command.Raw, Paragraph: command.Location.Paragraph, CommandIndex: command.Location.Ordinal, Start: command.Location.Start, End: command.Location.End, Err: fmt.Errorf("%w: %w", ErrCommandExecution, err)}
}

func isObjectResult(value any) bool {
	if value == nil {
		return false
	}
	if _, ok := value.(fmt.Stringer); ok {
		return false
	}
	v := reflect.ValueOf(value)
	for v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface {
		if v.IsNil() {
			return false
		}
		v = v.Elem()
	}
	switch v.Kind() {
	case reflect.Map, reflect.Slice, reflect.Array, reflect.Struct:
		return true
	default:
		return false
	}
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
		return nil, fmt.Errorf("#each expects an array or slice, got %T", value)
	}
}
