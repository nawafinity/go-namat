package tiraz

import (
	"archive/zip"
	"bytes"
	"context"
	"io"
	"sort"
	"strings"
	"testing"
)

const contentTypesXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="xml" ContentType="application/xml"/>
  <Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>
</Types>`

const relationshipsXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>
</Relationships>`

func TestRenderInsertAcrossWordRuns(t *testing.T) {
	document := wordDocument(`<w:p>
  <w:r><w:t xml:space="preserve">مرحباً </w:t></w:r>
  <w:r><w:t>[[INS agency.</w:t></w:r>
  <w:r><w:t>name]]</w:t></w:r>
  <w:r><w:t>!</w:t></w:r>
</w:p>`)
	template := testDOCX(t, map[string][]byte{
		"word/document.xml":       []byte(document),
		"word/media/original.bin": {0x00, 0x10, 0xfe, 0xff},
	})

	report, err := CreateReport(context.Background(), template, map[string]any{
		"agency": map[string]any{"name": "هيئة البيانات"},
	}, Options{})
	if err != nil {
		t.Fatalf("CreateReport returned an error: %v", err)
	}

	root := parseDocumentPart(t, report)
	if got, want := textOfParagraph(root.descendants("p")[0]), "مرحباً هيئة البيانات!"; got != want {
		t.Fatalf("paragraph text = %q, want %q", got, want)
	}
	if got := readPart(t, report, "word/media/original.bin"); !bytes.Equal(got, []byte{0x00, 0x10, 0xfe, 0xff}) {
		t.Fatalf("binary package part changed: %v", got)
	}
}

func TestInsertKeepsCommandRunAtTextBoundary(t *testing.T) {
	document := wordDocument(`<w:p>
  <w:r><w:rPr><w:color w:val="0000FF"/></w:rPr><w:t xml:space="preserve">قبل </w:t></w:r>
  <w:r><w:rPr><w:color w:val="FF0000"/></w:rPr><w:t>[[name]]</w:t></w:r>
</w:p>`)
	template := testDOCX(t, map[string][]byte{"word/document.xml": []byte(document)})

	report, err := CreateReport(context.Background(), template, map[string]any{"name": "القيمة"}, Options{})
	if err != nil {
		t.Fatalf("CreateReport returned an error: %v", err)
	}
	root := parseDocumentPart(t, report)
	runs := root.descendants("r")
	if got := textOfParagraph(runs[0]); got != "قبل " {
		t.Fatalf("first run changed to %q", got)
	}
	if got := textOfParagraph(runs[1]); got != "القيمة" {
		t.Fatalf("command run text = %q, want inserted value", got)
	}
}

func TestRenderCommandSplitAcrossParagraphs(t *testing.T) {
	document := wordDocument(`
<w:p><w:r><w:t>[[INS agency.</w:t></w:r></w:p>
<w:p><w:r><w:t>name]]</w:t></w:r></w:p>`)
	template := testDOCX(t, map[string][]byte{"word/document.xml": []byte(document)})

	report, err := CreateReport(context.Background(), template, map[string]any{
		"agency": map[string]any{"name": "هيئة البيانات"},
	}, Options{})
	if err != nil {
		t.Fatalf("CreateReport returned an error: %v", err)
	}

	text := documentText(t, report)
	if !strings.Contains(text, "هيئة البيانات") || strings.Contains(text, "[[") || strings.Contains(text, "]]") {
		t.Fatalf("unexpected rendered text: %q", text)
	}
}

func TestRenderConditionalWithElse(t *testing.T) {
	document := wordDocument(`
<w:p><w:r><w:t>[[IF agency.active]]</w:t></w:r></w:p>
<w:p><w:r><w:t>نشط: [[agency.name]]</w:t></w:r></w:p>
<w:p><w:r><w:t>[[ELSE]]</w:t></w:r></w:p>
<w:p><w:r><w:t>غير نشط</w:t></w:r></w:p>
<w:p><w:r><w:t>[[END-IF]]</w:t></w:r></w:p>`)
	template := testDOCX(t, map[string][]byte{"word/document.xml": []byte(document)})

	report, err := CreateReport(context.Background(), template, map[string]any{
		"agency": map[string]any{"active": true, "name": "ألف"},
	}, Options{})
	if err != nil {
		t.Fatalf("CreateReport returned an error: %v", err)
	}

	text := documentText(t, report)
	if !strings.Contains(text, "نشط: ألف") || strings.Contains(text, "غير نشط") || strings.Contains(text, "[[") {
		t.Fatalf("unexpected rendered text: %q", text)
	}
}

