package namat

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestCustomDelimitersAndSmartQuotes(t *testing.T) {
	template := testDOCX(t, map[string][]byte{"word/document.xml": []byte(wordDocument(`<w:p><w:r><w:t>{# name == ‘نمط’ ? name : 'x' #}</w:t></w:r></w:p>`))})
	report, err := CreateReport(context.Background(), template, map[string]any{"name": "نمط"}, Options{OpenDelimiter: "{#", CloseDelimiter: "#}", FixSmartQuotes: true})
	if err != nil {
		t.Fatalf("CreateReport: %v", err)
	}
	if got := documentText(t, report); !strings.Contains(got, "نمط") {
		t.Fatalf("unexpected text: %q", got)
	}
}

func TestErrorHandlerAndRejectNullish(t *testing.T) {
	template := testDOCX(t, map[string][]byte{"word/document.xml": []byte(wordDocument(`<w:p><w:r><w:t>[[missing.value]]</w:t></w:r></w:p>`))})
	report, err := CreateReport(context.Background(), template, nil, Options{
		RejectNullish: true,
		ErrorHandler: func(command string, err error) (any, error) {
			return "غير متاح", nil
		},
	})
	if err != nil {
		t.Fatalf("CreateReport: %v", err)
	}
	if got := documentText(t, report); !strings.Contains(got, "غير متاح") {
		t.Fatalf("unexpected text: %q", got)
	}
}

func TestTypedNullishAndObjectErrors(t *testing.T) {
	nullTemplate := testDOCX(t, map[string][]byte{"word/document.xml": []byte(wordDocument(`<w:p><w:r><w:t>[[value]]</w:t></w:r></w:p>`))})
	_, err := CreateReport(context.Background(), nullTemplate, map[string]any{"value": nil}, Options{RejectNullish: true})
	if !errors.Is(err, ErrNullishResult) || !errors.Is(err, ErrCommandExecution) {
		t.Fatalf("error = %v, want nullish and execution categories", err)
	}
	_, err = CreateReport(context.Background(), nullTemplate, map[string]any{"value": map[string]any{"x": 1}}, Options{})
	if !errors.Is(err, ErrObjectResult) {
		t.Fatalf("error = %v, want object result category", err)
	}
}

func TestCollectValidationErrors(t *testing.T) {
	document := wordDocument(`<w:p><w:r><w:t>[[INS (]]</w:t></w:r></w:p><w:p><w:r><w:t>[[INS )]]</w:t></w:r></w:p>`)
	_, err := Compile(testDOCX(t, map[string][]byte{"word/document.xml": []byte(document)}), Options{CollectErrors: true})
	var multiple *MultiError
	if !errors.As(err, &multiple) || len(multiple.Errors) != 2 {
		t.Fatalf("error = %#v, want two validation errors", err)
	}
}

func TestNestedConditionsAndLoops(t *testing.T) {
	document := wordDocument(`
<w:p><w:r><w:t>[[FOR group IN groups]]</w:t></w:r></w:p>
<w:p><w:r><w:t>[[IF $group.visible]]</w:t></w:r></w:p>
<w:p><w:r><w:t>[[$group.name]]</w:t></w:r></w:p>
<w:p><w:r><w:t>[[FOR item IN $group.items]]</w:t></w:r></w:p>
<w:p><w:r><w:t>[[$item]]</w:t></w:r></w:p>
<w:p><w:r><w:t>[[END-FOR item]]</w:t></w:r></w:p>
<w:p><w:r><w:t>[[END-IF]]</w:t></w:r></w:p>
<w:p><w:r><w:t>[[END-FOR group]]</w:t></w:r></w:p>`)
	template := testDOCX(t, map[string][]byte{"word/document.xml": []byte(document)})
	data := map[string]any{"groups": []any{
		map[string]any{"visible": true, "name": "أ", "items": []string{"1", "2"}},
		map[string]any{"visible": false, "name": "ب", "items": []string{"3"}},
	}}
	report, err := CreateReport(context.Background(), template, data, Options{})
	if err != nil {
		t.Fatalf("CreateReport: %v", err)
	}
	text := documentText(t, report)
	if !strings.Contains(text, "أ") || !strings.Contains(text, "1") || !strings.Contains(text, "2") || strings.Contains(text, "ب") || strings.Contains(text, "3") {
		t.Fatalf("unexpected nested output: %q", text)
	}
}

