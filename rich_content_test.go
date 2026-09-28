package namat

import (
	"context"
	"encoding/base64"
	"strings"
	"testing"
)

func TestRenderInlineImageAddsMediaRelationshipAndContentType(t *testing.T) {
	document := wordDocument(`<w:p><w:r><w:t xml:space="preserve">قبل </w:t></w:r><w:r><w:t>[[IMAGE logo()]]</w:t></w:r><w:r><w:t xml:space="preserve"> بعد</w:t></w:r></w:p>`)
	template := testDOCX(t, map[string][]byte{"word/document.xml": []byte(document)})
	png, err := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII=")
	if err != nil {
		t.Fatal(err)
	}
	report, err := CreateReport(context.Background(), template, nil, Options{
		Functions: map[string]Function{
			"logo": func(args ...any) (any, error) {
				return Image{Data: png, Extension: ".png", Width: 1.5, Height: 1.5, Alt: "شعار"}, nil
			},
		},
	})
	if err != nil {
		t.Fatalf("CreateReport: %v", err)
	}
	pkg, err := readPackage(report)
	if err != nil {
		t.Fatalf("read rendered package: %v", err)
	}
	if _, ok := pkg.Parts["word/media/namat-image-1.png"]; !ok {
		t.Fatal("rendered package does not contain image media")
	}
	documentXML := string(pkg.Parts["word/document.xml"].Data)
	if !strings.Contains(documentXML, "<w:drawing") || !strings.Contains(documentXML, `r:embed="rIdNamat1"`) {
		t.Fatalf("drawing markup is incomplete: %s", documentXML)
	}
	rels := string(pkg.Parts["word/_rels/document.xml.rels"].Data)
	if !strings.Contains(rels, `Target="media/namat-image-1.png"`) || !strings.Contains(rels, `/image"`) {
		t.Fatalf("image relationship is incomplete: %s", rels)
	}
	contentTypes := string(pkg.Parts["[Content_Types].xml"].Data)
	if !strings.Contains(contentTypes, `Extension="png"`) || !strings.Contains(contentTypes, `ContentType="image/png"`) {
		t.Fatalf("PNG content type is missing: %s", contentTypes)
	}
	if got := documentText(t, report); !strings.Contains(got, "قبل  بعد") || strings.Contains(got, "IMAGE") {
		t.Fatalf("unexpected visible text: %q", got)
	}
}

