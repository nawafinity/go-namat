package engine

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/xml"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

const complexTablesFixture = "testdata/compatibility/complex-tables.docx"

func TestPublicComplexTableFixture(t *testing.T) {
	template, err := os.ReadFile(complexTablesFixture)
	if err != nil {
		t.Fatal(err)
	}
	wantUntouched := zipPartForCompatibilityTest(t, template, "customXml/item1.xml")

	report, err := CreateReport(context.Background(), template, map[string]any{
		"title": "Compatibility matrix",
		"records": []map[string]any{
			{"name": "Alpha", "value": "First", "details": []string{"One", "Two"}},
			{"name": "Beta", "value": "Second", "details": []string{"Three"}},
		},
	}, Options{})
	if err != nil {
		t.Fatalf("render public compatibility fixture: %v", err)
	}

	document := zipPartForCompatibilityTest(t, report, "word/document.xml")
	for _, value := range []string{"Compatibility matrix", "Alpha", "Beta", "First", "Second", "One", "Two", "Three"} {
		if !bytes.Contains(document, []byte(value)) {
			t.Errorf("rendered document is missing %q", value)
		}
	}
	for _, command := range []string{"[[#each", "[[/each", "[[loop."} {
		if bytes.Contains(document, []byte(command)) {
			t.Errorf("rendered document retains command prefix %q", command)
		}
	}
	for _, property := range []string{"gridSpan", "vMerge", "TableGrid", "tblHeader"} {
		if !bytes.Contains(document, []byte(property)) {
			t.Errorf("rendered document lost table property %q", property)
		}
	}
	if got := bytes.Count(document, []byte("<w:tbl>")); got != 3 {
		t.Errorf("nested table count = %d, want 3 (one outer and one nested table per record)", got)
	}
	if got := zipPartForCompatibilityTest(t, report, "customXml/item1.xml"); !bytes.Equal(got, wantUntouched) {
		t.Fatal("untouched custom XML part changed")
	}
	validateCompatibilityPackageXML(t, report)
}

func TestLibreOfficeConvertsCompatibilityFixture(t *testing.T) {
	soffice := libreOfficeExecutable()
	if soffice == "" {
		if os.Getenv("NAMAT_REQUIRE_LIBREOFFICE") == "1" {
			t.Fatal("LibreOffice is required but neither soffice nor libreoffice is on PATH")
		}
		t.Skip("LibreOffice soffice executable is not installed")
	}
	template, err := os.ReadFile(complexTablesFixture)
	if err != nil {
		t.Fatal(err)
	}
	report, err := CreateReport(context.Background(), template, map[string]any{
		"title":   "LibreOffice compatibility",
		"records": []map[string]any{{"name": "Alpha", "value": "First", "details": []string{"One", "Two"}}},
	}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	directory := t.TempDir()
	input := filepath.Join(directory, "report.docx")
	if err := os.WriteFile(input, report, 0o600); err != nil {
		t.Fatal(err)
	}
	outputDirectory := filepath.Join(directory, "output")
	profileDirectory := filepath.Join(directory, "profile")
	if err := os.MkdirAll(outputDirectory, 0o700); err != nil {
		t.Fatal(err)
	}
	profileURI := "file://" + filepath.ToSlash(profileDirectory)
	if runtime.GOOS == "windows" {
		profileURI = "file:///" + filepath.ToSlash(profileDirectory)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, soffice,
		"--headless", "--nologo", "--nodefault", "--nolockcheck", "--nofirststartwizard",
		"-env:UserInstallation="+profileURI,
		"--convert-to", "pdf", "--outdir", outputDirectory, input,
	)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("LibreOffice conversion failed: %v\n%s", err, output)
	}
	if ctx.Err() != nil {
		t.Fatalf("LibreOffice conversion timed out: %v", ctx.Err())
	}
	info, err := os.Stat(filepath.Join(outputDirectory, "report.pdf"))
	if err != nil {
		t.Fatalf("LibreOffice did not create report.pdf: %v\n%s", err, output)
	}
	if info.Size() < 100 {
		t.Fatalf("LibreOffice created an unexpectedly small PDF (%d bytes)", info.Size())
	}
}