func TestRenderLoopRepeatsTableRows(t *testing.T) {
	document := wordDocument(`<w:tbl>
<w:tr><w:tc><w:p><w:r><w:t>[[FOR agency IN agencies]]</w:t></w:r></w:p></w:tc></w:tr>
<w:tr><w:tc><w:p><w:r><w:t>[[$idx + 1]] - [[$agency.name]]</w:t></w:r></w:p></w:tc></w:tr>
<w:tr><w:tc><w:p><w:r><w:t>[[END-FOR agency]]</w:t></w:r></w:p></w:tc></w:tr>
</w:tbl>`)
	template := testDOCX(t, map[string][]byte{"word/document.xml": []byte(document)})

	report, err := CreateReport(context.Background(), template, map[string]any{
		"agencies": []map[string]any{{"name": "الأولى"}, {"name": "الثانية"}},
	}, Options{})
	if err != nil {
		t.Fatalf("CreateReport returned an error: %v", err)
	}

	root := parseDocumentPart(t, report)
	rows := root.descendants("tr")
	if len(rows) != 2 {
		t.Fatalf("rendered row count = %d, want 2", len(rows))
	}
	if got := textOfParagraph(rows[0].descendants("p")[0]); got != "1 - الأولى" {
		t.Fatalf("first row = %q", got)
	}
	if got := textOfParagraph(rows[1].descendants("p")[0]); got != "2 - الثانية" {
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

func wordDocument(body string) string {
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
<w:body>` + body + `<w:sectPr/></w:body></w:document>`
}

func testDOCX(t *testing.T, parts map[string][]byte) []byte {
	t.Helper()
	all := map[string][]byte{
		"[Content_Types].xml": []byte(contentTypesXML),
		"_rels/.rels":         []byte(relationshipsXML),
	}
	for name, data := range parts {
		all[name] = data
	}
	var out bytes.Buffer
	writer := zip.NewWriter(&out)
	names := make([]string, 0, len(all))
	for name := range all {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		data := all[name]
		stream, err := writer.Create(name)
		if err != nil {
			t.Fatalf("create ZIP part %s: %v", name, err)
		}
		if _, err := stream.Write(data); err != nil {
			t.Fatalf("write ZIP part %s: %v", name, err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close test DOCX: %v", err)
	}
	return out.Bytes()
}

func readPart(t *testing.T, document []byte, name string) []byte {
	t.Helper()
	reader, err := zip.NewReader(bytes.NewReader(document), int64(len(document)))
	if err != nil {
		t.Fatalf("open rendered DOCX: %v", err)
	}
	for _, file := range reader.File {
		if file.Name != name {
			continue
		}
		stream, err := file.Open()
		if err != nil {
			t.Fatalf("open rendered part %s: %v", name, err)
		}
		data, readErr := io.ReadAll(stream)
		closeErr := stream.Close()
		if readErr != nil {
			t.Fatalf("read rendered part %s: %v", name, readErr)
		}
		if closeErr != nil {
			t.Fatalf("close rendered part %s: %v", name, closeErr)
		}
		return data
	}
	t.Fatalf("rendered part %s not found", name)
	return nil
}

func parseDocumentPart(t *testing.T, document []byte) *xmlNode {
	t.Helper()
	root, err := parseXML(readPart(t, document, "word/document.xml"))
	if err != nil {
		t.Fatalf("parse rendered document XML: %v", err)
	}
	return root
}

func documentText(t *testing.T, document []byte) string {
	t.Helper()
	root := parseDocumentPart(t, document)
	var out strings.Builder
	for _, paragraph := range root.descendants("p") {
		out.WriteString(textOfParagraph(paragraph))
		out.WriteByte('\n')
	}
	return out.String()
}
