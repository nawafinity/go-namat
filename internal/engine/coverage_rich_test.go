package engine

import (
	"archive/zip"
	"encoding/base64"
	"math"
	"reflect"
	"strings"
	"testing"
)

type imageValue struct {
	Data      any     `json:"data"`
	Extension string  `json:"extension"`
	Width     float64 `json:"width"`
	Height    float64 `json:"height"`
	Alt       string  `json:"alt"`
	Rotation  float64 `json:"rotation"`
	Caption   string  `json:"caption"`
	Thumbnail any     `json:"thumbnail"`
}

type linkValue struct {
	URL     string `json:"url"`
	Label   string `json:"label"`
	Tooltip string `json:"tooltip"`
}

func richTestState() *renderState {
	return &renderState{
		template: &Template{options: Options{}.normalized()},
		counter:  &renderCounter{},
		partName: "word/document.xml",
		pkg: &docxPackage{
			Parts: map[string]*packagePart{
				"[Content_Types].xml": {Header: zip.FileHeader{Name: "[Content_Types].xml", Method: zip.Deflate}, Data: []byte("<Types></Types>")},
			},
		},
	}
}

func TestImageValueConversionAndValidationEdges(t *testing.T) {
	valid := Image{Data: []byte{1}, Extension: "PNG", Width: 1, Height: 2}
	if got, err := imageFromValue(&valid); err != nil || got.Extension != "png" {
		t.Fatalf("image pointer = %#v, %v", got, err)
	}
	withoutThumbnail := imageValue{Data: []byte{1}, Extension: "png", Width: 1, Height: 1}
	var nilThumbnail *imageValue
	withoutThumbnail.Thumbnail = nilThumbnail
	if _, err := imageFromValue(withoutThumbnail); err != nil {
		t.Fatalf("typed nil thumbnail = %v", err)
	}
	jpeg := valid
	jpeg.Extension = ".JPEG"
	if got, err := validateImage(jpeg); err != nil || got.Extension != "jpg" {
		t.Fatalf("JPEG normalization = %#v, %v", got, err)
	}

	thumbnailData := base64.StdEncoding.EncodeToString([]byte{2})
	value := imageValue{
		Data:      base64.StdEncoding.EncodeToString([]byte{1}),
		Extension: "svg",
		Width:     3,
		Height:    4,
		Alt:       "alt",
		Rotation:  5,
		Caption:   "caption",
		Thumbnail: map[string]any{"data": thumbnailData, "extension": "JPEG"},
	}
	got, err := imageFromValue(value)
	if err != nil || got.Thumbnail == nil || got.Thumbnail.Extension != "jpg" || got.Caption != "caption" {
		t.Fatalf("structured image = %#v, %v", got, err)
	}
	value.Thumbnail = map[string]any{"data": "invalid", "extension": "png"}
	if _, err := imageFromValue(value); err == nil || !strings.Contains(err.Error(), "thumbnail") {
		t.Fatalf("invalid thumbnail error = %v", err)
	}

	cases := []struct {
		name  string
		image Image
	}{
		{"empty data", Image{Extension: "png", Width: 1, Height: 1}},
		{"extension", Image{Data: []byte{1}, Extension: "bmp", Width: 1, Height: 1}},
		{"zero width", Image{Data: []byte{1}, Extension: "png", Width: 0, Height: 1}},
		{"NaN", Image{Data: []byte{1}, Extension: "png", Width: math.NaN(), Height: 1}},
		{"infinite", Image{Data: []byte{1}, Extension: "png", Width: 1, Height: math.Inf(1)}},
		{"too large", Image{Data: []byte{1}, Extension: "png", Width: 101, Height: 1}},
		{"empty thumbnail", Image{Data: []byte{1}, Extension: "svg", Width: 1, Height: 1, Thumbnail: &Image{Extension: "png"}}},
		{"SVG thumbnail", Image{Data: []byte{1}, Extension: "svg", Width: 1, Height: 1, Thumbnail: &Image{Data: []byte{1}, Extension: "svg"}}},
		{"unknown thumbnail", Image{Data: []byte{1}, Extension: "svg", Width: 1, Height: 1, Thumbnail: &Image{Data: []byte{1}, Extension: "bmp"}}},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			if _, err := validateImage(test.image); err == nil {
				t.Fatal("invalid image succeeded")
			}
		})
	}
}

