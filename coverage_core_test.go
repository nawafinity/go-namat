package namat

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"io"
	"math"
	"reflect"
	"strings"
	"testing"
	"time"
)

type errorReader struct{ err error }

func (r errorReader) Read([]byte) (int, error) { return 0, r.err }

type errorReadCloser struct{ err error }

func (r errorReadCloser) Read([]byte) (int, error) { return 0, r.err }
func (errorReadCloser) Close() error               { return nil }

type errorWriter struct{ err error }

func (w errorWriter) Write([]byte) (int, error) { return 0, w.err }

type coverageShortWriter struct{}

func (coverageShortWriter) Write(p []byte) (int, error) { return len(p) - 1, nil }

type rootStringer string

func (s rootStringer) String() string { return string(s) }

func rawZIP(t *testing.T, entries []struct {
	name string
	data []byte
}) []byte {
	t.Helper()
	var out bytes.Buffer
	w := zip.NewWriter(&out)
	for _, entry := range entries {
		part, err := w.Create(entry.name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := part.Write(entry.data); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}

func TestCoreCommandAndErrorEdges(t *testing.T) {
	if _, err := parseCommand(string([]byte{0xff})); err == nil {
		t.Fatal("invalid UTF-8 command succeeded")
	}
	command, err := parseCommand("ALIAS greeting INS name")
	if err != nil || command.Type != CommandAlias || command.Variable != "greeting" {
		t.Fatalf("alias = %#v, %v", command, err)
	}
	if _, err := parseCommand("ALIAS 1bad INS name"); err == nil {
		t.Fatal("invalid alias name succeeded")
	}

	inner := errors.New("inner")
	if got := (&MultiError{}).Error(); got != "namat: no errors" {
		t.Fatalf("empty MultiError = %q", got)
	}
	var nilMulti *MultiError
	if nilMulti.Unwrap() != nil {
		t.Fatal("nil MultiError unwrap was not nil")
	}
	multi := &MultiError{Errors: []error{inner}}
	if got := multi.Unwrap(); len(got) != 1 || !errors.Is(got[0], inner) {
		t.Fatalf("MultiError unwrap = %#v", got)
	}
}

func TestMetadataFailureAndConversionEdges(t *testing.T) {
	if _, err := GetMetadata([]byte("not a zip")); err == nil {
		t.Fatal("invalid package metadata succeeded")
	}
	for _, test := range []struct {
		name  string
		parts map[string][]byte
	}{
		{"core", map[string][]byte{"word/document.xml": []byte(wordDocument("")), "docProps/core.xml": []byte("<broken>")}},
		{"extended", map[string][]byte{"word/document.xml": []byte(wordDocument("")), "docProps/app.xml": []byte("<broken>")}},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := GetMetadata(testDOCX(t, test.parts)); err == nil {
				t.Fatal("malformed metadata succeeded")
			}
		})
	}
	if _, err := simpleXMLValues([]byte("<broken>")); err == nil {
		t.Fatal("malformed XML values succeeded")
	}
	if metadataTime("") != nil || metadataTime("not-a-time") != nil {
		t.Fatal("invalid metadata time was accepted")
	}
	if got := metadataInt("invalid"); got != 0 {
		t.Fatalf("invalid metadata integer = %d", got)
	}
}

