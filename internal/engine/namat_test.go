package engine

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestRenderInsertAcrossWordRuns(t *testing.T) {
	document := wordDocument(`<w:p>
  <w:r><w:t xml:space="preserve">Hello </w:t></w:r>
  <w:r><w:t>[[INS record.</w:t></w:r>
  <w:r><w:t>name]]</w:t></w:r>
  <w:r><w:t>!</w:t></w:r>
</w:p>`)
	template := testDOCX(t, map[string][]byte{
		"word/document.xml":       []byte(document),
		"word/media/original.bin": {0x00, 0x10, 0xfe, 0xff},
	})

	report, err := CreateReport(context.Background(), template, map[string]any{
		"record": map[string]any{"name": "Synthetic Value"},
	}, Options{})
	if err != nil {
		t.Fatalf("CreateReport returned an error: %v", err)
	}

	root := parseDocumentPart(t, report)
	if got, want := textOfParagraph(root.descendants("p")[0]), "Hello Synthetic Value!"; got != want {
		t.Fatalf("paragraph text = %q, want %q", got, want)
	}
	if got := readPart(t, report, "word/media/original.bin"); !bytes.Equal(got, []byte{0x00, 0x10, 0xfe, 0xff}) {
		t.Fatalf("binary package part changed: %v", got)
	}
}

func TestInsertKeepsCommandRunAtTextBoundary(t *testing.T) {
	document := wordDocument(`<w:p>
  <w:r><w:rPr><w:color w:val="0000FF"/></w:rPr><w:t xml:space="preserve">Before </w:t></w:r>
  <w:r><w:rPr><w:color w:val="FF0000"/></w:rPr><w:t>[[name]]</w:t></w:r>
</w:p>`)
	template := testDOCX(t, map[string][]byte{"word/document.xml": []byte(document)})

	report, err := CreateReport(context.Background(), template, map[string]any{"name": "value"}, Options{})
	if err != nil {
		t.Fatalf("CreateReport returned an error: %v", err)
	}
	root := parseDocumentPart(t, report)
	runs := root.descendants("r")
	if got := textOfParagraph(runs[0]); got != "Before " {
		t.Fatalf("first run changed to %q", got)
	}
	if got := textOfParagraph(runs[1]); got != "value" {
		t.Fatalf("command run text = %q, want inserted value", got)
	}
}

func TestRenderCommandSplitAcrossParagraphs(t *testing.T) {
	document := wordDocument(`
<w:p><w:r><w:t>[[INS record.</w:t></w:r></w:p>
<w:p><w:r><w:t>name]]</w:t></w:r></w:p>`)
	template := testDOCX(t, map[string][]byte{"word/document.xml": []byte(document)})

	report, err := CreateReport(context.Background(), template, map[string]any{
		"record": map[string]any{"name": "Synthetic Value"},
	}, Options{})
	if err != nil {
		t.Fatalf("CreateReport returned an error: %v", err)
	}

	text := documentText(t, report)
	if !strings.Contains(text, "Synthetic Value") || strings.Contains(text, "[[") || strings.Contains(text, "]]") {
		t.Fatalf("unexpected rendered text: %q", text)
	}
}

func TestRenderConditionalWithElse(t *testing.T) {
	document := wordDocument(`
<w:p><w:r><w:t>[[IF record.active]]</w:t></w:r></w:p>
<w:p><w:r><w:t>Available: [[record.name]]</w:t></w:r></w:p>
<w:p><w:r><w:t>[[ELSE]]</w:t></w:r></w:p>
<w:p><w:r><w:t>Unavailable</w:t></w:r></w:p>
<w:p><w:r><w:t>[[END-IF]]</w:t></w:r></w:p>`)
	template := testDOCX(t, map[string][]byte{"word/document.xml": []byte(document)})

	report, err := CreateReport(context.Background(), template, map[string]any{
		"record": map[string]any{"active": true, "name": "Alpha"},
	}, Options{})
	if err != nil {
		t.Fatalf("CreateReport returned an error: %v", err)
	}

	text := documentText(t, report)
	if !strings.Contains(text, "Available: Alpha") || strings.Contains(text, "Unavailable") || strings.Contains(text, "[[") {
		t.Fatalf("unexpected rendered text: %q", text)
	}
}