func TestImageDataAndLinkConversionEdges(t *testing.T) {
	original := []byte{1, 2}
	got, err := decodeImageData(original)
	if err != nil || !reflect.DeepEqual(got, original) {
		t.Fatalf("byte image = %v, %v", got, err)
	}
	got[0] = 9
	if original[0] != 1 {
		t.Fatal("image bytes were not copied")
	}
	dataURI := "data:image/png;base64," + base64.StdEncoding.EncodeToString([]byte{3})
	if got, err := decodeImageData(dataURI); err != nil || !reflect.DeepEqual(got, []byte{3}) {
		t.Fatalf("data URI = %v, %v", got, err)
	}
	if _, err := decodeImageData("invalid"); err == nil {
		t.Fatal("invalid base64 succeeded")
	}
	if _, err := decodeImageData(1); err == nil {
		t.Fatal("unsupported image data succeeded")
	}

	valid := Link{URL: "https://example.test", Label: "Example"}
	if got, err := linkFromValue(&valid); err != nil || got != valid {
		t.Fatalf("link pointer = %#v, %v", got, err)
	}
	if got, err := linkFromValue(linkValue{URL: " https://example.test ", Tooltip: "tip"}); err != nil || got.Label != "https://example.test" || got.Tooltip != "tip" {
		t.Fatalf("structured link = %#v, %v", got, err)
	}
	if _, err := validateLink(Link{}); err == nil {
		t.Fatal("empty link succeeded")
	}
}

func TestRichNodeFailureEdges(t *testing.T) {
	state := richTestState()
	state.template.options.MaxOutputBytes = 1
	if _, err := state.imageNode(Image{Data: []byte{1, 2}, Extension: "png", Width: 1, Height: 1}); err == nil {
		t.Fatal("oversized image succeeded")
	}
	state.template.options.MaxOutputBytes = 1024
	state.pkg.Parts["[Content_Types].xml"].Data = []byte("<broken>")
	if _, err := state.imageNode(Image{Data: []byte{1}, Extension: "png", Width: 1, Height: 1}); err == nil {
		t.Fatal("image with malformed content types succeeded")
	}

	state = richTestState()
	if _, err := state.linkNode(Link{}); err == nil {
		t.Fatal("empty link node succeeded")
	}
	state = richTestState()
	state.pkg.Parts["word/_rels/document.xml.rels"] = &packagePart{Data: []byte("<broken>")}
	if _, err := state.linkNode(Link{URL: "https://example.test"}); err == nil {
		t.Fatal("link with malformed relationships succeeded")
	}
	state = richTestState()
	for _, link := range []Link{{URL: "relative"}, {URL: "ftp://example.test"}} {
		if _, err := state.linkNode(link); err == nil {
			t.Fatalf("invalid link %#v succeeded", link)
		}
	}

	state = richTestState()
	if _, err := state.htmlNode(nil); err == nil {
		t.Fatal("null HTML succeeded")
	}
	state = richTestState()
	state.pkg.Parts["[Content_Types].xml"].Data = []byte("<broken>")
	if _, err := state.htmlNode("<p>content</p>"); err == nil {
		t.Fatal("HTML with malformed content types succeeded")
	}
	state = richTestState()
	state.pkg.Parts["word/_rels/document.xml.rels"] = &packagePart{Data: []byte("<broken>")}
	if _, err := state.htmlNode("<p>content</p>"); err == nil {
		t.Fatal("HTML with malformed relationships succeeded")
	}

	state = richTestState()
	state.pkg.Parts["[Content_Types].xml"].Data = []byte("<Types><Default Extension=\"svg\" ContentType=\"image/svg+xml\"/><Default Extension=\"png\" ContentType=\"wrong\"/></Types>")
	image := Image{Data: []byte{1}, Extension: "svg", Width: 1, Height: 1, Thumbnail: &Image{Data: []byte{2}, Extension: "png"}}
	if _, err := state.imageNode(image); err == nil {
		t.Fatal("SVG thumbnail resource failure was not propagated")
	}
}

func TestTextRangeReplacementEdges(t *testing.T) {
	inserted := elementNode("inserted")
	empty := elementNode("p")
	if err := replaceTextRangeWithNode(empty, 0, 0, inserted); err == nil {
		t.Fatal("replacement in empty paragraph succeeded")
	}
	paragraph := mustParseElement(t, "<w:p><w:r><w:t>abc</w:t></w:r><w:r><w:t>def</w:t></w:r></w:p>")
	paragraph.Children = append([]*xmlNode{elementNode("bookmark")}, paragraph.Children...)
	for _, bounds := range [][2]int{{-1, 1}, {2, 1}, {0, 99}} {
		clone := paragraph.clone()
		if err := replaceTextRangeWithNode(clone, bounds[0], bounds[1], inserted); err == nil {
			t.Fatalf("invalid bounds %v succeeded", bounds)
		}
	}
	clone := paragraph.clone()
	if err := replaceTextRangeWithNode(clone, 1, 5, inserted); err != nil {
		t.Fatalf("multi-run replacement: %v", err)
	}
	if got := textOfParagraph(clone); got != "af" {
		t.Fatalf("multi-run remaining text = %q", got)
	}
	threeRuns := mustParseElement(t, "<w:p><w:r><w:t>ab</w:t></w:r><w:r><w:t>cd</w:t></w:r><w:r><w:t>ef</w:t></w:r></w:p>")
	if err := replaceTextRangeWithNode(threeRuns, 1, 5, inserted); err != nil || textOfParagraph(threeRuns) != "af" {
		t.Fatalf("three-run rich replacement = %q, %v", textOfParagraph(threeRuns), err)
	}
	multipleText := mustParseElement(t, "<w:r><w:t>a</w:t><w:t>b</w:t></w:r>")
	setExistingText(multipleText, "value")
	if got := textOfParagraph(multipleText); got != "value" {
		t.Fatalf("setExistingText = %q", got)
	}
}