func TestAggregateIterationLimit(t *testing.T) {
	document := wordDocument(`<w:p><w:r><w:t>[[FOR item IN items]]</w:t></w:r></w:p><w:p><w:r><w:t>[[$item]]</w:t></w:r></w:p><w:p><w:r><w:t>[[END-FOR]]</w:t></w:r></w:p>`)
	template := testDOCX(t, map[string][]byte{"word/document.xml": []byte(document)})
	_, err := CreateReport(context.Background(), template, map[string]any{"items": []int{1, 2, 3}}, Options{MaxIterations: 2})
	if err == nil || !strings.Contains(err.Error(), "maximum loop iterations") || !errors.Is(err, ErrSecurityLimit) {
		t.Fatalf("error = %v, want iteration limit", err)
	}
}

func TestCancelledContextStopsRender(t *testing.T) {
	template := testDOCX(t, map[string][]byte{"word/document.xml": []byte(wordDocument(`<w:p><w:r><w:t>[[name]]</w:t></w:r></w:p>`))})
	compiled, err := Compile(template, Options{})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = compiled.Render(ctx, map[string]any{"name": "x"})
	if err == nil || !strings.Contains(err.Error(), context.Canceled.Error()) {
		t.Fatalf("error = %v, want context cancellation", err)
	}
}

func TestCompiledTemplateRendersConcurrently(t *testing.T) {
	document := wordDocument(`<w:p><w:r><w:t>[[name]]</w:t></w:r></w:p><w:tbl><w:tr><w:tc><w:p><w:r><w:t>[[FOR item IN items]]</w:t></w:r></w:p></w:tc></w:tr><w:tr><w:tc><w:p><w:r><w:t>[[$item]]</w:t></w:r></w:p></w:tc></w:tr><w:tr><w:tc><w:p><w:r><w:t>[[END-FOR]]</w:t></w:r></w:p></w:tc></w:tr></w:tbl>`)
	compiled, err := Compile(testDOCX(t, map[string][]byte{"word/document.xml": []byte(document)}), Options{})
	if err != nil {
		t.Fatal(err)
	}
	const workers = 32
	var wg sync.WaitGroup
	errors := make(chan error, workers)
	for index := 0; index < workers; index++ {
		index := index
		wg.Add(1)
		go func() {
			defer wg.Done()
			name := fmt.Sprintf("doc-%d", index)
			report, renderErr := compiled.Render(context.Background(), map[string]any{"name": name, "items": []int{index, index + 1}})
			if renderErr != nil {
				errors <- renderErr
				return
			}
			pkg, packageErr := readPackage(report)
			if packageErr != nil {
				errors <- packageErr
				return
			}
			if !bytes.Contains(pkg.Parts["word/document.xml"].Data, []byte(name)) {
				errors <- fmt.Errorf("render %d contains another render's data", index)
			}
		}()
	}
	wg.Wait()
	close(errors)
	for err := range errors {
		t.Error(err)
	}
}

func TestReaderWriterAPIsAndLimits(t *testing.T) {
	template := testDOCX(t, map[string][]byte{"word/document.xml": []byte(wordDocument(`<w:p><w:r><w:t>[[name]]</w:t></w:r></w:p>`))})
	compiled, err := CompileReader(bytes.NewReader(template), Options{})
	if err != nil {
		t.Fatalf("CompileReader: %v", err)
	}
	var output bytes.Buffer
	if err := compiled.RenderTo(context.Background(), &output, map[string]any{"name": "نمط"}); err != nil {
		t.Fatalf("RenderTo: %v", err)
	}
	if _, err := readPackage(output.Bytes()); err != nil {
		t.Fatalf("rendered writer output is not DOCX: %v", err)
	}
	if _, err := CompileReader(bytes.NewReader(template), Options{MaxTemplateBytes: int64(len(template) - 1)}); err == nil || !strings.Contains(err.Error(), "MaxTemplateBytes") || !errors.Is(err, ErrSecurityLimit) {
		t.Fatalf("error = %v, want template size limit", err)
	}
	limited, err := Compile(template, Options{MaxOutputBytes: 1})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := limited.Render(context.Background(), map[string]any{"name": "x"}); err == nil || !errors.Is(err, ErrSecurityLimit) {
		t.Fatalf("error = %v, want output security limit", err)
	}
	if _, err := compiled.Render(context.Background(), map[string]any{"name": "x"}); err != nil {
		t.Fatal(err)
	}
}

func TestTimeoutOption(t *testing.T) {
	template := testDOCX(t, map[string][]byte{"word/document.xml": []byte(wordDocument(`<w:p><w:r><w:t>[[slow()]]</w:t></w:r></w:p>`))})
	_, err := CreateReport(context.Background(), template, nil, Options{
		Timeout: time.Nanosecond,
		Functions: map[string]Function{"slow": func(args ...any) (any, error) {
			time.Sleep(time.Millisecond)
			return "done", nil
		}},
	})
	if err == nil {
		t.Fatal("expected timeout")
	}
}
