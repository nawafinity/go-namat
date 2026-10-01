package engine

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/nawafinity/go-namat/internal/expr"
	"github.com/nawafinity/go-namat/internal/value"
)

func renderTestState(options Options, data any) *renderState {
	options = options.normalized()
	return &renderState{
		ctx:       context.Background(),
		template:  &Template{options: options},
		rootData:  data,
		variables: map[string]any{},
		declared:  map[string]struct{}{},
		functions: options.expressionFunctions(),
		counter:   &renderCounter{},
		pkg:       richTestState().pkg,
		partName:  "word/document.xml",
	}
}

func commandParagraph(t *testing.T, command string) *xmlNode {
	t.Helper()
	return mustParseElement(t, "<w:p><w:r><w:t>"+escapeXML(command)+"</w:t></w:r></w:p>")
}

func TestRenderSequenceCancellationAndStandaloneFailures(t *testing.T) {
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	state := renderTestState(Options{}, nil)
	state.ctx = canceled
	if _, err := state.processSequence([]*xmlNode{elementNode("p")}); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled sequence error = %v", err)
	}

	for _, test := range []struct {
		name      string
		paragraph *xmlNode
		configure func(*renderState)
	}{
		{"standalone parse error", commandParagraph(t, "[[#else extra]]"), nil},
		{"invalid expression", commandParagraph(t, "[[*missing]]"), nil},
		{"HTML wrong part", commandParagraph(t, "[[@html value]]"), func(s *renderState) { s.partName = "word/header1.xml" }},
		{"HTML evaluation", commandParagraph(t, "[[@html missing]]"), nil},
		{"RAW disabled", commandParagraph(t, "[[@raw-xml value]]"), nil},
		{"RAW evaluation", commandParagraph(t, "[[@raw-xml missing]]"), func(s *renderState) { s.template.options.AllowRawXML = true }},
		{"RAW malformed", commandParagraph(t, "[[@raw-xml value]]"), func(s *renderState) {
			s.template.options.AllowRawXML = true
			s.rootData = map[string]any{"value": "<broken>"}
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			state := renderTestState(Options{}, map[string]any{"value": "<p>ok</p>"})
			if test.configure != nil {
				test.configure(state)
			}
			if _, err := state.processSequence([]*xmlNode{test.paragraph.clone()}); err == nil {
				t.Fatal("expected sequence failure")
			}
		})
	}
}

func TestStructuralSequenceFailureBranches(t *testing.T) {
	paragraphs := func(values ...string) []*xmlNode {
		result := make([]*xmlNode, len(values))
		for index, value := range values {
			result[index] = commandParagraph(t, value)
		}
		return result
	}

	for _, test := range []struct {
		name      string
		children  []*xmlNode
		data      any
		max       int
		configure func(*renderState)
	}{
		{"structural parse", paragraphs("[[#if true]]", "[[#else extra]]", "[[/if]]"), nil, 10, nil},
		{"IF missing end", paragraphs("[[#if true]]"), nil, 10, nil},
		{"IF duplicate else", paragraphs("[[#if true]]", "[[#else]]", "[[#else]]", "[[/if]]"), nil, 10, nil},
		{"IF evaluation", paragraphs("[[#if missing]]", "[[/if]]"), nil, 10, nil},
		{"FOR evaluation", paragraphs("[[#each missing as item]]", "[[/each]]"), nil, 10, nil},
		{"FOR non-iterable", paragraphs("[[#each value as item]]", "[[/each]]"), map[string]any{"value": 1}, 10, nil},
		{"FOR limit", paragraphs("[[#each value as item]]", "[[/each]]"), map[string]any{"value": []int{1, 2}}, 1, nil},
		{"nested processing", paragraphs("[[#if true]]", "[[missing]]", "[[/if]]"), nil, 10, nil},
	} {
		t.Run(test.name, func(t *testing.T) {
			state := renderTestState(Options{MaxIterations: test.max}, test.data)
			if test.configure != nil {
				test.configure(state)
			}
			if _, err := state.processSequence(test.children); err == nil {
				t.Fatal("expected structural sequence error")
			}
		})
	}

	state := renderTestState(Options{}, nil)
	falseWithoutElse, err := state.processSequence(paragraphs("[[#if false]]", "hidden", "[[/if]]"))
	if err != nil || len(falseWithoutElse) != 0 {
		t.Fatalf("false IF = %d nodes, %v", len(falseWithoutElse), err)
	}
	falseWithElse, err := state.processSequence(paragraphs("[[#if false]]", "hidden", "[[#else]]", "visible", "[[/if]]"))
	if err != nil || len(falseWithElse) != 1 {
		t.Fatalf("false IF with ELSE = %d nodes, %v", len(falseWithElse), err)
	}

	row := mustParseElement(t, "<w:tr><w:tc><w:p><w:r><w:t>[[#if true]]</w:t></w:r></w:p><w:p><w:r><w:t>[[#each items as item]]</w:t></w:r></w:p></w:tc></w:tr>")
	if _, err := state.processSequence([]*xmlNode{row}); err == nil {
		t.Fatal("multiple structural commands in one row succeeded")
	}
	loopWithBadBody := paragraphs("[[#each items as item]]", "[[missing]]", "[[/each]]")
	state = renderTestState(Options{}, map[string]any{"items": []int{1}})
	if _, err := state.processSequence(loopWithBadBody); err == nil {
		t.Fatal("loop child failure was not propagated")
	}
}