func TestSingleXMLNodeValidation(t *testing.T) {
	if _, err := singleXMLNode("<broken>", "test node"); err == nil {
		t.Fatal("malformed single XML node succeeded")
	}
	if _, err := singleXMLNode("<a/><b/>", "test node"); err == nil {
		t.Fatal("multiple XML nodes succeeded")
	}
	if node, err := singleXMLNode("<a/>", "test node"); err != nil || !node.is("a") {
		t.Fatalf("single XML node = %#v, %v", node, err)
	}
}

func TestTextMarkupEdges(t *testing.T) {
	root := mustParseElement(t, "<w:p><w:r><w:t>line1\nline2</w:t><w:tab/></w:r></w:p>")
	if err := expandTextMarkup(root, Options{}); err != nil {
		t.Fatal(err)
	}
	if len(root.descendants("br")) != 1 {
		t.Fatalf("line break count = %d", len(root.descendants("br")))
	}

	escaped := mustParseElement(t, "<w:p><w:r><w:t>||&lt;w:br/&gt;||</w:t></w:r></w:p>")
	if err := expandTextMarkup(escaped, Options{}); err != nil || textOfParagraph(escaped) != "||<w:br/>||" || len(escaped.descendants("br")) != 0 {
		t.Fatalf("literal-looking text was interpreted: %q, %v", textOfParagraph(escaped), err)
	}
}

func TestRichReflectionHelpers(t *testing.T) {
	type sample struct {
		Visible string `json:"visible"`
		hidden  string
	}
	type stringKey string
	var nilPointer *sample
	var nilInterface any
	pointerToNilInterface := &nilInterface
	for _, test := range []struct {
		name  string
		value any
		field string
		want  any
		ok    bool
	}{
		{"nil", nil, "field", nil, false},
		{"nil pointer", nilPointer, "field", nil, false},
		{"pointer to nil interface", pointerToNilInterface, "field", nil, false},
		{"map exact", map[string]any{"field": 1}, "field", 1, true},
		{"map case mismatch", map[string]any{"FiElD": 1}, "field", nil, false},
		{"non-string map", map[int]string{1: "x"}, "field", nil, false},
		{"named string map", map[stringKey]int{"field": 2}, "field", 2, true},
		{"struct", sample{Visible: "yes"}, "visible", "yes", true},
		{"unexported", sample{hidden: "no"}, "hidden", nil, false},
		{"unsupported", 1, "field", nil, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, ok := valueField(test.value, test.field)
			if ok != test.ok || !reflect.DeepEqual(got, test.want) {
				t.Fatalf("valueField = %#v, %v; want %#v, %v", got, ok, test.want, test.ok)
			}
		})
	}

	value := int16(3)
	var nilNumber *int
	numbers := []struct {
		value any
		want  float64
	}{
		{nil, 0}, {nilNumber, 0}, {&value, 3},
		{int(1), 1}, {int8(1), 1}, {int16(1), 1}, {int32(1), 1}, {int64(1), 1},
		{uint(1), 1}, {uint8(1), 1}, {uint16(1), 1}, {uint32(1), 1}, {uint64(1), 1},
		{float32(1.5), 1.5}, {float64(2.5), 2.5}, {"3", 0},
	}
	for _, test := range numbers {
		if got := numericValue(test.value); got != test.want {
			t.Fatalf("numericValue(%T) = %v, want %v", test.value, got, test.want)
		}
	}
	if got := pathBase("file.xml"); got != "file.xml" {
		t.Fatalf("pathBase = %q", got)
	}
}

func mustParseElement(t *testing.T, source string) *xmlNode {
	t.Helper()
	root, err := parseXML([]byte(source))
	if err != nil {
		t.Fatal(err)
	}
	return firstElement(root)
}
