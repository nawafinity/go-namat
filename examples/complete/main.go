// Command complete creates a self-contained Arabic DOCX report demonstrating
// text, conditions, table-row loops, hyperlinks, and images.
package main

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"

	"github.com/nawafinity/go-namat"
)

func main() {
	output := "namat-demo.docx"
	if len(os.Args) > 1 {
		output = os.Args[1]
	}
	template := templateDOCX()
	report, err := namat.CreateReport(context.Background(), template, map[string]any{
		"title":       "تقرير جودة مكتبة نمط",
		"description": "تقرير تجريبي مولد بالكامل بمحرك Go أصلي",
		"url":         "https://github.com/nawafinity/go-namat",
		"ready":       true,
		"rows": []map[string]any{
			{"name": "تحليل القوالب", "status": "مكتمل"},
			{"name": "الجداول والشروط", "status": "مكتمل"},
			{"name": "الصور والروابط", "status": "مكتمل"},
		},
	}, namat.Options{Functions: map[string]namat.Function{
		"banner": func(args ...any) (any, error) {
			return namat.Image{Data: bannerPNG(), Extension: "png", Width: 12, Height: 4.8, Alt: "تدرج لوني تجريبي"}, nil
		},
	}})
	check(err)
	check(os.WriteFile(output, report, 0o644))
	fmt.Println(output)
}

