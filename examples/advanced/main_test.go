package main

import (
	"archive/zip"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAdvancedExampleRendersNestedLoopsAndFunctionResults(t *testing.T) {
	output := filepath.Join(t.TempDir(), "advanced.docx")
	if err := run([]string{"--template", "template.docx", "--data", "data.json", "--out", output}); err != nil {
		t.Fatal(err)
	}
	report, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	document := advancedPart(t, report, "word/document.xml")
	for _, expected := range []string{
		"\u0627\u0644\u062a\u062d\u0648\u0644 \u0627\u0644\u0631\u0642\u0645\u064a",
		"\u0645\u0646\u0635\u0629 \u0627\u0644\u062a\u0642\u0627\u0631\u064a\u0631",
		"275000.50 SAR",
		"\u2713 \u0645\u0643\u062a\u0645\u0644",
		"146.5 \u0633\u0627\u0639\u0629",
		"30 \u0633\u0628\u062a\u0645\u0628\u0631 2026",
	} {
		if !bytes.Contains(document, []byte(expected)) {
			t.Errorf("rendered document is missing %q", expected)
		}
	}
	if bytes.Contains(document, []byte("[[")) {
		t.Fatal("rendered document still contains template commands")
	}
	if tables := bytes.Count(document, []byte("<w:tbl>")); tables != 5 {
		t.Fatalf("rendered table count = %d, want 5 (2 outer and 3 nested)", tables)
	}
	if links := bytes.Count(document, []byte("<w:hyperlink")); links != 3 {
		t.Fatalf("rendered hyperlink count = %d, want 3", links)
	}
	media := 0
	reader, err := zip.NewReader(bytes.NewReader(report), int64(len(report)))
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range reader.File {
		if strings.HasPrefix(file.Name, "word/media/") {
			media++
		}
	}
	if media != 3 {
		t.Fatalf("generated chart count = %d, want 3", media)
	}
}

func TestAdvancedExampleRepeatBuildsMultiPageStressData(t *testing.T) {
	output := filepath.Join(t.TempDir(), "stress.docx")
	if err := run([]string{"--template", "template.docx", "--data", "data.json", "--repeat", "3", "--out", output}); err != nil {
		t.Fatal(err)
	}
	report, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	document := advancedPart(t, report, "word/document.xml")
	if tables := bytes.Count(document, []byte("<w:tbl>")); tables != 15 {
		t.Fatalf("rendered table count = %d, want 15", tables)
	}
	if links := bytes.Count(document, []byte("<w:hyperlink")); links != 9 {
		t.Fatalf("rendered hyperlink count = %d, want 9", links)
	}
	if !bytes.Contains(document, []byte("#03")) {
		t.Fatal("rendered document is missing the third repeated batch")
	}
}

func TestAdvancedExampleRejectsInvalidRepeat(t *testing.T) {
	for _, repeat := range []string{"0", "51"} {
		if err := run([]string{"--repeat", repeat}); err == nil {
			t.Fatalf("run --repeat %s succeeded, want error", repeat)
		}
	}
}

func advancedPart(t *testing.T, data []byte, name string) []byte {
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