func TestRenderLoopRepeatsTableRows(t *testing.T) {
	document := wordDocument(`<w:tbl>
<w:tr><w:tc><w:p><w:r><w:t>[[FOR record IN records]]</w:t></w:r></w:p></w:tc></w:tr>
<w:tr><w:tc><w:p><w:r><w:t>[[$idx + 1]] - [[$record.name]]</w:t></w:r></w:p></w:tc></w:tr>
<w:tr><w:tc><w:p><w:r><w:t>[[END-FOR record]]</w:t></w:r></w:p></w:tc></w:tr>
</w:tbl>`)
	template := testDOCX(t, map[string][]byte{"word/document.xml": []byte(document)})

	report, err := CreateReport(context.Background(), template, map[string]any{
		"records": []map[string]any{{"name": "First"}, {"name": "Second"}},
	}, Options{})
	if err != nil {
		t.Fatalf("CreateReport returned an error: %v", err)
	}

	root := parseDocumentPart(t, report)
	rows := root.descendants("tr")
	if len(rows) != 2 {
		t.Fatalf("rendered row count = %d, want 2", len(rows))
	}
	if got := textOfParagraph(rows[0].descendants("p")[0]); got != "1 - First" {
		t.Fatalf("first row = %q", got)
	}
	if got := textOfParagraph(rows[1].descendants("p")[0]); got != "2 - Second" {
		t.Fatalf("second row = %q", got)
	}
}

func TestListCommandsIncludesHeaders(t *testing.T) {
	template := testDOCX(t, map[string][]byte{
		"word/document.xml": []byte(wordDocument(`<w:p><w:r><w:t>[[INS title]]</w:t></w:r></w:p>`)),
		"word/header1.xml":  []byte(`<?xml version="1.0"?><w:hdr xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:p><w:r><w:t>[[IF visible]]</w:t></w:r></w:p><w:p><w:r><w:t>[[END-IF]]</w:t></w:r></w:p></w:hdr>`),
	})

	commands, err := ListCommands(template, Options{})
	if err != nil {
		t.Fatalf("ListCommands returned an error: %v", err)
	}
	if len(commands) != 3 {
		t.Fatalf("command count = %d, want 3", len(commands))
	}
	if commands[0].Type != CommandInsert || commands[1].Type != CommandIf || commands[2].Type != CommandEndIf {
		t.Fatalf("unexpected command order: %#v", commands)
	}
}

func TestCompileRejectsUnbalancedStructure(t *testing.T) {
	template := testDOCX(t, map[string][]byte{
		"word/document.xml": []byte(wordDocument(`<w:p><w:r><w:t>[[IF active]]</w:t></w:r></w:p>`)),
	})

	_, err := Compile(template, Options{})
	if err == nil || !strings.Contains(err.Error(), "no matching end") {
		t.Fatalf("Compile error = %v, want unmatched structure error", err)
	}
}

func TestDOCMRoundTripPreservesMacroProject(t *testing.T) {
	macro := []byte{0xd0, 0xcf, 0x11, 0xe0, 0x00, 0x01, 0x02, 0x03}
	macroContentTypes := `<?xml version="1.0"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/><Default Extension="bin" ContentType="application/vnd.ms-office.vbaProject"/><Override PartName="/word/document.xml" ContentType="application/vnd.ms-word.document.macroEnabled.main+xml"/></Types>`
	template := testDOCX(t, map[string][]byte{
		"[Content_Types].xml": []byte(macroContentTypes),
		"word/document.xml":   []byte(wordDocument(`<w:p><w:r><w:t>[[name]]</w:t></w:r></w:p>`)),
		"word/vbaProject.bin": macro,
	})
	report, err := CreateReport(context.Background(), template, map[string]any{"name": "macro"}, Options{})
	if err != nil {
		t.Fatalf("CreateReport: %v", err)
	}
	if got := readPart(t, report, "word/vbaProject.bin"); !bytes.Equal(got, macro) {
		t.Fatalf("macro project changed: %v", got)
	}
	if got := string(readPart(t, report, "[Content_Types].xml")); !strings.Contains(got, "macroEnabled.main+xml") {
		t.Fatalf("macro content type changed: %s", got)
	}
}

func TestRejectsMismatchedEndForName(t *testing.T) {
	document := wordDocument(`<w:p><w:r><w:t>[[FOR item IN items]]</w:t></w:r></w:p><w:p><w:r><w:t>[[END-FOR other]]</w:t></w:r></w:p>`)
	_, err := Compile(testDOCX(t, map[string][]byte{"word/document.xml": []byte(document)}), Options{})
	if err == nil || !strings.Contains(err.Error(), "does not match") {
		t.Fatalf("error = %v, want mismatched loop variable", err)
	}
}