func templateDOCX() []byte {
	parts := map[string]string{
		"[Content_Types].xml": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/><Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/></Types>`,
		"_rels/.rels":         `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/></Relationships>`,
		"word/document.xml": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body>
<w:p><w:pPr><w:bidi/><w:jc w:val="center"/><w:spacing w:after="160"/></w:pPr><w:r><w:rPr><w:rtl/><w:b/><w:sz w:val="36"/><w:color w:val="16324F"/></w:rPr><w:t>[[title]]</w:t></w:r></w:p>
<w:p><w:pPr><w:bidi/><w:jc w:val="center"/><w:spacing w:after="260"/></w:pPr><w:r><w:rPr><w:rtl/><w:sz w:val="23"/><w:color w:val="526579"/></w:rPr><w:t>[[description]]</w:t></w:r></w:p>
<w:p><w:pPr><w:jc w:val="center"/><w:spacing w:after="220"/></w:pPr><w:r><w:t>[[IMAGE banner()]]</w:t></w:r></w:p>
<w:p><w:pPr><w:bidi/><w:jc w:val="right"/><w:spacing w:before="120" w:after="120"/></w:pPr><w:r><w:rPr><w:rtl/><w:b/><w:sz w:val="27"/></w:rPr><w:t>نتائج الاختبار</w:t></w:r></w:p>
<w:tbl><w:tblPr><w:bidiVisual/><w:tblW w:w="9000" w:type="dxa"/><w:tblBorders><w:top w:val="single" w:sz="6" w:color="D9D9D9"/><w:left w:val="single" w:sz="6" w:color="D9D9D9"/><w:bottom w:val="single" w:sz="6" w:color="D9D9D9"/><w:right w:val="single" w:sz="6" w:color="D9D9D9"/><w:insideH w:val="single" w:sz="6" w:color="D9D9D9"/><w:insideV w:val="single" w:sz="6" w:color="D9D9D9"/></w:tblBorders><w:tblCellMar><w:top w:w="100" w:type="dxa"/><w:left w:w="120" w:type="dxa"/><w:bottom w:w="100" w:type="dxa"/><w:right w:w="120" w:type="dxa"/></w:tblCellMar></w:tblPr>
<w:tr><w:trPr><w:tblHeader/></w:trPr><w:tc><w:tcPr><w:shd w:fill="16324F"/></w:tcPr><w:p><w:pPr><w:bidi/><w:jc w:val="center"/></w:pPr><w:r><w:rPr><w:rtl/><w:b/><w:color w:val="FFFFFF"/></w:rPr><w:t>الحالة</w:t></w:r></w:p></w:tc><w:tc><w:tcPr><w:shd w:fill="16324F"/></w:tcPr><w:p><w:pPr><w:bidi/><w:jc w:val="center"/></w:pPr><w:r><w:rPr><w:rtl/><w:b/><w:color w:val="FFFFFF"/></w:rPr><w:t>الاختبار</w:t></w:r></w:p></w:tc><w:tc><w:tcPr><w:shd w:fill="16324F"/></w:tcPr><w:p><w:pPr><w:bidi/><w:jc w:val="center"/></w:pPr><w:r><w:rPr><w:rtl/><w:b/><w:color w:val="FFFFFF"/></w:rPr><w:t>م</w:t></w:r></w:p></w:tc></w:tr>
<w:tr><w:tc><w:p><w:r><w:t>[[FOR row IN rows]]</w:t></w:r></w:p></w:tc></w:tr>
<w:tr><w:tc><w:p><w:pPr><w:bidi/><w:jc w:val="center"/></w:pPr><w:r><w:rPr><w:rtl/></w:rPr><w:t>[[$row.status]]</w:t></w:r></w:p></w:tc><w:tc><w:p><w:pPr><w:bidi/><w:jc w:val="right"/></w:pPr><w:r><w:rPr><w:rtl/></w:rPr><w:t>[[$row.name]]</w:t></w:r></w:p></w:tc><w:tc><w:p><w:pPr><w:bidi/><w:jc w:val="center"/></w:pPr><w:r><w:rPr><w:rtl/></w:rPr><w:t>[[$idx + 1]]</w:t></w:r></w:p></w:tc></w:tr>
<w:tr><w:tc><w:p><w:r><w:t>[[END-FOR row]]</w:t></w:r></w:p></w:tc></w:tr></w:tbl>
<w:p><w:r><w:t>[[IF ready]]</w:t></w:r></w:p>
<w:p><w:pPr><w:bidi/><w:jc w:val="right"/><w:spacing w:before="220"/></w:pPr><w:r><w:rPr><w:rtl/></w:rPr><w:t>الحالة العامة: جاهز للاختبار</w:t></w:r></w:p>
<w:p><w:r><w:t>[[ELSE]]</w:t></w:r></w:p>
<w:p><w:pPr><w:bidi/><w:jc w:val="right"/><w:spacing w:before="220"/></w:pPr><w:r><w:rPr><w:rtl/></w:rPr><w:t>الحالة العامة: قيد العمل</w:t></w:r></w:p>
<w:p><w:r><w:t>[[END-IF]]</w:t></w:r></w:p>
<w:p><w:pPr><w:bidi/><w:jc w:val="right"/></w:pPr><w:r><w:rPr><w:rtl/></w:rPr><w:t xml:space="preserve">المستودع: </w:t></w:r><w:r><w:t>[[LINK ({ url: url, label: 'go-namat' })]]</w:t></w:r></w:p>
<w:sectPr><w:pgSz w:w="12240" w:h="15840"/><w:pgMar w:top="1080" w:right="1080" w:bottom="1080" w:left="1080" w:header="720" w:footer="720" w:gutter="0"/></w:sectPr></w:body></w:document>`,
	}
	var out bytes.Buffer
	writer := zip.NewWriter(&out)
	for _, name := range []string{"[Content_Types].xml", "_rels/.rels", "word/document.xml"} {
		stream, _ := writer.Create(name)
		_, _ = stream.Write([]byte(parts[name]))
	}
	_ = writer.Close()
	return out.Bytes()
}

func bannerPNG() []byte {
	canvas := image.NewRGBA(image.Rect(0, 0, 900, 360))
	for y := 0; y < 360; y++ {
		for x := 0; x < 900; x++ {
			canvas.SetRGBA(x, y, color.RGBA{R: uint8(18 + x*25/900), G: uint8(48 + y*55/360), B: uint8(78 + x*45/900), A: 255})
		}
	}
	var out bytes.Buffer
	_ = png.Encode(&out, canvas)
	return out.Bytes()
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}
