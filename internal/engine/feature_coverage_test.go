package engine

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
)

func TestCommandSyntaxVariantsAndAssignments(t *testing.T) {
	document := wordDocument(`
<w:p><w:r><w:t>[[name]]|[[name]]|[[name]]</w:t></w:r></w:p>
<w:p><w:r><w:t>[[#let subtotal = 2 + 3]]</w:t></w:r></w:p>
<w:p><w:r><w:t>[[#let total = subtotal * 2]]</w:t></w:r></w:p>
<w:p><w:r><w:t>[[total]]</w:t></w:r></w:p>`)
	report, err := CreateReport(context.Background(), testDOCX(t, map[string][]byte{
		"word/document.xml": []byte(document),
	}), map[string]any{"name": "value"}, Options{})
	if err != nil {
		t.Fatalf("CreateReport: %v", err)
	}
	text := documentText(t, report)
	if !strings.Contains(text, "value|value|value") || !strings.Contains(text, "10") {
		t.Fatalf("command variants produced unexpected text: %q", text)
	}
}

func TestStrictObjectResultsAndLineBreakOptions(t *testing.T) {
	template := testDOCX(t, map[string][]byte{
		"word/document.xml": []byte(wordDocument(`<w:p><w:r><w:t>[[value]]</w:t></w:r></w:p>`)),
	})
	if _, err := CreateReport(context.Background(), template, map[string]any{
		"value": map[string]int{"x": 1},
	}, Options{}); err == nil || !errors.Is(err, ErrObjectResult) {
		t.Fatalf("object insertion error = %v", err)
	}

	report, err := CreateReport(context.Background(), template, map[string]any{
		"value": "first\nsecond",
	}, Options{DisableLineBreaks: true})
	if err != nil {
		t.Fatalf("DisableLineBreaks render: %v", err)
	}
	documentXML := string(readPart(t, report, "word/document.xml"))
	if strings.Contains(documentXML, "<w:br") || !strings.Contains(documentText(t, report), "first\nsecond") {
		t.Fatalf("line-break option was not respected: %s", documentXML)
	}
}

func TestReservedWordsRemainOrdinaryDataNames(t *testing.T) {
	template := testDOCX(t, map[string][]byte{"word/document.xml": []byte(wordDocument(
		`<w:p><w:r><w:t>[[image]]|[[link]]|[[html]]|[[query]]|[[!active]]</w:t></w:r></w:p>`,
	))})
	report, err := CreateReport(context.Background(), template, map[string]any{
		"image": "i", "link": "l", "html": "h", "query": "q", "active": false,
	}, Options{})
	if err != nil {
		t.Fatalf("render reserved data fields: %v", err)
	}
	if text := documentText(t, report); !strings.Contains(text, "i|l|h|q|true") {
		t.Fatalf("reserved field output = %q", text)
	}
}

