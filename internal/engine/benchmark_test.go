package engine

import (
	"context"
	"fmt"
	"testing"
)

func benchmarkFixture(b testing.TB) ([]byte, map[string]any) {
	document := wordDocument(`<w:p><w:r><w:t>[[title]]</w:t></w:r></w:p><w:tbl><w:tr><w:tc><w:p><w:r><w:t>[[FOR row IN rows]]</w:t></w:r></w:p></w:tc></w:tr><w:tr><w:tc><w:p><w:r><w:t>[[$idx + 1]] [[$row.name]] [[$row.value]]</w:t></w:r></w:p></w:tc></w:tr><w:tr><w:tc><w:p><w:r><w:t>[[END-FOR]]</w:t></w:r></w:p></w:tc></w:tr></w:tbl>`)
	rows := make([]map[string]any, 100)
	for index := range rows {
		rows[index] = map[string]any{"name": fmt.Sprintf("row-%d", index), "value": index * 10}
	}
	return testDOCX(b, map[string][]byte{"word/document.xml": []byte(document)}), map[string]any{"title": "Performance", "rows": rows}
}

func BenchmarkCompile(b *testing.B) {
	template, _ := benchmarkFixture(b)
	b.ReportAllocs()
	b.SetBytes(int64(len(template)))
	for index := 0; index < b.N; index++ {
		if _, err := Compile(template, Options{}); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRenderCompiled100Rows(b *testing.B) {
	template, data := benchmarkFixture(b)
	compiled, err := Compile(template, Options{})
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	for index := 0; index < b.N; index++ {
		if _, err := compiled.Render(context.Background(), data); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCreateReport100Rows(b *testing.B) {
	template, data := benchmarkFixture(b)
	b.ReportAllocs()
	for index := 0; index < b.N; index++ {
		if _, err := CreateReport(context.Background(), template, data, Options{}); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParallelRender100Rows(b *testing.B) {
	template, data := benchmarkFixture(b)
	compiled, err := Compile(template, Options{})
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if _, err := compiled.Render(context.Background(), data); err != nil {
				b.Fatal(err)
			}
		}
	})
}
