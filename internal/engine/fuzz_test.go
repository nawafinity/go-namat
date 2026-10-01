package engine

import (
	"context"
	"strings"
	"testing"
	"time"
)

func FuzzParseCommand(f *testing.F) {
	for _, seed := range []string{
		"name", "(", "#each values as item", "#each values", "/each extra",
		"#let name = value", "#let name", "@image logo", "@raw-xml value",
		strings.Repeat("!", 512) + "true", "",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, source string) {
		_, _ = parseCommand(source)
	})
}

func FuzzReadPackage(f *testing.F) {
	valid := testDOCX(f, map[string][]byte{"word/document.xml": []byte(wordDocument(`<w:p/>`))})
	f.Add([]byte("not a zip"))
	f.Add([]byte("PK\x03\x04truncated"))
	f.Add(valid)
	f.Add(valid[:len(valid)/2])
	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = readPackageWithLimits(data, 1<<20, 4<<20, 100)
	})
}

func FuzzRenderLimits(f *testing.F) {
	f.Add(int64(1), 1)
	f.Add(int64(4096), 2)
	f.Fuzz(func(t *testing.T, outputLimit int64, iterationLimit int) {
		if outputLimit < 0 {
			outputLimit = -outputLimit
		}
		outputLimit = outputLimit%4096 + 1
		if iterationLimit < 0 {
			iterationLimit = -iterationLimit
		}
		iterationLimit = iterationLimit%8 + 1
		document := wordDocument(`<w:p><w:r><w:t>[[#each items as item]]</w:t></w:r></w:p><w:p><w:r><w:t>[[item]]</w:t></w:r></w:p><w:p><w:r><w:t>[[/each]]</w:t></w:r></w:p>`)
		template := testDOCX(t, map[string][]byte{"word/document.xml": []byte(document)})
		_, _ = CreateReport(context.Background(), template, map[string]any{"items": []int{1, 2, 3}}, Options{
			MaxOutputBytes: outputLimit,
			MaxIterations:  iterationLimit,
			Timeout:        time.Second,
		})
	})
}

func FuzzRenderCommandText(f *testing.F) {
	f.Add("name")
	f.Add("name ?? 'x'")
	f.Add("record?.name")
	f.Fuzz(func(t *testing.T, expression string) {
		// Keep fuzz input in an attribute-free text node and escape XML-sensitive
		// characters so malformed OOXML does not obscure command parser failures.
		expression = escapeXML(expression)
		document := wordDocument(`<w:p><w:r><w:t>[[` + expression + `]]</w:t></w:r></w:p>`)
		template := testDOCX(t, map[string][]byte{"word/document.xml": []byte(document)})
		_, _ = CreateReport(context.Background(), template, map[string]any{"name": "x"}, Options{Timeout: time.Second})
	})
}
