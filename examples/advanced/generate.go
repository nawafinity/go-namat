//go:build ignore

package main

import (
	"archive/zip"
	"bytes"
	"fmt"
	"os"
	"sort"
	"time"
)

func main() {
	parts := map[string]string{
		"[Content_Types].xml": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="xml" ContentType="application/xml"/>
  <Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>
  <Override PartName="/word/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.styles+xml"/>
</Types>`,
		"_rels/.rels": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/></Relationships>`,
		"word/_rels/document.xml.rels": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/>
</Relationships>`,
		"word/styles.xml": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:styles xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:style w:type="paragraph" w:default="1" w:styleId="Normal"><w:name w:val="Normal"/><w:rPr><w:sz w:val="21"/></w:rPr></w:style>
  <w:style w:type="table" w:styleId="TableGrid"><w:name w:val="Table Grid"/></w:style>
</w:styles>`,
		"word/document.xml": documentXML(),
	}
	var names []string
	for name := range parts {
		names = append(names, name)
	}
	sort.Strings(names)
	var output bytes.Buffer
	writer := zip.NewWriter(&output)
	for _, name := range names {
		header := &zip.FileHeader{Name: name, Method: zip.Deflate}
		header.SetModTime(time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC))
		entry, err := writer.CreateHeader(header)
		if err != nil {
			panic(err)
		}
		if _, err := entry.Write([]byte(parts[name])); err != nil {
			panic(err)
		}
	}
	if err := writer.Close(); err != nil {
		panic(err)
	}
	if err := os.WriteFile("template.docx", output.Bytes(), 0o644); err != nil {
		panic(err)
	}
	fmt.Printf("wrote template.docx (%d bytes)\n", output.Len())
}

