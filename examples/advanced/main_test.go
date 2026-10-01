package main

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"image"
	"io"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/nawafinity/go-namat"
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

func TestAdvancedMainAndRunFailureEdges(t *testing.T) {
	originalArgs, originalExit := os.Args, exit
	defer func() {
		os.Args = originalArgs
		exit = originalExit
	}()
	output := filepath.Join(t.TempDir(), "main.docx")
	os.Args = []string{"advanced", "--template", "template.docx", "--data", "data.json", "--out", output}
	code := -1
	exit = func(value int) { code = value }
	main()
	if code != -1 {
		t.Fatalf("successful main exit code = %d", code)
	}
	os.Args = []string{"advanced", "--repeat", "0"}
	main()
	if code != 1 {
		t.Fatalf("failing main exit code = %d", code)
	}

	directory := t.TempDir()
	validTemplate, err := os.ReadFile("template.docx")
	if err != nil {
		t.Fatal(err)
	}
	validTemplatePath := filepath.Join(directory, "template.docx")
	validDataPath := filepath.Join(directory, "data.json")
	badDataPath := filepath.Join(directory, "bad.json")
	arrayDataPath := filepath.Join(directory, "array.json")
	badTemplatePath := filepath.Join(directory, "bad.docx")
	if err := os.WriteFile(validTemplatePath, validTemplate, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(validDataPath, []byte(`{"departments":[]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(badDataPath, []byte(`{`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(arrayDataPath, []byte(`[]`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(badTemplatePath, []byte("bad"), 0o600); err != nil {
		t.Fatal(err)
	}

	tests := [][]string{
		{"--unknown"},
		{"unexpected"},
		{"--template", filepath.Join(directory, "missing.docx")},
		{"--template", validTemplatePath, "--data", filepath.Join(directory, "missing.json")},
		{"--template", validTemplatePath, "--data", badDataPath},
		{"--template", validTemplatePath, "--data", arrayDataPath, "--repeat", "2"},
		{"--template", badTemplatePath, "--data", validDataPath},
		{"--template", validTemplatePath, "--data", "data.json", "--out", directory},
	}
	for _, args := range tests {
		if err := run(args); err == nil {
			t.Fatalf("run(%v) succeeded", args)
		}
	}
}

func TestRepeatAndCloneValidationEdges(t *testing.T) {
	original := map[string]any{"departments": []any{map[string]any{"name": "D", "projects": []any{map[string]any{"name": "P", "url": "https://example.test", "tasks": []any{map[string]any{"name": "T"}}}}}}}
	if got, err := repeatReportData(original, 1); err != nil || !reflect.DeepEqual(got, original) {
		t.Fatalf("repeat one = %#v, %v", got, err)
	}
	for _, input := range []any{
		1,
		map[string]any{"departments": 1},
		map[string]any{"departments": []any{1}},
		map[string]any{"departments": []any{map[string]any{"projects": 1}}},
		map[string]any{"departments": []any{map[string]any{"projects": []any{1}}}},
		map[string]any{"departments": []any{map[string]any{"projects": []any{map[string]any{"tasks": 1}}}}},
		map[string]any{"departments": []any{map[string]any{"projects": []any{map[string]any{"tasks": []any{1}}}}}},
	} {
		if _, err := repeatReportData(input, 2); err == nil {
			t.Fatalf("repeatReportData(%#v) succeeded", input)
		}
	}
	clone := cloneJSONValue(map[string]any{"items": []any{map[string]any{"value": 1}}, "scalar": "x"}).(map[string]any)
	clone["items"].([]any)[0].(map[string]any)["value"] = 2
	if clone["scalar"] != "x" {
		t.Fatalf("clone scalar = %#v", clone["scalar"])
	}
}

func TestAdvancedFunctionValidationEdges(t *testing.T) {
	functions := advancedFunctions()
	call := func(name string, args ...any) (any, error) {
		return functions[name].Call(context.Background(), args...)
	}
	for _, name := range []string{"upper", "count", "sumBudgets", "sumHours", "statusLabel", "taskBadge", "hours", "projectLink"} {
		if _, err := call(name); err == nil {
			t.Fatalf("%s accepted no arguments", name)
		}
	}
	if _, err := call("money", 1); err == nil {
		t.Fatal("money accepted one argument")
	}
	if _, err := call("count", 1); err == nil {
		t.Fatal("count accepted a scalar")
	}
	if got, err := call("count", "abc"); err != nil || got != 3 {
		t.Fatalf("count string = %#v, %v", got, err)
	}
	if _, err := call("money", "bad", "SAR"); err == nil {
		t.Fatal("money accepted non-numeric amount")
	}
	if got, err := call("statusLabel", "unknown"); err != nil || got != "unknown" {
		t.Fatalf("unknown status = %#v, %v", got, err)
	}
	if got, err := call("taskBadge", "unknown"); err != nil || got != "unknown" {
		t.Fatalf("unknown task badge = %#v, %v", got, err)
	}
	if _, err := call("hours", "bad"); err == nil {
		t.Fatal("hours accepted non-numeric value")
	}
	if _, err := call("projectLink", "bad"); err == nil {
		t.Fatal("projectLink accepted non-object")
	}

	if _, err := sumField("bad", "value"); err == nil {
		t.Fatal("sumField accepted non-list")
	}
	if _, err := sumField([]any{1}, "value"); err == nil {
		t.Fatal("sumField accepted non-object item")
	}
	if _, err := sumField([]any{map[string]any{"value": "bad"}}, "value"); err == nil {
		t.Fatal("sumField accepted non-numeric field")
	}

	if _, err := sparkline(); err == nil {
		t.Fatal("sparkline accepted no arguments")
	}
	if _, err := sparkline([]any{}, "empty"); err == nil {
		t.Fatal("sparkline accepted empty values")
	}
	if _, err := sparkline([]any{"bad"}, "bad"); err == nil {
		t.Fatal("sparkline accepted non-numeric value")
	}
	if _, err := sparkline([]any{int64(-1), int64(101)}, "clamped"); err != nil {
		t.Fatalf("sparkline clamping: %v", err)
	}
	originalEncode := encodePNG
	encodePNG = func(io.Writer, image.Image) error { return errors.New("encode") }
	t.Cleanup(func() { encodePNG = originalEncode })
	if _, err := sparkline([]any{int64(1)}, "encode"); err == nil {
		t.Fatal("sparkline encode failure was not propagated")
	}
}

func TestAdvancedNumericConversions(t *testing.T) {
	decimal, _ := namat.ParseDecimal("1.5")
	values := []any{decimal, float64(1), float32(1), int(1), int64(1), int32(1), uint(1), uint64(1), uint32(1)}
	for _, value := range values {
		if got, ok := numeric(value); !ok || got <= 0 {
			t.Fatalf("numeric(%T) = %v, %v", value, got, ok)
		}
		if got, ok := decimalNumber(value); !ok || got.String() == "" {
			t.Fatalf("decimalNumber(%T) = %v, %v", value, got, ok)
		}
	}
	if _, ok := numeric("bad"); ok {
		t.Fatal("numeric accepted string")
	}
	for _, value := range []any{"bad", math.NaN(), math.Inf(1)} {
		if _, ok := decimalNumber(value); ok {
			t.Fatalf("decimalNumber(%v) succeeded", value)
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