func TestCompileAndRenderPublicFailureEdges(t *testing.T) {
	document := testDOCX(t, map[string][]byte{"word/document.xml": []byte(wordDocument(""))})
	if _, err := Compile(document, Options{OpenDelimiter: "x", CloseDelimiter: "x"}); err == nil {
		t.Fatal("matching delimiters succeeded")
	}
	if _, err := Compile(document, Options{MaxTemplateBytes: 1}); !errors.Is(err, ErrSecurityLimit) {
		t.Fatalf("template limit error = %v", err)
	}
	wantReadErr := errors.New("read failed")
	if _, err := CompileReader(errorReader{err: wantReadErr}, Options{}); !errors.Is(err, wantReadErr) {
		t.Fatalf("CompileReader error = %v", err)
	}
	if _, err := CreateReport(context.Background(), []byte("bad"), nil, Options{}); err == nil {
		t.Fatal("CreateReport accepted invalid template")
	}
	if _, err := CreateReportReader(context.Background(), strings.NewReader("bad"), nil, Options{}); err == nil {
		t.Fatal("CreateReportReader accepted invalid template")
	}

	queryTemplate := &Template{options: Options{}.normalized(), pkg: &docxPackage{}, queries: []string{"one", "two"}}
	if _, err := queryTemplate.Render(context.Background(), nil); err == nil {
		t.Fatal("multiple queries rendered")
	}
	queryTemplate.queries = []string{"one"}
	if _, err := queryTemplate.Render(context.Background(), nil); err == nil {
		t.Fatal("query without resolver rendered")
	}
	queryTemplate.options.QueryResolver = func(context.Context, string) (any, error) { return nil, wantReadErr }
	if _, err := queryTemplate.Render(context.Background(), nil); !errors.Is(err, wantReadErr) {
		t.Fatalf("query resolver error = %v", err)
	}

	compiled, err := Compile(document, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if err := compiled.RenderTo(context.Background(), nil, nil); err == nil {
		t.Fatal("nil output writer succeeded")
	}
	if err := compiled.RenderTo(context.Background(), errorWriter{err: wantReadErr}, nil); !errors.Is(err, wantReadErr) {
		t.Fatalf("output write error = %v", err)
	}
	if err := compiled.RenderTo(context.Background(), coverageShortWriter{}, nil); !errors.Is(err, io.ErrShortWrite) {
		t.Fatalf("short output error = %v", err)
	}
	limited, err := Compile(document, Options{MaxOutputBytes: 1})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := limited.Render(context.Background(), nil); !errors.Is(err, ErrSecurityLimit) {
		t.Fatalf("output limit error = %v", err)
	}
	invalidCompression, err := Compile(document, Options{})
	if err != nil {
		t.Fatal(err)
	}
	invalidCompression.options.CompressionLevel = 100
	if _, err := invalidCompression.Render(context.Background(), nil); err == nil {
		t.Fatal("render with invalid compression succeeded")
	}
	queryTemplate.queries = []string{"one", "two"}
	if err := queryTemplate.RenderTo(context.Background(), io.Discard, nil); err == nil {
		t.Fatal("RenderTo did not propagate render failure")
	}
}

func TestCompilationValidationEdges(t *testing.T) {
	for _, test := range []struct {
		name string
		xml  string
	}{
		{"malformed XML", "<broken>[[value]]"},
		{"malformed command", wordDocument("<w:p><w:r><w:t>[[ELSE extra]]</w:t></w:r></w:p>")},
		{"duplicate alias", wordDocument("<w:p><w:r><w:t>[[ALIAS a INS value]]</w:t></w:r></w:p><w:p><w:r><w:t>[[ALIAS a INS other]]</w:t></w:r></w:p>")},
		{"invalid alias body", wordDocument("<w:p><w:r><w:t>[[ALIAS a ELSE extra]]</w:t></w:r></w:p>")},
	} {
		t.Run(test.name, func(t *testing.T) {
			document := testDOCX(t, map[string][]byte{"word/document.xml": []byte(test.xml)})
			if _, err := Compile(document, Options{}); err == nil {
				t.Fatal("invalid template compiled")
			}
		})
	}

	collectXML := wordDocument("<w:p><w:r><w:t>[[INS @]]</w:t></w:r></w:p><w:p><w:r><w:t>[[IF active]]</w:t></w:r></w:p>")
	_, err := Compile(testDOCX(t, map[string][]byte{"word/document.xml": []byte(collectXML)}), Options{CollectErrors: true})
	var multi *MultiError
	if !errors.As(err, &multi) || len(multi.Errors) != 2 {
		t.Fatalf("collected error = %#v", err)
	}

	malformed := testDOCX(t, map[string][]byte{"word/document.xml": []byte("<broken>[[x]]")})
	if _, err := ListCommands(malformed, Options{}); err == nil {
		t.Fatal("ListCommands accepted malformed XML")
	}
	badCommand := testDOCX(t, map[string][]byte{"word/document.xml": []byte(wordDocument("<w:p><w:r><w:t>[[ELSE extra]]</w:t></w:r></w:p>"))})
	if _, err := ListCommands(badCommand, Options{}); err == nil {
		t.Fatal("ListCommands accepted malformed command")
	}
	if _, err := ListCommands([]byte("bad"), Options{}); err == nil {
		t.Fatal("ListCommands accepted invalid package")
	}
	valid := testDOCX(t, map[string][]byte{"word/document.xml": []byte(wordDocument(""))})
	if _, err := ListCommands(valid, Options{MaxTemplateBytes: 1}); !errors.Is(err, ErrSecurityLimit) {
		t.Fatalf("ListCommands limit error = %v", err)
	}
}

func TestNamatValueHelperEdges(t *testing.T) {
	text := "abc"
	var nilPointer *string
	for _, test := range []struct {
		value any
		want  int
		err   bool
	}{
		{nil, 0, false},
		{nilPointer, 0, false},
		{&text, 3, false},
		{[2]int{}, 2, false},
		{map[string]int{"a": 1}, 1, false},
		{1, 0, true},
	} {
		got, err := collectionLength(test.value)
		if got != test.want || (err != nil) != test.err {
			t.Fatalf("collectionLength(%T) = %d, %v", test.value, got, err)
		}
	}
	for value, want := range map[any]string{float32(1.5): "1.5", rootStringer("value"): "value", 2: "2"} {
		if got := formatValue(value); got != want {
			t.Fatalf("formatValue(%#v) = %q, want %q", value, got, want)
		}
	}
	for _, source := range []string{"= 1", "name =", "bad-name = 1"} {
		if _, _, err := parseAssignment(source); err == nil {
			t.Fatalf("parseAssignment(%q) succeeded", source)
		}
	}

	functions := builtinFunctions()
	if _, err := functions["len"](); err == nil {
		t.Fatal("len accepted zero arguments")
	}
	if _, err := functions["string"](); err == nil {
		t.Fatal("string accepted zero arguments")
	}
	if got, err := functions["len"]("abc"); err != nil || got != 3 {
		t.Fatalf("builtin len = %#v, %v", got, err)
	}
	if got, err := functions["string"](2); err != nil || got != "2" {
		t.Fatalf("builtin string = %#v, %v", got, err)
	}

	ctx, cancel := renderContext(nil, time.Second)
	defer cancel()
	if ctx == nil {
		t.Fatal("renderContext returned nil")
	}
}

func TestPackageValidationAndSerializationEdges(t *testing.T) {
	required := []struct {
		name string
		data []byte
	}{
		{"[Content_Types].xml", []byte("<Types/>")},
		{"word/document.xml", []byte("<document/>")},
	}
	for _, test := range []struct {
		name    string
		entries []struct {
			name string
			data []byte
		}
		partLimit  int64
		totalLimit int64
		partsLimit int
	}{
		{"too many parts", required, 1000, 1000, 1},
		{"absolute name", append(required, struct {
			name string
			data []byte
		}{"/bad", nil}), 1000, 1000, 10},
		{"unclean name", append(required, struct {
			name string
			data []byte
		}{"word/../bad", nil}), 1000, 1000, 10},
		{"part too large", required, 1, 1000, 10},
		{"total too large", required, 1000, 1, 10},
	} {
		t.Run(test.name, func(t *testing.T) {
			data := rawZIP(t, test.entries)
			if _, err := readPackageWithLimits(data, test.partLimit, test.totalLimit, test.partsLimit); err == nil {
				t.Fatal("invalid package succeeded")
			}
		})
	}
	if _, err := readPackage(rawZIP(t, required[:1])); err == nil {
		t.Fatal("package without document succeeded")
	}
	if _, err := readPackage(rawZIP(t, required[1:])); err == nil {
		t.Fatal("package without content types succeeded")
	}
	unsupported := rawZIPWithMethod(t, 99, "[Content_Types].xml", []byte("content"))
	if _, err := readPackageWithLimits(unsupported, 1000, 1000, 10); err == nil {
		t.Fatal("package with unsupported compression succeeded")
	}
	corrupt := rawZIPWithMethod(t, zip.Store, "[Content_Types].xml", []byte("unique-payload-for-corruption"))
	index := bytes.Index(corrupt, []byte("unique-payload-for-corruption"))
	if index < 0 {
		t.Fatal("stored payload not found")
	}
	corrupt[index] ^= 0xff
	if _, err := readPackageWithLimits(corrupt, 1000, 1000, 10); err == nil {
		t.Fatal("corrupt package payload succeeded")
	}

	pkg := &docxPackage{Parts: map[string]*packagePart{}, Order: []string{"missing"}}
	pkg.Parts["extra"] = &packagePart{Header: zip.FileHeader{Name: "extra", Method: zip.Deflate}, Data: []byte("value")}
	data, err := pkg.bytes()
	if err != nil {
		t.Fatal(err)
	}
	if len(data) == 0 {
		t.Fatal("serialized package is empty")
	}
	if _, err := pkg.bytesWithCompression(100); err == nil {
		t.Fatal("invalid compression level succeeded")
	}
	orderedInvalid := &docxPackage{Parts: map[string]*packagePart{"bad": {Header: zip.FileHeader{Method: 99}}}, Order: []string{"bad"}}
	if _, err := orderedInvalid.bytes(); err == nil {
		t.Fatal("ordered unsupported ZIP method succeeded")
	}
	extraInvalid := &docxPackage{Parts: map[string]*packagePart{"bad": {Header: zip.FileHeader{Method: 99}}}}
	if _, err := extraInvalid.bytes(); err == nil {
		t.Fatal("extra unsupported ZIP method succeeded")
	}
	if _, err := readPackagePart(errorReadCloser{err: errors.New("read")}, 10, "part"); err == nil {
		t.Fatal("package part read error succeeded")
	}
	if _, err := readPackagePart(io.NopCloser(strings.NewReader("too long")), 2, "part"); !errors.Is(err, ErrSecurityLimit) {
		t.Fatalf("package part post-read limit error = %v", err)
	}
	if got, err := readPackagePart(io.NopCloser(strings.NewReader("ok")), 2, "part"); err != nil || string(got) != "ok" {
		t.Fatalf("package part read = %q, %v", got, err)
	}
}

func rawZIPWithMethod(t *testing.T, method uint16, name string, data []byte) []byte {
	t.Helper()
	var out bytes.Buffer
	w := zip.NewWriter(&out)
	w.RegisterCompressor(method, func(destination io.Writer) (io.WriteCloser, error) {
		return nopWriteCloser{Writer: destination}, nil
	})
	header := &zip.FileHeader{Name: name, Method: method}
	part, err := w.CreateHeader(header)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}

type nopWriteCloser struct{ io.Writer }

func (nopWriteCloser) Close() error { return nil }

func TestOOXMLPackageHelperEdges(t *testing.T) {
	pkg := &docxPackage{Parts: map[string]*packagePart{}, Order: nil}
	for _, name := range []string{"", "../bad"} {
		if err := pkg.addPart(name, nil, zip.Store); err == nil {
			t.Fatalf("addPart(%q) succeeded", name)
		}
	}
	if err := pkg.addPart("/word\\part.xml", []byte("value"), zip.Store); err != nil {
		t.Fatal(err)
	}
	if err := pkg.addPart("word/part.xml", nil, zip.Store); err == nil {
		t.Fatal("duplicate part succeeded")
	}

	pkg.Parts["word/_rels/document.xml.rels"] = &packagePart{Data: []byte("<broken>")}
	if _, err := pkg.addRelationship("word/document.xml", "type", "target", ""); err == nil {
		t.Fatal("malformed relationships succeeded")
	}
	pkg.Parts["word/_rels/document.xml.rels"].Data = []byte("<Wrong/>")
	if _, err := pkg.addRelationship("word/document.xml", "type", "target", ""); err == nil {
		t.Fatal("wrong relationships root succeeded")
	}
	pkg.Parts["word/_rels/document.xml.rels"].Data = []byte("<Relationships><Relationship Id=\"rIdNamat1\"/></Relationships>")
	id, err := pkg.addRelationship("word/document.xml", "type", "target", "External")
	if err != nil || id != "rIdNamat2" {
		t.Fatalf("relationship = %q, %v", id, err)
	}

	for _, data := range []string{"<broken>", "<Wrong/>"} {
		pkg.Parts["[Content_Types].xml"] = &packagePart{Data: []byte(data)}
		if err := pkg.ensureDefaultContentType("png", "image/png"); err == nil {
			t.Fatalf("content types %q succeeded", data)
		}
	}
	pkg.Parts["[Content_Types].xml"] = &packagePart{Data: []byte("<Types><Default Extension=\"png\" ContentType=\"wrong\"/></Types>")}
	if err := pkg.ensureDefaultContentType(".PNG", "image/png"); err == nil {
		t.Fatal("conflicting content type succeeded")
	}
	pkg.Parts["[Content_Types].xml"].Data = []byte("<Types><Default Extension=\"png\" ContentType=\"image/png\"/></Types>")
	if err := pkg.ensureDefaultContentType("PNG", "image/png"); err != nil {
		t.Fatalf("matching content type: %v", err)
	}
	fresh := &docxPackage{Parts: map[string]*packagePart{"[Content_Types].xml": {Data: []byte("<Types></Types>")}}}
	if _, err := fresh.addRelationship("word/document.xml", "type", "target", ""); err != nil {
		t.Fatalf("new relationships part: %v", err)
	}
	if err := fresh.ensureDefaultContentType("gif", "image/gif"); err != nil {
		t.Fatalf("new content type: %v", err)
	}
	if firstElement(&xmlNode{}) != nil || attributeValue(&xmlNode{}, "missing") != "" {
		t.Fatal("empty XML helper returned a value")
	}
}

func TestXMLNodeEdges(t *testing.T) {
	root, err := parseXML([]byte("<?pi data?><!directive><!--comment--><root attr=\"&amp;\">text</root>"))
	if err != nil {
		t.Fatal(err)
	}
	data := root.bytes()
	for _, fragment := range []string{"<?pi data?>", "<!directive>", "<!--comment-->", "attr=\"&amp;\""} {
		if !strings.Contains(string(data), fragment) {
			t.Fatalf("serialized XML missing %q: %s", fragment, data)
		}
	}
	if _, err := parseXML([]byte("<root>")); err == nil {
		t.Fatal("unclosed XML succeeded")
	}
	if _, err := parseXML([]byte("<a></b>")); err == nil {
		t.Fatal("mismatched XML succeeded")
	}
	if _, err := parseXML([]byte("</a>")); err == nil {
		t.Fatal("unexpected closing XML succeeded")
	}
	if _, err := parseXML([]byte("<a>&unknown;</a>")); err == nil {
		t.Fatal("invalid XML entity succeeded")
	}
	if got := (*xmlNode)(nil).clone(); got != nil {
		t.Fatalf("nil clone = %#v", got)
	}
	empty := &xmlNode{}
	if err := setTextOfNode(empty, ""); err != nil {
		t.Fatal(err)
	}
	if err := setTextOfNode(empty, "value"); err == nil {
		t.Fatal("setting missing text node succeeded")
	}
	textNode := &xmlNode{Type: xmlElement, Name: xmlName("t"), Children: []*xmlNode{{Type: xmlText, Data: "a"}, {Type: xmlText, Data: "b"}}}
	if err := setTextOfNode(textNode, "value"); err != nil {
		t.Fatal(err)
	}
	if textNode.Children[0].Data != "value" || textNode.Children[1].Data != "" {
		t.Fatalf("setTextOfNode result = %#v", textNode.Children)
	}
	if _, err := parseXMLFragment("<broken>"); err == nil {
		t.Fatal("malformed fragment succeeded")
	}
	if nodes, err := parseXMLFragment(""); err != nil || len(nodes) != 0 {
		t.Fatalf("empty fragment = %#v, %v", nodes, err)
	}
}

func xmlName(local string) xml.Name { return xml.Name{Local: local} }

func TestDrawingAndObjectHelpers(t *testing.T) {
	root, err := parseXML([]byte("<root><docPr id=\"4\"/><docPr id=\"bad\"/><cNvPr id=\"7\"/></root>"))
	if err != nil {
		t.Fatal(err)
	}
	if got := maxDrawingIdentifier(map[string]*xmlNode{"part": root}); got != 7 {
		t.Fatalf("max drawing id = %d", got)
	}
	var nilPointer *struct{}
	for _, test := range []struct {
		value any
		want  bool
	}{
		{nil, false},
		{rootStringer("x"), false},
		{nilPointer, false},
		{map[string]int{}, true},
		{[]int{}, true},
		{[1]int{}, true},
		{struct{}{}, true},
		{1, false},
	} {
		if got := isObjectResult(test.value); got != test.want {
			t.Fatalf("isObjectResult(%T) = %v, want %v", test.value, got, test.want)
		}
	}
	if expressionTruthy(nil) || expressionTruthy(false) || expressionTruthy(" ") || expressionTruthy(float64(0)) || expressionTruthy(float32(0)) || expressionTruthy(0) {
		t.Fatal("false-like value was truthy")
	}
	for _, value := range []any{true, "x", float64(1), float32(1), 1, struct{}{}} {
		if !expressionTruthy(value) {
			t.Fatalf("expressionTruthy(%#v) = false", value)
		}
	}
	if math.IsNaN(numericValue(math.NaN())) == false {
		t.Fatal("numericValue did not preserve NaN")
	}
	if !reflect.DeepEqual(cloneNodes(nil), []*xmlNode{}) {
		t.Fatal("cloneNodes(nil) was not an empty slice")
	}
}