func TestFindStructuralEndEdges(t *testing.T) {
	options := Options{}.normalized()
	children := []*xmlNode{
		commandParagraph(t, "[[#if true]]"),
		commandParagraph(t, "[[#if true]]"),
		commandParagraph(t, "[[/if]]"),
		commandParagraph(t, "[[/if]]"),
	}
	end, elseIndex, err := findStructuralEnd(children, 0, CommandIf, options)
	if err != nil || end != 3 || elseIndex != -1 {
		t.Fatalf("nested structural end = %d, %d, %v", end, elseIndex, err)
	}
	bad := []*xmlNode{commandParagraph(t, "[[#if true]]"), commandParagraph(t, "[[#else extra]]")}
	if _, _, err := findStructuralEnd(bad, 0, CommandIf, options); err == nil {
		t.Fatal("malformed structural command succeeded")
	}
}

func TestRenderParagraphFailureAndRecoveryEdges(t *testing.T) {
	wantErr := errors.New("handler failed")
	tests := []struct {
		name      string
		command   string
		data      any
		options   Options
		configure func(*renderState)
	}{
		{"span error", "[[#else extra]]", nil, Options{}, nil},
		{"invalid expression", "[[*missing]]", nil, Options{}, nil},
		{"assignment syntax", "[[#let invalid]]", nil, Options{}, nil},
		{"assignment evaluation", "[[#let value = missing]]", nil, Options{}, nil},
		{"insert evaluation", "[[missing]]", nil, Options{}, nil},
		{"handler failure", "[[missing]]", nil, Options{ErrorHandler: func(string, error) (any, error) { return nil, wantErr }}, nil},
		{"null rejected", "[[value]]", map[string]any{"value": nil}, Options{}, nil},
		{"null handler failure", "[[value]]", map[string]any{"value": nil}, Options{ErrorHandler: func(string, error) (any, error) { return nil, wantErr }}, nil},
		{"object rejected", "[[value]]", map[string]any{"value": map[string]any{}}, Options{}, nil},
		{"object handler failure", "[[value]]", map[string]any{"value": map[string]any{}}, Options{ErrorHandler: func(string, error) (any, error) { return nil, wantErr }}, nil},
		{"image evaluation", "[[@image missing]]", nil, Options{}, nil},
		{"image materialization", "[[@image value]]", map[string]any{"value": 1}, Options{}, nil},
		{"link evaluation", "[[@link missing]]", nil, Options{}, nil},
		{"link materialization", "[[@link value]]", map[string]any{"value": Link{URL: "relative"}}, Options{}, nil},
		{"inline HTML", "prefix [[@html value]]", map[string]any{"value": "x"}, Options{}, nil},
		{"inline structural", "prefix [[#if true]]", nil, Options{}, nil},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state := renderTestState(test.options, test.data)
			if test.configure != nil {
				test.configure(state)
			}
			if err := state.renderParagraph(commandParagraph(t, test.command)); err == nil {
				t.Fatal("expected paragraph render error")
			}
		})
	}

	for _, test := range []struct {
		name    string
		value   any
		options Options
		want    string
	}{
		{"evaluation recovery", nil, Options{ErrorHandler: func(string, error) (any, error) { return "fallback", nil }}, "fallback"},
		{"null recovery", nil, Options{ErrorHandler: func(string, error) (any, error) { return "null", nil }}, "null"},
		{"object recovery", map[string]any{}, Options{ErrorHandler: func(string, error) (any, error) { return "object", nil }}, "object"},
	} {
		t.Run(test.name, func(t *testing.T) {
			state := renderTestState(test.options, map[string]any{"value": test.value})
			command := "[[value]]"
			if test.name == "evaluation recovery" {
				command = "[[missing]]"
			}
			paragraph := commandParagraph(t, command)
			if err := state.renderParagraph(paragraph); err != nil || textOfParagraph(paragraph) != test.want {
				t.Fatalf("recovery text = %q, %v", textOfParagraph(paragraph), err)
			}
		})
	}

	state := renderTestState(Options{}, map[string]any{"value": 2})
	paragraph := commandParagraph(t, "[[#let variable = value]]")
	if _, err := state.processSequence([]*xmlNode{paragraph}); err != nil || state.variables["variable"] != 2 {
		t.Fatalf("SET result = %#v, %v", state.variables, err)
	}
	plain := commandParagraph(t, "plain")
	if err := state.renderParagraph(plain); err != nil {
		t.Fatalf("plain paragraph: %v", err)
	}
}

