package namat

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"unicode/utf8"

	internalengine "github.com/nawafinity/go-namat/internal/engine"
)

func TestPublicFacadeDelegatesToEngine(t *testing.T) {
	_, _ = Compile(nil, Options{})
	_, _ = CompileReader(nil, Options{})
	_, _ = CreateReport(context.Background(), nil, nil, Options{})
	_, _ = CreateReportReader(context.Background(), nil, nil, Options{})
	_, _ = ListCommands(nil, Options{})
	_, _ = GetMetadata(nil)
}

func TestPublicFacadeSuccessPaths(t *testing.T) {
	document := facadeDOCX(t, map[string]string{
		"word/document.xml": `<?xml version="1.0"?><w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body><w:p><w:r><w:t>[[upper(name)]]</w:t></w:r></w:p></w:body></w:document>`,
		"docProps/core.xml": `<?xml version="1.0"?><cp:coreProperties xmlns:cp="http://schemas.openxmlformats.org/package/2006/metadata/core-properties" xmlns:dc="http://purl.org/dc/elements/1.1/"><dc:title>Facade fixture</dc:title></cp:coreProperties>`,
	})
	options := Options{
		Functions: map[string]Function{
			"upper": func(args ...any) (any, error) { return strings.ToUpper(fmt.Sprint(args[0])), nil },
		},
		QueryResolver:      func(context.Context, string) (any, error) { return nil, nil },
		ErrorHandler:       func(string, error) (any, error) { return nil, nil },
		AllowedLinkSchemes: []string{"https"},
	}

	compiled, err := Compile(document, options)
	if err != nil {
		t.Fatal(err)
	}
	if len(compiled.Commands()) != 1 || compiled.Commands()[0].Type != CommandInsert {
		t.Fatalf("commands = %#v", compiled.Commands())
	}
	if commands := (*Template)(nil).Commands(); commands != nil {
		t.Fatalf("nil template commands = %#v", commands)
	}
	report, err := compiled.Render(context.Background(), map[string]any{"name": "namat"})
	if err != nil || len(report) == 0 {
		t.Fatalf("Render = %d bytes, %v", len(report), err)
	}
	var rendered bytes.Buffer
	if err := compiled.RenderTo(context.Background(), &rendered, map[string]any{"name": "namat"}); err != nil || rendered.Len() == 0 {
		t.Fatalf("RenderTo = %d bytes, %v", rendered.Len(), err)
	}
	if _, err := CompileReader(bytes.NewReader(document), options); err != nil {
		t.Fatalf("CompileReader: %v", err)
	}
	if report, err := CreateReport(context.Background(), document, map[string]any{"name": "namat"}, options); err != nil || len(report) == 0 {
		t.Fatalf("CreateReport = %d bytes, %v", len(report), err)
	}
	if report, err := CreateReportReader(context.Background(), bytes.NewReader(document), map[string]any{"name": "namat"}, options); err != nil || len(report) == 0 {
		t.Fatalf("CreateReportReader = %d bytes, %v", len(report), err)
	}
	commands, err := ListCommands(document, options)
	if err != nil || len(commands) != 1 || commands[0].Expression != "upper(name)" {
		t.Fatalf("ListCommands = %#v, %v", commands, err)
	}
	metadata, err := GetMetadata(document)
	if err != nil || metadata.Title != "Facade fixture" {
		t.Fatalf("GetMetadata = %#v, %v", metadata, err)
	}
}