func documentXML() string {
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><w:body>
  <w:p><w:pPr><w:bidi/><w:jc w:val="center"/><w:spacing w:after="220"/></w:pPr><w:r><w:rPr><w:rtl/><w:b/><w:sz w:val="36"/><w:color w:val="000000"/></w:rPr><w:t>[[title]]</w:t></w:r></w:p>
  <w:p><w:pPr><w:bidi/><w:jc w:val="center"/></w:pPr><w:r><w:rPr><w:rtl/><w:color w:val="657786"/></w:rPr><w:t>نموذج يختبر الحلقات المتداخلة والاستدعاءات الديناميكية</w:t></w:r></w:p>
  <w:p><w:pPr><w:bidi/><w:jc w:val="center"/><w:spacing w:after="140"/></w:pPr><w:r><w:rPr><w:rtl/><w:color w:val="666666"/></w:rPr><w:t>تاريخ التقرير: [[generated_at]]</w:t></w:r></w:p>
  <w:p><w:r><w:t>[[#each departments as department]]</w:t></w:r></w:p>
  <w:p><w:r><w:t>[[#let departmentBudget = sumBudgets(department.projects)]]</w:t></w:r></w:p>
  <w:p><w:pPr><w:bidi/><w:jc w:val="right"/><w:keepNext/><w:spacing w:before="240" w:after="100"/></w:pPr><w:r><w:rPr><w:rtl/><w:b/><w:sz w:val="28"/><w:color w:val="1F6F8B"/></w:rPr><w:t>[[department.name | upper]] — [[department.projects | count]] مشاريع</w:t></w:r></w:p>
  <w:p><w:pPr><w:bidi/><w:jc w:val="right"/><w:keepNext/></w:pPr><w:r><w:rPr><w:rtl/></w:rPr><w:t>المالك: [[department.owner]]</w:t></w:r></w:p>
  <w:p><w:r><w:t>[[#if department.active]]</w:t></w:r></w:p>
  <w:p><w:pPr><w:bidi/><w:jc w:val="right"/><w:keepNext/></w:pPr><w:r><w:rPr><w:rtl/><w:color w:val="178A55"/><w:b/></w:rPr><w:t>القسم نشط ويستقبل أعمالًا جديدة</w:t></w:r></w:p>
  <w:p><w:r><w:t>[[#else]]</w:t></w:r></w:p>
  <w:p><w:pPr><w:bidi/><w:jc w:val="right"/><w:keepNext/></w:pPr><w:r><w:rPr><w:rtl/><w:color w:val="B23A48"/><w:b/></w:rPr><w:t>القسم متوقف مؤقتًا للمراجعة</w:t></w:r></w:p>
  <w:p><w:r><w:t>[[/if]]</w:t></w:r></w:p>
  <w:p><w:pPr><w:bidi/><w:jc w:val="right"/><w:keepNext/><w:spacing w:after="100"/></w:pPr><w:r><w:rPr><w:rtl/><w:b/></w:rPr><w:t>إجمالي ميزانية القسم: [[money(departmentBudget, currency)]]</w:t></w:r></w:p>
  <w:tbl><w:tblPr><w:tblW w:w="14000" w:type="dxa"/><w:jc w:val="center"/><w:tblLayout w:type="fixed"/><w:tblBorders><w:top w:val="single" w:sz="8" w:color="7D8C99"/><w:left w:val="single" w:sz="8" w:color="7D8C99"/><w:bottom w:val="single" w:sz="8" w:color="7D8C99"/><w:right w:val="single" w:sz="8" w:color="7D8C99"/><w:insideH w:val="single" w:sz="6" w:color="BBC4CC"/><w:insideV w:val="single" w:sz="6" w:color="BBC4CC"/></w:tblBorders><w:tblCellMar><w:top w:w="90" w:type="dxa"/><w:left w:w="100" w:type="dxa"/><w:bottom w:w="90" w:type="dxa"/><w:right w:w="100" w:type="dxa"/></w:tblCellMar></w:tblPr>
    <w:tblGrid><w:gridCol w:w="2400"/><w:gridCol w:w="1600"/><w:gridCol w:w="1800"/><w:gridCol w:w="2500"/><w:gridCol w:w="5700"/></w:tblGrid>
    <w:tr><w:trPr><w:tblHeader/></w:trPr>
      <w:tc><w:tcPr><w:shd w:fill="173F5F"/></w:tcPr><w:p><w:pPr><w:bidi/><w:jc w:val="center"/></w:pPr><w:r><w:rPr><w:rtl/><w:b/><w:color w:val="FFFFFF"/></w:rPr><w:t>المشروع</w:t></w:r></w:p></w:tc>
      <w:tc><w:tcPr><w:shd w:fill="173F5F"/></w:tcPr><w:p><w:pPr><w:bidi/><w:jc w:val="center"/></w:pPr><w:r><w:rPr><w:rtl/><w:b/><w:color w:val="FFFFFF"/></w:rPr><w:t>الحالة</w:t></w:r></w:p></w:tc>
      <w:tc><w:tcPr><w:shd w:fill="173F5F"/></w:tcPr><w:p><w:pPr><w:bidi/><w:jc w:val="center"/></w:pPr><w:r><w:rPr><w:rtl/><w:b/><w:color w:val="FFFFFF"/></w:rPr><w:t>الميزانية</w:t></w:r></w:p></w:tc>
      <w:tc><w:tcPr><w:shd w:fill="173F5F"/></w:tcPr><w:p><w:pPr><w:bidi/><w:jc w:val="center"/></w:pPr><w:r><w:rPr><w:rtl/><w:b/><w:color w:val="FFFFFF"/></w:rPr><w:t>التقدم</w:t></w:r></w:p></w:tc>
      <w:tc><w:tcPr><w:shd w:fill="173F5F"/></w:tcPr><w:p><w:pPr><w:bidi/><w:jc w:val="center"/></w:pPr><w:r><w:rPr><w:rtl/><w:b/><w:color w:val="FFFFFF"/></w:rPr><w:t>المهام المتداخلة</w:t></w:r></w:p></w:tc>
    </w:tr>
    <w:tr><w:tc><w:tcPr><w:gridSpan w:val="5"/></w:tcPr><w:p><w:r><w:t>[[#each department.projects as project]]</w:t></w:r></w:p></w:tc></w:tr>
    <w:tr><w:trPr><w:cantSplit/></w:trPr>
      <w:tc><w:p><w:r><w:t>[[#let projectHours = sumHours(project.tasks)]]</w:t></w:r></w:p><w:p><w:r><w:t>[[@link projectLink(project)]]</w:t></w:r></w:p><w:p><w:pPr><w:bidi/><w:jc w:val="right"/></w:pPr><w:r><w:rPr><w:rtl/><w:color w:val="657786"/></w:rPr><w:t>[[projectHours | hours]] إجمالي</w:t></w:r></w:p></w:tc>
      <w:tc><w:p><w:pPr><w:bidi/><w:jc w:val="center"/></w:pPr><w:r><w:rPr><w:rtl/></w:rPr><w:t>[[project.status | statusLabel]]</w:t></w:r></w:p></w:tc>
      <w:tc><w:p><w:pPr><w:bidi/><w:jc w:val="center"/></w:pPr><w:r><w:t>[[money(project.budget, currency)]]</w:t></w:r></w:p></w:tc>
      <w:tc><w:p><w:pPr><w:jc w:val="center"/></w:pPr><w:r><w:t>[[@image sparkline(project.progress, project.name)]]</w:t></w:r></w:p></w:tc>
      <w:tc><w:p><w:pPr><w:bidi/><w:jc w:val="right"/></w:pPr><w:r><w:rPr><w:rtl/><w:b/></w:rPr><w:t>[[project.tasks | count]] مهام</w:t></w:r></w:p>
        <w:tbl><w:tblPr><w:tblW w:w="5400" w:type="dxa"/><w:tblLayout w:type="fixed"/><w:tblBorders><w:top w:val="single" w:sz="4" w:color="AAB4BC"/><w:left w:val="single" w:sz="4" w:color="AAB4BC"/><w:bottom w:val="single" w:sz="4" w:color="AAB4BC"/><w:right w:val="single" w:sz="4" w:color="AAB4BC"/><w:insideH w:val="single" w:sz="4" w:color="D5DBE0"/><w:insideV w:val="single" w:sz="4" w:color="D5DBE0"/></w:tblBorders><w:tblCellMar><w:top w:w="45" w:type="dxa"/><w:left w:w="60" w:type="dxa"/><w:bottom w:w="45" w:type="dxa"/><w:right w:w="60" w:type="dxa"/></w:tblCellMar></w:tblPr><w:tblGrid><w:gridCol w:w="500"/><w:gridCol w:w="2500"/><w:gridCol w:w="1300"/><w:gridCol w:w="1100"/></w:tblGrid>
          <w:tr><w:tc><w:p><w:r><w:t>#</w:t></w:r></w:p></w:tc><w:tc><w:p><w:r><w:t>المهمة</w:t></w:r></w:p></w:tc><w:tc><w:p><w:r><w:t>الحالة</w:t></w:r></w:p></w:tc><w:tc><w:p><w:r><w:t>الجهد</w:t></w:r></w:p></w:tc></w:tr>
          <w:tr><w:tc><w:tcPr><w:gridSpan w:val="4"/></w:tcPr><w:p><w:r><w:t>[[#each project.tasks as task]]</w:t></w:r></w:p></w:tc></w:tr>
          <w:tr><w:tc><w:p><w:r><w:t>[[loop.number]]</w:t></w:r></w:p></w:tc><w:tc><w:p><w:pPr><w:bidi/><w:jc w:val="right"/></w:pPr><w:r><w:rPr><w:rtl/></w:rPr><w:t>[[task.name]]</w:t></w:r></w:p></w:tc><w:tc><w:p><w:pPr><w:bidi/><w:jc w:val="right"/></w:pPr><w:r><w:rPr><w:rtl/></w:rPr><w:t>[[task.state | taskBadge]]</w:t></w:r></w:p></w:tc><w:tc><w:p><w:pPr><w:bidi/><w:jc w:val="right"/></w:pPr><w:r><w:rPr><w:rtl/></w:rPr><w:t>[[task.hours | hours]]</w:t></w:r></w:p></w:tc></w:tr>
          <w:tr><w:tc><w:tcPr><w:gridSpan w:val="4"/></w:tcPr><w:p><w:r><w:t>[[/each]]</w:t></w:r></w:p></w:tc></w:tr>
        </w:tbl><w:p/>
      </w:tc>
    </w:tr>
    <w:tr><w:tc><w:tcPr><w:gridSpan w:val="5"/></w:tcPr><w:p><w:r><w:t>[[/each]]</w:t></w:r></w:p></w:tc></w:tr>
  </w:tbl>
  <w:p><w:r><w:t>[[/each]]</w:t></w:r></w:p>
  <w:sectPr><w:pgSz w:w="15840" w:h="12240" w:orient="landscape"/><w:pgMar w:top="850" w:right="700" w:bottom="850" w:left="700"/></w:sectPr>
</w:body></w:document>`
}