func TestScannerHandlesClosingDelimiterInsideStringsAndIndexes(t *testing.T) {
	template := testDOCX(t, map[string][]byte{"word/document.xml": []byte(wordDocument(
		`<w:p><w:r><w:t>[[ 'inside ]] text' ]]|[[items[0]]]</w:t></w:r></w:p>`,
	))})
	report, err := CreateReport(context.Background(), template, map[string]any{"items": []string{"first"}}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if text := documentText(t, report); !strings.Contains(text, "inside ]] text|first") {
		t.Fatalf("scanner output = %q", text)
	}
}

func TestPartScopesAreIndependent(t *testing.T) {
	template := testDOCX(t, map[string][]byte{
		"word/document.xml": []byte(wordDocument(`<w:p><w:r><w:t>[[#let local = 'document']]</w:t></w:r></w:p><w:p><w:r><w:t>[[local]]</w:t></w:r></w:p>`)),
		"word/header1.xml":  []byte(`<?xml version="1.0"?><w:hdr xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:p><w:r><w:t>[[local]]</w:t></w:r></w:p></w:hdr>`),
	})
	if _, err := CreateReport(context.Background(), template, nil, Options{}); err == nil || !strings.Contains(err.Error(), "local") {
		t.Fatalf("part-local value leaked: %v", err)
	}
}

func TestLoopParentAndAggregateEvaluationBudget(t *testing.T) {
	document := wordDocument(`<w:p><w:r><w:t>[[#each groups as group]]</w:t></w:r></w:p><w:p><w:r><w:t>[[#each group as item]]</w:t></w:r></w:p><w:p><w:r><w:t>[[loop.parent.number]].[[loop.number]]</w:t></w:r></w:p><w:p><w:r><w:t>[[/each]]</w:t></w:r></w:p><w:p><w:r><w:t>[[/each]]</w:t></w:r></w:p>`)
	template := testDOCX(t, map[string][]byte{"word/document.xml": []byte(document)})
	report, err := CreateReport(context.Background(), template, map[string]any{"groups": [][]int{{1, 2}, {3}}}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	text := documentText(t, report)
	for _, want := range []string{"1.1", "1.2", "2.1"} {
		if !strings.Contains(text, want) {
			t.Fatalf("loop parent output %q is missing from %q", want, text)
		}
	}

	budgetTemplate := testDOCX(t, map[string][]byte{"word/document.xml": []byte(wordDocument(`<w:p><w:r><w:t>[[1]] [[2]]</w:t></w:r></w:p>`))})
	if _, err := CreateReport(context.Background(), budgetTemplate, nil, Options{MaxEvaluationSteps: 1}); err == nil || !strings.Contains(err.Error(), "step limit") {
		t.Fatalf("aggregate evaluation limit error = %v", err)
	}
}

func TestImageFormatsAndValidation(t *testing.T) {
	template := testDOCX(t, map[string][]byte{
		"word/document.xml": []byte(wordDocument(`<w:p><w:r><w:t>[[@image jpeg]]</w:t></w:r></w:p><w:p><w:r><w:t>[[@image gif]]</w:t></w:r></w:p>`)),
	})
	report, err := CreateReport(context.Background(), template, map[string]any{
		"jpeg": &Image{Data: []byte("synthetic-jpeg"), Extension: "jpeg", Width: 1, Height: 1},
		"gif":  Image{Data: []byte("synthetic-gif"), Extension: "gif", Width: 1, Height: 1},
	}, Options{})
	if err != nil {
		t.Fatalf("render JPEG and GIF: %v", err)
	}
	if !bytes.Contains(readPart(t, report, "[Content_Types].xml"), []byte("image/jpeg")) ||
		!bytes.Contains(readPart(t, report, "[Content_Types].xml"), []byte("image/gif")) {
		t.Fatal("JPEG or GIF content type is missing")
	}

	invalidCases := []struct {
		name  string
		value any
	}{
		{name: "empty data", value: Image{Extension: "png", Width: 1, Height: 1}},
		{name: "unsupported extension", value: Image{Data: []byte("x"), Extension: "bmp", Width: 1, Height: 1}},
		{name: "invalid dimensions", value: Image{Data: []byte("x"), Extension: "png", Width: 0, Height: 1}},
		{name: "invalid base64", value: map[string]any{"data": "%%%", "extension": "png", "width": 1, "height": 1}},
		{name: "invalid SVG fallback", value: Image{Data: []byte("<svg/>"), Extension: "svg", Width: 1, Height: 1, Thumbnail: &Image{Data: []byte("<svg/>"), Extension: "svg"}}},
	}
	single := testDOCX(t, map[string][]byte{
		"word/document.xml": []byte(wordDocument(`<w:p><w:r><w:t>[[@image value]]</w:t></w:r></w:p>`)),
	})
	for _, test := range invalidCases {
		t.Run(test.name, func(t *testing.T) {
			if _, err := CreateReport(context.Background(), single, map[string]any{"value": test.value}, Options{}); err == nil {
				t.Fatal("expected image validation error")
			}
		})
	}
}

func TestLinkSchemesAndPointerValue(t *testing.T) {
	template := testDOCX(t, map[string][]byte{
		"word/document.xml": []byte(wordDocument(`<w:p><w:r><w:t>[[@link link]]</w:t></w:r></w:p>`)),
	})
	report, err := CreateReport(context.Background(), template, map[string]any{
		"link": &Link{URL: "custom://resource/1", Tooltip: "Synthetic tooltip"},
	}, Options{AllowedLinkSchemes: []string{"CUSTOM"}})
	if err != nil {
		t.Fatalf("custom link scheme: %v", err)
	}
	documentXML := string(readPart(t, report, "word/document.xml"))
	relationships := string(readPart(t, report, "word/_rels/document.xml.rels"))
	if !strings.Contains(documentXML, "Synthetic tooltip") || !strings.Contains(relationships, "custom://resource/1") {
		t.Fatalf("pointer link was not rendered: document=%s relationships=%s", documentXML, relationships)
	}
}

func TestHTMLAndRawXMLPlacementRules(t *testing.T) {
	headerTemplate := testDOCX(t, map[string][]byte{
		"word/document.xml": []byte(wordDocument(`<w:p/>`)),
		"word/header1.xml":  []byte(`<?xml version="1.0"?><w:hdr xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:p><w:r><w:t>[[@html html]]</w:t></w:r></w:p></w:hdr>`),
	})
	if _, err := CreateReport(context.Background(), headerTemplate, map[string]any{"html": "<p>test</p>"}, Options{}); err == nil || !strings.Contains(err.Error(), "only in word/document.xml") {
		t.Fatalf("header HTML error = %v", err)
	}

	inlineRawXML := testDOCX(t, map[string][]byte{
		"word/document.xml": []byte(wordDocument(`<w:p><w:r><w:t>prefix [[@raw-xml xml]] suffix</w:t></w:r></w:p>`)),
	})
	if _, err := CreateReport(context.Background(), inlineRawXML, map[string]any{"xml": "<w:tab/>"}, Options{AllowRawXML: true}); err == nil || !strings.Contains(err.Error(), "must occupy its own paragraph") {
		t.Fatalf("inline RAW-XML error = %v", err)
	}
}

func TestAllWordTextPartsAreRendered(t *testing.T) {
	template := testDOCX(t, map[string][]byte{
		"word/document.xml":  []byte(wordDocument(`<w:p><w:r><w:t>[[value]]</w:t></w:r></w:p>`)),
		"word/footer1.xml":   []byte(`<?xml version="1.0"?><w:ftr xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:p><w:r><w:t>[[value]]</w:t></w:r></w:p></w:ftr>`),
		"word/footnotes.xml": []byte(`<?xml version="1.0"?><w:footnotes xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:footnote w:id="1"><w:p><w:r><w:t>[[value]]</w:t></w:r></w:p></w:footnote></w:footnotes>`),
	})
	report, err := CreateReport(context.Background(), template, map[string]any{"value": "rendered"}, Options{})
	if err != nil {
		t.Fatalf("CreateReport: %v", err)
	}
	for _, part := range []string{"word/document.xml", "word/footer1.xml", "word/footnotes.xml"} {
		if !bytes.Contains(readPart(t, report, part), []byte("rendered")) {
			t.Errorf("part %s was not rendered", part)
		}
	}
}

func TestPublicAPIGuardsAndDefaults(t *testing.T) {
	if _, err := CompileReader(nil, Options{}); err == nil {
		t.Fatal("CompileReader accepted a nil reader")
	}
	if _, err := Compile([]byte("not a DOCX"), Options{}); !errors.Is(err, ErrInvalidTemplate) {
		t.Fatalf("invalid package error = %v", err)
	}
	template := testDOCX(t, map[string][]byte{
		"word/document.xml": []byte(wordDocument(`<w:p><w:r><w:t>[[value]]</w:t></w:r></w:p>`)),
	})
	if _, err := Compile(template, Options{OpenDelimiter: "%%", CloseDelimiter: "%%"}); err == nil {
		t.Fatal("Compile accepted equal delimiters")
	}
	if _, err := Compile(template, Options{LanguageVersion: "legacy"}); err == nil || !strings.Contains(err.Error(), "only v1") {
		t.Fatalf("legacy language version error = %v", err)
	}
	compiled, err := Compile(template, Options{CompressionLevel: 99})
	if err != nil {
		t.Fatal(err)
	}
	if err := compiled.RenderTo(context.Background(), nil, map[string]any{"value": "x"}); err == nil {
		t.Fatal("RenderTo accepted a nil writer")
	}
	if _, err := compiled.Render(nil, map[string]any{"value": "x"}); err != nil {
		t.Fatalf("Render rejected a nil context: %v", err)
	}
	var nilTemplate *Template
	if commands := nilTemplate.Commands(); commands != nil {
		t.Fatalf("nil template Commands = %#v", commands)
	}
}

func TestErrorHandlerCanAbortRendering(t *testing.T) {
	sentinel := errors.New("handler rejected recovery")
	template := testDOCX(t, map[string][]byte{
		"word/document.xml": []byte(wordDocument(`<w:p><w:r><w:t>[[missing]]</w:t></w:r></w:p>`)),
	})
	_, err := CreateReport(context.Background(), template, nil, Options{ErrorHandler: func(string, error) (any, error) {
		return nil, sentinel
	}})
	if !errors.Is(err, sentinel) || !errors.Is(err, ErrCommandExecution) {
		t.Fatalf("handler error = %v", err)
	}
}

func TestMultilingualUnicodeRendering(t *testing.T) {
	values := []string{
		"English",
		"\u0645\u0631\u062d\u0628\u0627",
		"\u65e5\u672c\u8a9e",
		"\U0001f30d",
	}
	template := testDOCX(t, map[string][]byte{
		"word/document.xml": []byte(wordDocument(`<w:p><w:r><w:t>[[value]]</w:t></w:r></w:p>`)),
	})
	for _, value := range values {
		report, err := CreateReport(context.Background(), template, map[string]any{"value": value}, Options{})
		if err != nil {
			t.Fatalf("render %q: %v", value, err)
		}
		if got := documentText(t, report); !strings.Contains(got, value) {
			t.Fatalf("Unicode value changed: got %q, want %q", got, value)
		}
	}
}

func TestRTLFormattingPropertiesArePreserved(t *testing.T) {
	const rtlText = "\u0645\u0631\u062d\u0628\u0627"
	document := wordDocument(`<w:p><w:pPr><w:bidi/></w:pPr><w:r><w:rPr><w:rtl/></w:rPr><w:t>[[value]]</w:t></w:r></w:p>`)
	report, err := CreateReport(context.Background(), testDOCX(t, map[string][]byte{
		"word/document.xml": []byte(document),
	}), map[string]any{"value": rtlText}, Options{})
	if err != nil {
		t.Fatalf("CreateReport: %v", err)
	}
	documentXML := string(readPart(t, report, "word/document.xml"))
	if !strings.Contains(documentXML, "<w:bidi") ||
		!strings.Contains(documentXML, "<w:rtl") ||
		!strings.Contains(documentXML, rtlText) {
		t.Fatalf("RTL formatting or text was not preserved: %s", documentXML)
	}
}