func TestRenderSVGWithFallbackThumbnail(t *testing.T) {
	png, _ := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII=")
	template := testDOCX(t, map[string][]byte{"word/document.xml": []byte(wordDocument(`<w:p><w:r><w:t>[[IMAGE graphic()]]</w:t></w:r></w:p>`))})
	report, err := CreateReport(context.Background(), template, nil, Options{Functions: map[string]Function{
		"graphic": func(args ...any) (any, error) {
			return Image{Data: []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="10" height="10"><rect width="10" height="10"/></svg>`), Extension: "svg", Width: 2, Height: 2, Thumbnail: &Image{Data: png, Extension: "png"}}, nil
		},
	}})
	if err != nil {
		t.Fatalf("CreateReport: %v", err)
	}
	pkg, err := readPackage(report)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := pkg.Parts["word/media/namat-image-1.svg"]; !ok {
		t.Fatal("SVG media part missing")
	}
	if _, ok := pkg.Parts["word/media/namat-image-1.png"]; !ok {
		t.Fatal("fallback PNG media part missing")
	}
	if documentXML := string(pkg.Parts["word/document.xml"].Data); !strings.Contains(documentXML, "asvg:svgBlip") {
		t.Fatalf("SVG extension markup missing: %s", documentXML)
	}
}

func TestRenderImageFromSyntheticObject(t *testing.T) {
	png := "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII="
	template := testDOCX(t, map[string][]byte{
		"word/document.xml": []byte(wordDocument(`<w:p><w:r><w:t>[[IMAGE image]]</w:t></w:r></w:p>`)),
	})
	report, err := CreateReport(context.Background(), template, map[string]any{
		"image": map[string]any{
			"data": png, "extension": "png", "width": 2, "height": 1,
			"alt": "Synthetic image", "rotation": 15, "caption": "Generated fixture",
		},
	}, Options{})
	if err != nil {
		t.Fatalf("CreateReport: %v", err)
	}
	documentXML := string(readPart(t, report, "word/document.xml"))
	if !strings.Contains(documentXML, `descr="Synthetic image"`) ||
		!strings.Contains(documentXML, `rot="900000"`) ||
		!strings.Contains(documentXML, "Generated fixture") {
		t.Fatalf("image properties are incomplete: %s", documentXML)
	}
}

func TestImageInHeaderUsesHeaderRelationshipPart(t *testing.T) {
	png, _ := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII=")
	template := testDOCX(t, map[string][]byte{
		"word/document.xml": []byte(wordDocument(`<w:p/>`)),
		"word/header1.xml":  []byte(`<?xml version="1.0"?><w:hdr xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:p><w:r><w:t>[[IMAGE logo()]]</w:t></w:r></w:p></w:hdr>`),
	})
	report, err := CreateReport(context.Background(), template, nil, Options{Functions: map[string]Function{
		"logo": func(args ...any) (any, error) { return Image{Data: png, Extension: "png", Width: 1, Height: 1}, nil },
	}})
	if err != nil {
		t.Fatalf("CreateReport: %v", err)
	}
	pkg, err := readPackage(report)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := pkg.Parts["word/_rels/header1.xml.rels"]; !ok {
		t.Fatal("header relationship part is missing")
	}
	if !strings.Contains(string(pkg.Parts["word/header1.xml"].Data), "<w:drawing") {
		t.Fatal("header drawing is missing")
	}
}

func TestImageDrawingIdentifiersDoNotCollide(t *testing.T) {
	png, _ := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII=")
	document := wordDocument(`<w:p><w:r><w:drawing><wp:inline xmlns:wp="http://schemas.openxmlformats.org/drawingml/2006/wordprocessingDrawing"><wp:docPr id="41" name="Existing"/></wp:inline></w:drawing></w:r></w:p><w:p><w:r><w:t>[[IMAGE logo()]]</w:t></w:r></w:p>`)
	template := testDOCX(t, map[string][]byte{"word/document.xml": []byte(document)})
	report, err := CreateReport(context.Background(), template, nil, Options{Functions: map[string]Function{
		"logo": func(args ...any) (any, error) { return Image{Data: png, Extension: "png", Width: 1, Height: 1}, nil },
	}})
	if err != nil {
		t.Fatalf("CreateReport: %v", err)
	}
	documentXML := string(readPart(t, report, "word/document.xml"))
	if !strings.Contains(documentXML, `wp:docPr id="42"`) || !strings.Contains(documentXML, `pic:cNvPr id="42"`) {
		t.Fatalf("new drawing did not continue after existing identifier: %s", documentXML)
	}
}

func TestRenderInlineLinkFromObjectExpression(t *testing.T) {
	document := wordDocument(`<w:p><w:r><w:t>[[LINK ({ url: 'https://example.com/document', label: 'رابط تجريبي' })]]</w:t></w:r><w:r><w:t xml:space="preserve"> متاح</w:t></w:r></w:p>`)
	template := testDOCX(t, map[string][]byte{"word/document.xml": []byte(document)})
	report, err := CreateReport(context.Background(), template, nil, Options{})
	if err != nil {
		t.Fatalf("CreateReport: %v", err)
	}
	pkg, err := readPackage(report)
	if err != nil {
		t.Fatal(err)
	}
	documentXML := string(pkg.Parts["word/document.xml"].Data)
	if !strings.Contains(documentXML, "<w:hyperlink") || !strings.Contains(documentXML, "رابط تجريبي") {
		t.Fatalf("hyperlink markup is missing: %s", documentXML)
	}
	rels := string(pkg.Parts["word/_rels/document.xml.rels"].Data)
	if !strings.Contains(rels, `Target="https://example.com/document"`) || !strings.Contains(rels, `TargetMode="External"`) {
		t.Fatalf("external hyperlink relationship is incomplete: %s", rels)
	}
	if got := documentText(t, report); !strings.Contains(got, "رابط تجريبي متاح") {
		t.Fatalf("unexpected visible text: %q", got)
	}
}

func TestRejectsDisallowedLinkScheme(t *testing.T) {
	template := testDOCX(t, map[string][]byte{"word/document.xml": []byte(wordDocument(`<w:p><w:r><w:t>[[LINK ({ url: 'file:///secret', label: 'x' })]]</w:t></w:r></w:p>`))})
	_, err := CreateReport(context.Background(), template, nil, Options{})
	if err == nil || !strings.Contains(err.Error(), "not allowed") {
		t.Fatalf("error = %v, want disallowed scheme", err)
	}
}

func TestRenderHTMLAltChunk(t *testing.T) {
	template := testDOCX(t, map[string][]byte{"word/document.xml": []byte(wordDocument(`<w:p><w:r><w:t>[[HTML html]]</w:t></w:r></w:p>`))})
	report, err := CreateReport(context.Background(), template, map[string]any{"html": `<html><body><p dir="rtl">مرحبا</p></body></html>`}, Options{})
	if err != nil {
		t.Fatalf("CreateReport: %v", err)
	}
	pkg, err := readPackage(report)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(pkg.Parts["word/namat-html-1.html"].Data); !strings.Contains(got, "مرحبا") {
		t.Fatalf("unexpected HTML part: %q", got)
	}
	if !strings.Contains(string(pkg.Parts["word/document.xml"].Data), "<w:altChunk") {
		t.Fatal("document does not contain altChunk")
	}
	if !strings.Contains(string(pkg.Parts["word/_rels/document.xml.rels"].Data), `/aFChunk"`) {
		t.Fatal("document does not contain aFChunk relationship")
	}
}

func TestLiteralXMLAndLineBreaks(t *testing.T) {
	template := testDOCX(t, map[string][]byte{"word/document.xml": []byte(wordDocument(`<w:p><w:r><w:t>[[value]]</w:t></w:r></w:p>`))})
	report, err := CreateReport(context.Background(), template, map[string]any{"value": "أول\nثان||<w:tab/>||ثالث"}, Options{AllowRawXML: true})
	if err != nil {
		t.Fatalf("CreateReport: %v", err)
	}
	documentXML := string(readPart(t, report, "word/document.xml"))
	if !strings.Contains(documentXML, "<w:br") || !strings.Contains(documentXML, "<w:tab") {
		t.Fatalf("line break or literal XML missing: %s", documentXML)
	}
}

func TestRawXMLRequiresOptIn(t *testing.T) {
	template := testDOCX(t, map[string][]byte{"word/document.xml": []byte(wordDocument(`<w:p><w:r><w:t>[[RAW-XML xml()]]</w:t></w:r></w:p>`))})
	options := Options{Functions: map[string]Function{"xml": func(args ...any) (any, error) { return `<w:p><w:r><w:t>خام</w:t></w:r></w:p>`, nil }}}
	if _, err := CreateReport(context.Background(), template, nil, options); err == nil || !strings.Contains(err.Error(), "disabled") {
		t.Fatalf("error = %v, want raw XML opt-in error", err)
	}
	options.AllowRawXML = true
	report, err := CreateReport(context.Background(), template, nil, options)
	if err != nil {
		t.Fatalf("CreateReport with opt-in: %v", err)
	}
	if got := documentText(t, report); !strings.Contains(got, "خام") {
		t.Fatalf("raw XML text missing: %q", got)
	}
}

func TestAliasAndQueryResolver(t *testing.T) {
	document := wordDocument(`<w:p><w:r><w:t>[[QUERY record by key]]</w:t></w:r></w:p><w:p><w:r><w:t>[[ALIAS recordName INS record.name]]</w:t></w:r></w:p><w:p><w:r><w:t>[[*recordName]]</w:t></w:r></w:p>`)
	template := testDOCX(t, map[string][]byte{"word/document.xml": []byte(document)})
	called := false
	report, err := CreateReport(context.Background(), template, map[string]any{"record": map[string]any{"name": "wrong"}}, Options{
		QueryResolver: func(ctx context.Context, query string) (any, error) {
			called = true
			if query != "record by key" {
				t.Fatalf("query = %q", query)
			}
			return map[string]any{"record": map[string]any{"name": "الصحيح"}}, nil
		},
	})
	if err != nil {
		t.Fatalf("CreateReport: %v", err)
	}
	if !called {
		t.Fatal("query resolver was not called")
	}
	text := documentText(t, report)
	if !strings.Contains(text, "الصحيح") || strings.Contains(text, "ALIAS") || strings.Contains(text, "QUERY") {
		t.Fatalf("unexpected rendered text: %q", text)
	}
}
