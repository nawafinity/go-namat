package main

import (
	"archive/zip"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestHelpAndVersion(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := run([]string{"help"}, &stdout, &stderr); code != 0 || !bytes.Contains(stdout.Bytes(), []byte("namat render")) {
		t.Fatalf("help code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := run([]string{"version"}, &stdout, &stderr); code != 0 || stdout.String() == "" {
		t.Fatalf("version code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
}

func TestAtomicWriteRefusesImplicitReplacement(t *testing.T) {
	directory := t.TempDir()
	name := filepath.Join(directory, "report.docx")
	if err := atomicWrite(name, []byte("first"), false); err != nil {
		t.Fatal(err)
	}
	if err := atomicWrite(name, []byte("second"), false); err == nil {
		t.Fatal("expected rename failure when destination exists")
	}
	content, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "first" {
		t.Fatalf("existing output changed to %q", content)
	}
	if err := atomicWrite(name, []byte("second"), true); err != nil {
		t.Fatal(err)
	}
	content, _ = os.ReadFile(name)
	if string(content) != "second" {
		t.Fatalf("forced output = %q", content)
	}
}

func TestUnknownCommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := run([]string{"unknown"}, &stdout, &stderr); code != 2 || !bytes.Contains(stderr.Bytes(), []byte("unknown command")) {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
}

func TestInspectMetadataAndRenderCommands(t *testing.T) {
	directory := t.TempDir()
	templatePath := filepath.Join(directory, "template.docx")
	dataPath := filepath.Join(directory, "data.json")
	outputPath := filepath.Join(directory, "output.docx")
	if err := os.WriteFile(templatePath, syntheticTemplate(t), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dataPath, []byte(`{"value":"synthetic"}`), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	if code := run([]string{"inspect", "--json", templatePath}, &stdout, &stderr); code != 0 || !bytes.Contains(stdout.Bytes(), []byte(`"commands":1`)) {
		t.Fatalf("inspect code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := run([]string{"metadata", templatePath}, &stdout, &stderr); code != 0 || !bytes.Contains(stdout.Bytes(), []byte(`"Title": ""`)) {
		t.Fatalf("metadata code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := run([]string{"render", "--data", dataPath, "--out", outputPath, templatePath}, &stdout, &stderr); code != 0 {
		t.Fatalf("render code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	report, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(readZIPPart(t, report, "word/document.xml"), []byte("synthetic")) {
		t.Fatal("rendered package does not contain the synthetic value")
	}
}

func syntheticTemplate(t *testing.T) []byte {
	t.Helper()
	parts := map[string]string{
		"[Content_Types].xml": `<?xml version="1.0"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/><Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/></Types>`,
		"_rels/.rels":         `<?xml version="1.0"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/></Relationships>`,
		"word/document.xml":   `<?xml version="1.0"?><w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body><w:p><w:r><w:t>[[value]]</w:t></w:r></w:p><w:sectPr/></w:body></w:document>`,
	}
	var output bytes.Buffer
	writer := zip.NewWriter(&output)
	for _, name := range []string{"[Content_Types].xml", "_rels/.rels", "word/document.xml"} {
		part, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := part.Write([]byte(parts[name])); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return output.Bytes()
}

func readZIPPart(t *testing.T, data []byte, name string) []byte {
	t.Helper()
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range reader.File {
		if file.Name != name {
			continue
		}
		stream, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		content, readErr := io.ReadAll(stream)
		closeErr := stream.Close()
		if readErr != nil {
			t.Fatal(readErr)
		}
		if closeErr != nil {
			t.Fatal(closeErr)
		}
		return content
	}
	t.Fatalf("ZIP part %q not found", name)
	return nil
}