func TestHeaderFooterLoopsLinksAndEndnotes(t *testing.T) {
	template := testDOCX(t, map[string][]byte{
		"word/document.xml": []byte(wordDocument(`<w:p><w:r><w:t>Body</w:t></w:r></w:p>`)),
		"word/header1.xml":  []byte(`<?xml version="1.0"?><w:hdr xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:p><w:r><w:t>[[#each items as item]]</w:t></w:r></w:p><w:p><w:r><w:t>[[item]]</w:t></w:r></w:p><w:p><w:r><w:t>[[/each]]</w:t></w:r></w:p><w:p><w:r><w:t>[[@link link]]</w:t></w:r></w:p></w:hdr>`),
		"word/footer1.xml":  []byte(`<?xml version="1.0"?><w:ftr xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:p><w:r><w:t>[[#each items as item]]</w:t></w:r></w:p><w:p><w:r><w:t>[[loop.index]]:[[item]]</w:t></w:r></w:p><w:p><w:r><w:t>[[/each]]</w:t></w:r></w:p></w:ftr>`),
		"word/endnotes.xml": []byte(`<?xml version="1.0"?><w:endnotes xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:endnote w:id="1"><w:p><w:r><w:t>[[value]]</w:t></w:r></w:p><w:p><w:r><w:t>[[@link link]]</w:t></w:r></w:p></w:endnote></w:endnotes>`),
	})
	report, err := CreateReport(context.Background(), template, map[string]any{
		"items": []string{"Alpha", "Beta"},
		"value": "Endnote value",
		"link":  Link{URL: "https://example.test/evidence", Label: "Evidence"},
	}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	for _, partName := range []string{"word/header1.xml", "word/footer1.xml"} {
		part := zipPartForCompatibilityTest(t, report, partName)
		if !bytes.Contains(part, []byte("Alpha")) || !bytes.Contains(part, []byte("Beta")) || bytes.Contains(part, []byte("[[")) {
			t.Errorf("%s did not render its loop correctly: %s", partName, part)
		}
	}
	endnotes := zipPartForCompatibilityTest(t, report, "word/endnotes.xml")
	if !bytes.Contains(endnotes, []byte("Endnote value")) || !bytes.Contains(endnotes, []byte("Evidence")) {
		t.Fatalf("endnotes were not fully rendered: %s", endnotes)
	}
	for _, relsName := range []string{"word/_rels/header1.xml.rels", "word/_rels/endnotes.xml.rels"} {
		rels := zipPartForCompatibilityTest(t, report, relsName)
		if !bytes.Contains(rels, []byte("https://example.test/evidence")) {
			t.Errorf("%s is missing its part-local hyperlink", relsName)
		}
	}
}

func libreOfficeExecutable() string {
	for _, name := range []string{"soffice", "libreoffice"} {
		if path, err := exec.LookPath(name); err == nil {
			return path
		}
	}
	return ""
}

func zipPartForCompatibilityTest(t *testing.T, data []byte, name string) []byte {
	t.Helper()
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range reader.File {
		if filepath.ToSlash(file.Name) != name {
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

func validateCompatibilityPackageXML(t *testing.T, data []byte) {
	t.Helper()
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range reader.File {
		if !strings.HasSuffix(strings.ToLower(file.Name), ".xml") && !strings.HasSuffix(strings.ToLower(file.Name), ".rels") {
			continue
		}
		stream, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		decoder := xml.NewDecoder(stream)
		for {
			if _, err := decoder.Token(); err == io.EOF {
				break
			} else if err != nil {
				_ = stream.Close()
				t.Fatalf("part %s is not well-formed XML: %v", file.Name, err)
			}
		}
		if err := stream.Close(); err != nil {
			t.Fatal(err)
		}
	}
}
