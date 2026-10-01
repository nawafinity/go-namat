package engine

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"os"
	"strings"
	"testing"
)

func TestPostRenderAndGeneratedPartLimits(t *testing.T) {
	template := testDOCX(t, map[string][]byte{
		"word/document.xml": []byte(wordDocument(`<w:p><w:r><w:t>[[value]]</w:t></w:r></w:p>`)),
	})
	compiled, err := Compile(template, Options{})
	if err != nil {
		t.Fatal(err)
	}
	compiled.options.MaxPackageParts = len(compiled.pkg.Parts) - 1
	if _, err := compiled.Render(context.Background(), map[string]any{"value": "x"}); !errors.Is(err, ErrSecurityLimit) {
		t.Fatalf("post-render package limit error = %v", err)
	}

	state := &renderState{template: compiled, pkg: compiled.pkg.clone()}
	compiled.options.MaxPackageParts = 100
	compiled.options.MaxPartBytes = 1
	if err := state.addGeneratedPart("word/too-large.bin", []byte("xx"), zip.Store); !errors.Is(err, ErrSecurityLimit) {
		t.Fatalf("generated part size error = %v", err)
	}
	compiled.options.MaxPartBytes = 1 << 20
	compiled.options.MaxUncompressedBytes = state.pkg.uncompressedBytes() + 1
	if err := state.addGeneratedPart("word/too-much.bin", []byte("xx"), zip.Store); !errors.Is(err, ErrSecurityLimit) {
		t.Fatalf("generated aggregate size error = %v", err)
	}
}

func TestHTMLGeneratedPartSizeLimit(t *testing.T) {
	template := testDOCX(t, map[string][]byte{
		"word/document.xml": []byte(wordDocument(`<w:p><w:r><w:t>[[@html html]]</w:t></w:r></w:p>`)),
	})
	compiled, err := Compile(template, Options{})
	if err != nil {
		t.Fatal(err)
	}
	compiled.options.MaxPartBytes = int64(len(compiled.pkg.Parts["word/document.xml"].Data) + 1)
	_, err = compiled.Render(context.Background(), map[string]any{"html": strings.Repeat("x", int(compiled.options.MaxPartBytes)+1)})
	if !errors.Is(err, ErrSecurityLimit) {
		t.Fatalf("HTML generated part error = %v", err)
	}
}

func TestPackageLimitValidationBranches(t *testing.T) {
	pkg := &docxPackage{Parts: map[string]*packagePart{
		"a": {Data: []byte("aa")},
		"b": {Data: []byte("bb")},
	}}
	for _, test := range []struct {
		name              string
		maxPart, maxTotal int64
		maxParts          int
	}{
		{name: "parts", maxPart: 10, maxTotal: 10, maxParts: 1},
		{name: "part size", maxPart: 1, maxTotal: 10, maxParts: 10},
		{name: "aggregate", maxPart: 10, maxTotal: 3, maxParts: 10},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := pkg.validateLimits(test.maxPart, test.maxTotal, test.maxParts); !errors.Is(err, ErrSecurityLimit) {
				t.Fatalf("validateLimits error = %v", err)
			}
		})
	}
}

func TestBoundedPackageWriterCancellationAndExhaustedLimit(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	writer := &boundedPackageWriter{ctx: ctx, destination: &bytes.Buffer{}, limit: 10}
	if written, err := writer.Write([]byte("x")); written != 0 || !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled write = %d, %v", written, err)
	}

	destination := bytes.NewBufferString("x")
	writer = &boundedPackageWriter{ctx: context.Background(), destination: destination, limit: 1}
	if written, err := writer.Write([]byte("y")); written != 0 || !errors.Is(err, ErrSecurityLimit) {
		t.Fatalf("exhausted write = %d, %v", written, err)
	}
}

func TestListCommandsSkipsCommandlessWordParts(t *testing.T) {
	template, err := os.ReadFile(complexTablesFixture)
	if err != nil {
		t.Fatal(err)
	}
	commands, err := ListCommands(template, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(commands) == 0 {
		t.Fatal("fixture commands were not discovered")
	}
}

func TestXMLWriterPreservesDocumentWhitespace(t *testing.T) {
	root, err := parseXML([]byte("<?xml version=\"1.0\"?>\n<w:document xmlns:w=\"urn:test\"><w:p>line\nnext &amp; safe</w:p></w:document>"))
	if err != nil {
		t.Fatal(err)
	}
	serialized := string(root.bytes())
	if strings.Contains(serialized, "?>&#xA;") || !strings.Contains(serialized, "?>\n<w:document") {
		t.Fatalf("document whitespace was serialized as a character reference: %q", serialized)
	}
	if !strings.Contains(serialized, "line\nnext &amp; safe") {
		t.Fatalf("element text was not escaped while preserving its newline: %q", serialized)
	}
}