func TestPublicErrorTranslationAndTypes(t *testing.T) {
	if publicError(nil) != nil {
		t.Fatal("nil error was not preserved")
	}
	inner := errors.New("synthetic failure")
	translated := publicError(&internalengine.Error{Part: "word/document.xml", Command: "INS value", Err: inner})
	var templateErr *Error
	if !errors.As(translated, &templateErr) || !errors.Is(translated, inner) {
		t.Fatalf("translated error = %T %v", translated, translated)
	}
	if templateErr.Error() == "" || templateErr.Unwrap() != inner {
		t.Fatalf("template error contract = %v", templateErr)
	}
	if (&Error{Part: "part", Err: inner}).Error() == "" || (&Error{Err: inner}).Error() == "" {
		t.Fatal("template error formatting returned an empty message")
	}

	multiple := publicError(&internalengine.MultiError{Errors: []error{inner}})
	var multi *MultiError
	if !errors.As(multiple, &multi) || len(multi.Unwrap()) != 1 || multi.Error() == "" {
		t.Fatalf("translated multi-error = %T %v", multiple, multiple)
	}
	var nilMulti *MultiError
	if nilMulti.Error() != "namat: no errors" || nilMulti.Unwrap() != nil || (&MultiError{}).Error() != "namat: no errors" {
		t.Fatal("empty multi-error contract changed")
	}

	categories := []struct {
		internal error
		public   error
	}{
		{internalengine.ErrInvalidTemplate, ErrInvalidTemplate},
		{internalengine.ErrCommandSyntax, ErrCommandSyntax},
		{internalengine.ErrCommandExecution, ErrCommandExecution},
		{internalengine.ErrNullishResult, ErrNullishResult},
		{internalengine.ErrObjectResult, ErrObjectResult},
		{internalengine.ErrSecurityLimit, ErrSecurityLimit},
	}
	for _, category := range categories {
		translated = publicError(fmt.Errorf("detail: %w", category.internal))
		if !errors.Is(translated, category.public) || translated.Error() == "" {
			t.Fatalf("category %v was not translated: %v", category.public, translated)
		}
		if len(translated.(categorizedError).Unwrap()) != 2 {
			t.Fatalf("category unwrap = %#v", translated)
		}
	}
	if got := publicError(inner); got != inner {
		t.Fatalf("ordinary error changed: %v", got)
	}
}

func facadeDOCX(t *testing.T, parts map[string]string) []byte {
	t.Helper()
	base := map[string]string{
		"[Content_Types].xml": `<?xml version="1.0"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/><Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/></Types>`,
		"_rels/.rels":         `<?xml version="1.0"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/></Relationships>`,
	}
	for name, content := range parts {
		base[name] = content
	}
	var output bytes.Buffer
	writer := zip.NewWriter(&output)
	for name, content := range base {
		entry, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entry.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return output.Bytes()
}

func TestCommittedTestsUseEnglishAndProductNeutralData(t *testing.T) {
	forbidden := []string{"o" + "eg", "data." + "gov.sa/" + "oe", "od" + "oe"}
	err := filepath.WalkDir(".", func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if path == ".git" || path == "dist" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, "_test.go") && !strings.Contains(filepath.ToSlash(path), "/testdata/") {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		lower := strings.ToLower(string(content))
		for _, marker := range forbidden {
			if strings.Contains(lower, marker) {
				t.Errorf("%s contains product-specific marker %q", path, marker)
			}
		}
		if utf8.Valid(content) {
			for _, r := range string(content) {
				if isArabicScriptRune(r) {
					t.Errorf("%s contains a literal Arabic-script rune; use English test source and Unicode escapes for multilingual cases", path)
					break
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func isArabicScriptRune(r rune) bool {
	return r >= 0x0600 && r <= 0x06ff ||
		r >= 0x0750 && r <= 0x077f ||
		r >= 0x0870 && r <= 0x089f ||
		r >= 0x08a0 && r <= 0x08ff ||
		r >= 0xfb50 && r <= 0xfdff ||
		r >= 0xfe70 && r <= 0xfeff
}

func TestFeatureCoverageMatrixReferencesExistingTests(t *testing.T) {
	matrix, err := os.ReadFile("docs/FEATURE_COVERAGE.md")
	if err != nil {
		t.Fatal(err)
	}
	referencePattern := regexp.MustCompile(`(?:Test|Fuzz)[A-Za-z0-9_]+`)
	references := referencePattern.FindAllString(string(matrix), -1)
	if len(references) < 40 {
		t.Fatalf("feature matrix has only %d test references; expected at least 40", len(references))
	}

	var sources strings.Builder
	err = filepath.WalkDir(".", func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if path == ".git" || path == "dist" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, "_test.go") {
			return nil
		}
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		sources.Write(content)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, reference := range references {
		if !strings.Contains(sources.String(), "func "+reference+"(") {
			t.Errorf("feature matrix references missing test %s", reference)
		}
	}
}
