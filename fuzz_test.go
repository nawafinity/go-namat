package namat

import (
	"context"
	"testing"
	"time"
)

func FuzzParseCommand(f *testing.F) {
	for _, seed := range []string{"INS name", "FOR x IN values", "END-FOR x", "ALIAS n INS name", "*n", "QUERY query { x }", ""} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, source string) {
		_, _ = parseCommand(source)
	})
}

func FuzzReadPackage(f *testing.F) {
	f.Add([]byte("not a zip"))
	f.Add(testDOCX(f, map[string][]byte{"word/document.xml": []byte(wordDocument(`<w:p/>`))}))
	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = readPackageWithLimits(data, 1<<20, 4<<20, 100)
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
