package namat

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

// Metadata contains standard Word core and extended document properties.
// Values such as Pages and Words are cached by Word and are not recalculated
// by Namat.
type Metadata struct {
	Title                string
	Subject              string
	Creator              string
	Keywords             string
	Description          string
	LastModifiedBy       string
	Revision             string
	Category             string
	ContentStatus        string
	Company              string
	Template             string
	Application          string
	AppVersion           string
	Created              *time.Time
	Modified             *time.Time
	LastPrinted          *time.Time
	Pages                int
	Words                int
	Characters           int
	CharactersWithSpaces int
	Lines                int
	Paragraphs           int
}

// GetMetadata reads cached metadata fields from a DOCX or DOCM package.
func GetMetadata(document []byte) (Metadata, error) {
	pkg, err := readPackage(document)
	if err != nil {
		return Metadata{}, err
	}
	var metadata Metadata
	if part, ok := pkg.Parts["docProps/core.xml"]; ok {
		values, err := simpleXMLValues(part.Data)
		if err != nil {
			return Metadata{}, fmt.Errorf("namat: parse core metadata: %w", err)
		}
		metadata.Title = values["title"]
		metadata.Subject = values["subject"]
		metadata.Creator = values["creator"]
		metadata.Keywords = values["keywords"]
		metadata.Description = values["description"]
		metadata.LastModifiedBy = values["lastModifiedBy"]
		metadata.Revision = values["revision"]
		metadata.Category = values["category"]
		metadata.ContentStatus = values["contentStatus"]
		metadata.Created = metadataTime(values["created"])
		metadata.Modified = metadataTime(values["modified"])
		metadata.LastPrinted = metadataTime(values["lastPrinted"])
	}
	if part, ok := pkg.Parts["docProps/app.xml"]; ok {
		values, err := simpleXMLValues(part.Data)
		if err != nil {
			return Metadata{}, fmt.Errorf("namat: parse extended metadata: %w", err)
		}
		metadata.Company = values["Company"]
		metadata.Template = values["Template"]
		metadata.Application = values["Application"]
		metadata.AppVersion = values["AppVersion"]
		metadata.Pages = metadataInt(values["Pages"])
		metadata.Words = metadataInt(values["Words"])
		metadata.Characters = metadataInt(values["Characters"])
		metadata.CharactersWithSpaces = metadataInt(values["CharactersWithSpaces"])
		metadata.Lines = metadataInt(values["Lines"])
		metadata.Paragraphs = metadataInt(values["Paragraphs"])
	}
	return metadata, nil
}

func simpleXMLValues(data []byte) (map[string]string, error) {
	decoder := xml.NewDecoder(bytes.NewReader(data))
	values := make(map[string]string)
	type frame struct {
		name string
		text strings.Builder
	}
	var stack []*frame
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			return values, nil
		}
		if err != nil {
			return nil, err
		}
		switch current := token.(type) {
		case xml.StartElement:
			stack = append(stack, &frame{name: current.Name.Local})
		case xml.CharData:
			if len(stack) > 0 {
				stack[len(stack)-1].text.Write([]byte(current))
			}
		case xml.EndElement:
			finished := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			value := strings.TrimSpace(finished.text.String())
			if value != "" {
				values[finished.name] = value
			}
		}
	}
}

func metadataTime(value string) *time.Time {
	if value == "" {
		return nil
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return nil
	}
	return &parsed
}

func metadataInt(value string) int {
	parsed, _ := strconv.Atoi(strings.TrimSpace(value))
	return parsed
}