func TestEvaluateCancellationEdges(t *testing.T) {
	state := renderTestState(Options{}, nil)
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	state.ctx = canceled
	if _, err := state.evaluate("true"); !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-evaluation cancellation = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	state = renderTestState(Options{Functions: map[string]FunctionSpec{
		"cancel": {Returns: value.Bool, Call: func(context.Context, ...any) (any, error) {
			cancel()
			return true, nil
		}},
	}}, nil)
	state.ctx = ctx
	if _, err := state.evaluate("cancel()"); !errors.Is(err, context.Canceled) {
		t.Fatalf("post-evaluation cancellation = %v", err)
	}
	state = renderTestState(Options{}, nil)
	if _, err := state.evaluate("@"); err == nil {
		t.Fatal("invalid expression evaluated")
	}
}

func TestParagraphReplacementAndIterableEdges(t *testing.T) {
	if err := replaceParagraphText(elementNode("p"), nil); err == nil {
		t.Fatal("replacement without text nodes succeeded")
	}
	paragraph := mustParseElement(t, "<w:p><w:r><w:t>abc</w:t></w:r><w:r><w:t>def</w:t></w:r></w:p>")
	if err := replaceParagraphText(paragraph.clone(), []textReplacement{{start: 99, end: 99, value: "x"}}); err == nil {
		t.Fatal("missing replacement start succeeded")
	}
	if err := replaceParagraphText(paragraph.clone(), []textReplacement{{start: 1, end: 99, value: "x"}}); err == nil {
		t.Fatal("missing replacement end succeeded")
	}
	clone := paragraph.clone()
	if err := replaceParagraphText(clone, []textReplacement{{start: 1, end: 5, value: "X"}}); err != nil || textOfParagraph(clone) != "aXf" {
		t.Fatalf("multi-node replacement = %q, %v", textOfParagraph(clone), err)
	}
	three := mustParseElement(t, "<w:p><w:r><w:t>ab</w:t></w:r><w:r><w:t>cd</w:t></w:r><w:r><w:t>ef</w:t></w:r></w:p>")
	if err := replaceParagraphText(three, []textReplacement{{start: 1, end: 5, value: "X"}}); err != nil || textOfParagraph(three) != "aXf" {
		t.Fatalf("three-node replacement = %q, %v", textOfParagraph(three), err)
	}
	if _, _, ok := locateTextOffset(textNodes(paragraph), 99, false); ok {
		t.Fatal("out-of-range text offset succeeded")
	}

	values := [2]int{1, 2}
	var nilPointer *[]int
	for _, test := range []struct {
		value any
		len   int
		err   bool
	}{
		{nil, 0, false},
		{nilPointer, 0, false},
		{&values, 2, false},
		{[]string{"a"}, 1, false},
		{1, 0, true},
	} {
		got, err := iterable(test.value)
		if len(got) != test.len || (err != nil) != test.err {
			t.Fatalf("iterable(%T) = %#v, %v", test.value, got, err)
		}
	}

	var nilObject any
	pointerToNilObject := &nilObject
	if isObjectResult(pointerToNilObject) {
		t.Fatal("pointer to nil interface reported as object")
	}
}

func TestParagraphActionErrors(t *testing.T) {
	paragraph := commandParagraph(t, "text")
	if err := applyParagraphAction(paragraph.clone(), paragraphAction{start: 99, end: 99, node: elementNode("node")}); err == nil {
		t.Fatal("invalid node action succeeded")
	}
	value := "value"
	if err := applyParagraphAction(paragraph.clone(), paragraphAction{start: 99, end: 99, text: &value}); err == nil {
		t.Fatal("invalid text action succeeded")
	}
	if err := applyParagraphAction(paragraph.clone(), paragraphAction{}); err == nil {
		t.Fatal("empty paragraph action succeeded")
	}
	if err := applyParagraphActions(paragraph.clone(), []paragraphAction{{}}); err == nil {
		t.Fatal("paragraph action list did not propagate failure")
	}
	if err := finalizeParagraph(paragraph.clone(), []paragraphAction{{}}, Options{}.normalized()); err == nil {
		t.Fatal("paragraph finalization did not propagate action failure")
	}
}

func TestRenderProcessNodePropagatesChildError(t *testing.T) {
	state := renderTestState(Options{}, nil)
	bad := elementNode("root")
	bad.Children = append(bad.Children, commandParagraph(t, "[[missing]]"))
	if err := state.processNode(bad); err == nil {
		t.Fatal("child render error was not propagated")
	}

	state = renderTestState(Options{}, nil)
	state.functions["fail"] = expr.Function{Signature: expr.Signature{Returns: value.Any}, Call: func(context.Context, ...any) (any, error) { return nil, fmt.Errorf("failed") }}
	if _, err := state.evaluate("fail()"); err == nil {
		t.Fatal("host function error was not propagated")
	}
}
