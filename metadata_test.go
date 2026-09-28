package namat

import (
	"testing"
	"time"
)

func TestGetMetadata(t *testing.T) {
	document := testDOCX(t, map[string][]byte{
		"word/document.xml": []byte(wordDocument(`<w:p><w:r><w:t>document</w:t></w:r></w:p>`)),
		"docProps/core.xml": []byte(`<?xml version="1.0"?><cp:coreProperties xmlns:cp="http://schemas.openxmlformats.org/package/2006/metadata/core-properties" xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:dcterms="http://purl.org/dc/terms/"><dc:title>تقرير</dc:title><dc:creator>Namat</dc:creator><cp:revision>7</cp:revision><dcterms:created>2026-09-28T10:00:00Z</dcterms:created></cp:coreProperties>`),
		"docProps/app.xml":  []byte(`<?xml version="1.0"?><Properties xmlns="http://schemas.openxmlformats.org/officeDocument/2006/extended-properties"><Application>Microsoft Word</Application><Pages>3</Pages><Words>450</Words><Paragraphs>18</Paragraphs><Company>Nawafinity</Company></Properties>`),
	})
	metadata, err := GetMetadata(document)
	if err != nil {
		t.Fatalf("GetMetadata: %v", err)
	}
	if metadata.Title != "تقرير" || metadata.Creator != "Namat" || metadata.Revision != "7" {
		t.Fatalf("unexpected core metadata: %#v", metadata)
	}
	if metadata.Pages != 3 || metadata.Words != 450 || metadata.Paragraphs != 18 || metadata.Company != "Nawafinity" {
		t.Fatalf("unexpected extended metadata: %#v", metadata)
	}
	want := time.Date(2026, 9, 28, 10, 0, 0, 0, time.UTC)
	if metadata.Created == nil || !metadata.Created.Equal(want) {
		t.Fatalf("created = %v, want %v", metadata.Created, want)
	}
}
