package engine

import (
	"archive/zip"
	"bytes"
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

func wordDocument(body string) string {
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
<w:body>` + body + `<w:sectPr/></w:body></w:document>`
}

func testDOCX(t testing.TB, parts map[string][]byte) []byte {
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
		stream, err := writer.Create(name)
		if err != nil {
			t.Fatalf("create ZIP part %s: %v", name, err)
		}
		if _, err := stream.Write(all[name]); err != nil {
			t.Fatalf("write ZIP part %s: %v", name, err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close test DOCX: %v", err)
	}
	return out.Bytes()
}

func readPart(t testing.TB, document []byte, name string) []byte {
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

func parseDocumentPart(t testing.TB, document []byte) *xmlNode {
	t.Helper()
	root, err := parseXML(readPart(t, document, "word/document.xml"))
	if err != nil {
		t.Fatalf("parse rendered document XML: %v", err)
	}
	return root
}

func documentText(t testing.TB, document []byte) string {
	t.Helper()
	root := parseDocumentPart(t, document)
	var out strings.Builder
	for _, paragraph := range root.descendants("p") {
		out.WriteString(textOfParagraph(paragraph))
		out.WriteByte('\n')
	}
	return out.String()
}
