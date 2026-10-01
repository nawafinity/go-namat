package engine

import (
	"context"
	"encoding/base64"
	"errors"
	"strings"
	"testing"
)

func TestGeneratedPartsHonorPackageLimits(t *testing.T) {
	template := testDOCX(t, map[string][]byte{"word/document.xml": []byte(wordDocument(`<w:p><w:r><w:t>[[@image image]]</w:t></w:r></w:p>`))})
	compiled, err := Compile(template, Options{})
	if err != nil {
		t.Fatal(err)
	}
	compiled.options.MaxPackageParts = len(compiled.pkg.Parts)
	_, err = compiled.Render(context.Background(), map[string]any{
		"image": Image{Data: []byte("image"), Extension: "png", Width: 1, Height: 1},
	})
	if !errors.Is(err, ErrSecurityLimit) || !strings.Contains(err.Error(), "MaxPackageParts") {
		t.Fatalf("generated part limit error = %v", err)
	}
}

func TestRenderInlineImageAddsMediaRelationshipAndContentType(t *testing.T) {
	document := wordDocument(`<w:p><w:r><w:t xml:space="preserve">Before </w:t></w:r><w:r><w:t>[[@image logo()]]</w:t></w:r><w:r><w:t xml:space="preserve"> after</w:t></w:r></w:p>`)
	template := testDOCX(t, map[string][]byte{"word/document.xml": []byte(document)})
	png, err := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII=")
	if err != nil {
		t.Fatal(err)
	}
	report, err := CreateReport(context.Background(), template, nil, Options{
		Functions: map[string]FunctionSpec{
			"logo": testFunction(func(context.Context, ...any) (any, error) {
				return Image{Data: png, Extension: ".png", Width: 1.5, Height: 1.5, Alt: "Logo"}, nil
			}),
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
	if got := documentText(t, report); !strings.Contains(got, "Before  after") || strings.Contains(got, "IMAGE") {
		t.Fatalf("unexpected visible text: %q", got)
	}
}

func TestRenderSVGWithFallbackThumbnail(t *testing.T) {
	png, _ := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII=")
	template := testDOCX(t, map[string][]byte{"word/document.xml": []byte(wordDocument(`<w:p><w:r><w:t>[[@image graphic()]]</w:t></w:r></w:p>`))})
	report, err := CreateReport(context.Background(), template, nil, Options{Functions: map[string]FunctionSpec{
		"graphic": testFunction(func(context.Context, ...any) (any, error) {
			return Image{Data: []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="10" height="10"><rect width="10" height="10"/></svg>`), Extension: "svg", Width: 2, Height: 2, Thumbnail: &Image{Data: png, Extension: "png"}}, nil
		}),
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
		"word/document.xml": []byte(wordDocument(`<w:p><w:r><w:t>[[@image image]]</w:t></w:r></w:p>`)),
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
		"word/header1.xml":  []byte(`<?xml version="1.0"?><w:hdr xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:p><w:r><w:t>[[@image logo()]]</w:t></w:r></w:p></w:hdr>`),
	})
	report, err := CreateReport(context.Background(), template, nil, Options{Functions: map[string]FunctionSpec{
		"logo": testFunction(func(context.Context, ...any) (any, error) {
			return Image{Data: png, Extension: "png", Width: 1, Height: 1}, nil
		}),
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
	document := wordDocument(`<w:p><w:r><w:drawing><wp:inline xmlns:wp="http://schemas.openxmlformats.org/drawingml/2006/wordprocessingDrawing"><wp:docPr id="41" name="Existing"/></wp:inline></w:drawing></w:r></w:p><w:p><w:r><w:t>[[@image logo()]]</w:t></w:r></w:p>`)
	template := testDOCX(t, map[string][]byte{"word/document.xml": []byte(document)})
	report, err := CreateReport(context.Background(), template, nil, Options{Functions: map[string]FunctionSpec{
		"logo": testFunction(func(context.Context, ...any) (any, error) {
			return Image{Data: png, Extension: "png", Width: 1, Height: 1}, nil
		}),
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
	document := wordDocument(`<w:p><w:r><w:t>[[@link ({ url: 'https://example.com/document', label: 'Example link' })]]</w:t></w:r><w:r><w:t xml:space="preserve"> available</w:t></w:r></w:p>`)
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
	if !strings.Contains(documentXML, "<w:hyperlink") || !strings.Contains(documentXML, "Example link") {
		t.Fatalf("hyperlink markup is missing: %s", documentXML)
	}
	rels := string(pkg.Parts["word/_rels/document.xml.rels"].Data)
	if !strings.Contains(rels, `Target="https://example.com/document"`) || !strings.Contains(rels, `TargetMode="External"`) {
		t.Fatalf("external hyperlink relationship is incomplete: %s", rels)
	}
	if got := documentText(t, report); !strings.Contains(got, "Example link available") {
		t.Fatalf("unexpected visible text: %q", got)
	}
}

func TestRejectsDisallowedLinkScheme(t *testing.T) {
	template := testDOCX(t, map[string][]byte{"word/document.xml": []byte(wordDocument(`<w:p><w:r><w:t>[[@link ({ url: 'file:///secret', label: 'x' })]]</w:t></w:r></w:p>`))})
	_, err := CreateReport(context.Background(), template, nil, Options{})
	if err == nil || !strings.Contains(err.Error(), "not allowed") {
		t.Fatalf("error = %v, want disallowed scheme", err)
	}
}

func TestRenderHTMLAltChunk(t *testing.T) {
	template := testDOCX(t, map[string][]byte{"word/document.xml": []byte(wordDocument(`<w:p><w:r><w:t>[[@html html]]</w:t></w:r></w:p>`))})
	report, err := CreateReport(context.Background(), template, map[string]any{"html": `<html><body><p>Hello</p></body></html>`}, Options{})
	if err != nil {
		t.Fatalf("CreateReport: %v", err)
	}
	pkg, err := readPackage(report)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(pkg.Parts["word/namat-html-1.html"].Data); !strings.Contains(got, "Hello") {
		t.Fatalf("unexpected HTML part: %q", got)
	}
	if !strings.Contains(string(pkg.Parts["word/document.xml"].Data), "<w:altChunk") {
		t.Fatal("document does not contain altChunk")
	}
	if !strings.Contains(string(pkg.Parts["word/_rels/document.xml.rels"].Data), `/aFChunk"`) {
		t.Fatal("document does not contain aFChunk relationship")
	}
}

func TestLineBreaksDoNotInterpretLiteralXML(t *testing.T) {
	template := testDOCX(t, map[string][]byte{"word/document.xml": []byte(wordDocument(`<w:p><w:r><w:t>[[value]]</w:t></w:r></w:p>`))})
	report, err := CreateReport(context.Background(), template, map[string]any{"value": "first\nsecond||<w:tab/>||third"}, Options{AllowRawXML: true})
	if err != nil {
		t.Fatalf("CreateReport: %v", err)
	}
	documentXML := string(readPart(t, report, "word/document.xml"))
	if !strings.Contains(documentXML, "<w:br") || strings.Contains(documentXML, "<w:tab") || !strings.Contains(documentXML, "&lt;w:tab/&gt;") {
		t.Fatalf("line break conversion or XML escaping is wrong: %s", documentXML)
	}
}

func TestRawXMLRequiresOptIn(t *testing.T) {
	template := testDOCX(t, map[string][]byte{"word/document.xml": []byte(wordDocument(`<w:p><w:r><w:t>[[@raw-xml xml()]]</w:t></w:r></w:p>`))})
	options := Options{Functions: map[string]FunctionSpec{"xml": testFunction(func(context.Context, ...any) (any, error) { return `<w:p><w:r><w:t>raw</w:t></w:r></w:p>`, nil })}}
	if _, err := CreateReport(context.Background(), template, nil, options); err == nil || !strings.Contains(err.Error(), "disabled") {
		t.Fatalf("error = %v, want raw XML opt-in error", err)
	}
	options.AllowRawXML = true
	report, err := CreateReport(context.Background(), template, nil, options)
	if err != nil {
		t.Fatalf("CreateReport with opt-in: %v", err)
	}
	if got := documentText(t, report); !strings.Contains(got, "raw") {
		t.Fatalf("raw XML text missing: %q", got)
	}
}

func TestLexicalLet(t *testing.T) {
	document := wordDocument(`<w:p><w:r><w:t>[[#let recordName = record.name]]</w:t></w:r></w:p><w:p><w:r><w:t>[[recordName]]</w:t></w:r></w:p>`)
	template := testDOCX(t, map[string][]byte{"word/document.xml": []byte(document)})
	report, err := CreateReport(context.Background(), template, map[string]any{"record": map[string]any{"name": "resolved"}}, Options{})
	if err != nil {
		t.Fatalf("CreateReport: %v", err)
	}
	text := documentText(t, report)
	if !strings.Contains(text, "resolved") || strings.Contains(text, "#let") {
		t.Fatalf("unexpected rendered text: %q", text)
	}
}
