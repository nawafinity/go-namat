//go:build ignore

// Command generate recreates the public complex-tables.docx compatibility
// fixture. The fixture is synthetic and dedicated to the public domain under
// CC0-1.0; it contains no customer or production data.
package main

import (
	"archive/zip"
	"bytes"
	"fmt"
	"os"
	"sort"
	"time"
)

const wordNamespace = "http://schemas.openxmlformats.org/wordprocessingml/2006/main"

func main() {
	parts := map[string]string{
		"[Content_Types].xml": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="xml" ContentType="application/xml"/>
  <Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>
  <Override PartName="/word/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.styles+xml"/>
  <Override PartName="/docProps/core.xml" ContentType="application/vnd.openxmlformats-package.core-properties+xml"/>
  <Override PartName="/docProps/app.xml" ContentType="application/vnd.openxmlformats-officedocument.extended-properties+xml"/>
</Types>`,
		"_rels/.rels": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>
  <Relationship Id="rId2" Type="http://schemas.openxmlformats.org/package/2006/relationships/metadata/core-properties" Target="docProps/core.xml"/>
  <Relationship Id="rId3" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/extended-properties" Target="docProps/app.xml"/>
</Relationships>`,
		"docProps/core.xml": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<cp:coreProperties xmlns:cp="http://schemas.openxmlformats.org/package/2006/metadata/core-properties" xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:dcterms="http://purl.org/dc/terms/" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
  <dc:title>Namat public complex table fixture</dc:title>
  <dc:creator>Namat contributors</dc:creator>
  <dcterms:created xsi:type="dcterms:W3CDTF">2026-09-30T00:00:00Z</dcterms:created>
</cp:coreProperties>`,
		"docProps/app.xml": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Properties xmlns="http://schemas.openxmlformats.org/officeDocument/2006/extended-properties" xmlns:vt="http://schemas.openxmlformats.org/officeDocument/2006/docPropsVTypes"><Application>Namat fixture generator</Application></Properties>`,
		"word/_rels/document.xml.rels": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/>
</Relationships>`,
		"word/styles.xml": `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:styles xmlns:w="` + wordNamespace + `">
  <w:style w:type="paragraph" w:default="1" w:styleId="Normal"><w:name w:val="Normal"/></w:style>
  <w:style w:type="table" w:styleId="TableGrid"><w:name w:val="Table Grid"/><w:tblPr><w:tblBorders><w:top w:val="single" w:sz="4" w:color="808080"/><w:left w:val="single" w:sz="4" w:color="808080"/><w:bottom w:val="single" w:sz="4" w:color="808080"/><w:right w:val="single" w:sz="4" w:color="808080"/><w:insideH w:val="single" w:sz="4" w:color="808080"/><w:insideV w:val="single" w:sz="4" w:color="808080"/></w:tblBorders></w:tblPr></w:style>
</w:styles>`,
		"customXml/item1.xml": `<?xml version="1.0"?><fixture preservation="byte-for-byte">untouched</fixture>`,
		"word/document.xml":   documentXML(),
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
	if err := os.WriteFile("complex-tables.docx", output.Bytes(), 0o644); err != nil {
		panic(err)
	}
	fmt.Printf("wrote complex-tables.docx (%d bytes)\n", output.Len())
}

func documentXML() string {
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="` + wordNamespace + `"><w:body>
  <w:p><w:pPr><w:jc w:val="center"/></w:pPr><w:r><w:rPr><w:b/><w:sz w:val="32"/></w:rPr><w:t>[[title]]</w:t></w:r></w:p>
  <w:tbl>
    <w:tblPr><w:tblStyle w:val="TableGrid"/><w:tblW w:w="7200" w:type="dxa"/><w:tblBorders><w:top w:val="single" w:sz="8" w:color="808080"/><w:left w:val="single" w:sz="8" w:color="808080"/><w:bottom w:val="single" w:sz="8" w:color="808080"/><w:right w:val="single" w:sz="8" w:color="808080"/><w:insideH w:val="single" w:sz="6" w:color="B0B0B0"/><w:insideV w:val="single" w:sz="6" w:color="B0B0B0"/></w:tblBorders><w:tblCellMar><w:top w:w="80" w:type="dxa"/><w:left w:w="100" w:type="dxa"/><w:bottom w:w="80" w:type="dxa"/><w:right w:w="100" w:type="dxa"/></w:tblCellMar></w:tblPr>
    <w:tblGrid><w:gridCol w:w="2400"/><w:gridCol w:w="2400"/><w:gridCol w:w="2400"/></w:tblGrid>
    <w:tr><w:trPr><w:tblHeader/></w:trPr><w:tc><w:tcPr><w:gridSpan w:val="3"/><w:shd w:fill="D9EAF7"/></w:tcPr><w:p><w:r><w:rPr><w:b/></w:rPr><w:t>Records</w:t></w:r></w:p></w:tc></w:tr>
    <w:tr><w:tc><w:tcPr><w:gridSpan w:val="3"/></w:tcPr><w:p><w:r><w:t>[[#each records as record]]</w:t></w:r></w:p></w:tc></w:tr>
    <w:tr>
      <w:tc><w:tcPr><w:vMerge w:val="restart"/><w:shd w:fill="FFF2CC"/></w:tcPr><w:p><w:r><w:rPr><w:b/></w:rPr><w:t>[[record.name]]</w:t></w:r></w:p></w:tc>
      <w:tc><w:tcPr><w:gridSpan w:val="2"/></w:tcPr><w:p><w:r><w:t>[[record.value]]</w:t></w:r></w:p>
        <w:tbl><w:tblPr><w:tblStyle w:val="TableGrid"/><w:tblBorders><w:top w:val="single" w:sz="4" w:color="A0A0A0"/><w:left w:val="single" w:sz="4" w:color="A0A0A0"/><w:bottom w:val="single" w:sz="4" w:color="A0A0A0"/><w:right w:val="single" w:sz="4" w:color="A0A0A0"/><w:insideH w:val="single" w:sz="4" w:color="C0C0C0"/></w:tblBorders></w:tblPr><w:tblGrid><w:gridCol w:w="4800"/></w:tblGrid>
          <w:tr><w:tc><w:p><w:r><w:t>[[#each record.details as detail]]</w:t></w:r></w:p></w:tc></w:tr>
          <w:tr><w:tc><w:p><w:r><w:t>[[loop.index + 1]]. [[detail]]</w:t></w:r></w:p></w:tc></w:tr>
          <w:tr><w:tc><w:p><w:r><w:t>[[/each]]</w:t></w:r></w:p></w:tc></w:tr>
        </w:tbl>
        <w:p/>
      </w:tc>
    </w:tr>
    <w:tr><w:tc><w:tcPr><w:vMerge/></w:tcPr><w:p/></w:tc><w:tc><w:tcPr><w:gridSpan w:val="2"/></w:tcPr><w:p><w:r><w:rPr><w:i/></w:rPr><w:t>Continuation</w:t></w:r></w:p></w:tc></w:tr>
    <w:tr><w:tc><w:tcPr><w:gridSpan w:val="3"/></w:tcPr><w:p><w:r><w:t>[[/each]]</w:t></w:r></w:p></w:tc></w:tr>
  </w:tbl>
  <w:p><w:r><w:t>Fixture end</w:t></w:r></w:p>
  <w:sectPr><w:pgSz w:w="12240" w:h="15840"/><w:pgMar w:top="1440" w:right="1440" w:bottom="1440" w:left="1440"/></w:sectPr>
</w:body></w:document>`
}
