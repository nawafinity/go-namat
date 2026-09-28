package namat

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"math"
	"net/url"
	"reflect"
	"strconv"
	"strings"
)

const emuPerCentimeter = 360000.0

// Image describes an inline image returned by an IMAGE expression.
type Image struct {
	// Data contains encoded PNG, JPEG, GIF, or SVG bytes.
	Data []byte
	// Extension identifies the image format without requiring a leading dot.
	Extension string
	// Width is the rendered width in centimeters.
	Width float64
	// Height is the rendered height in centimeters.
	Height float64
	// Alt is the accessibility description stored in drawing properties.
	Alt string
	// Rotation is measured clockwise in degrees.
	Rotation float64
	// Caption is optional text emitted below the inline image.
	Caption string
	// Thumbnail is an optional raster fallback for an SVG image.
	Thumbnail *Image
}

// Link describes an external hyperlink returned by a LINK expression.
type Link struct {
	// URL is an absolute external URL whose scheme must be allowed by Options.
	URL string
	// Label is the visible link text; an empty label falls back to URL.
	Label string
	// Tooltip is optional hover text stored in the hyperlink element.
	Tooltip string
}

type paragraphAction struct {
	start int
	end   int
	text  *string
	node  *xmlNode
}

func imageFromValue(value any) (Image, error) {
	if image, ok := value.(Image); ok {
		return validateImage(image)
	}
	if image, ok := value.(*Image); ok && image != nil {
		return validateImage(*image)
	}
	dataValue, _ := valueField(value, "data")
	extensionValue, _ := valueField(value, "extension")
	widthValue, _ := valueField(value, "width")
	heightValue, _ := valueField(value, "height")
	altValue, _ := valueField(value, "alt")
	rotationValue, _ := valueField(value, "rotation")
	captionValue, _ := valueField(value, "caption")
	thumbnailValue, hasThumbnail := valueField(value, "thumbnail")
	data, err := decodeImageData(dataValue)
	if err != nil {
		return Image{}, err
	}
	image := Image{
		Data:      data,
		Extension: formatValue(extensionValue),
		Width:     numericValue(widthValue),
		Height:    numericValue(heightValue),
		Alt:       formatValue(altValue),
		Rotation:  numericValue(rotationValue),
		Caption:   formatValue(captionValue),
	}
	if hasThumbnail && thumbnailValue != nil {
		thumbnailDataValue, _ := valueField(thumbnailValue, "data")
		thumbnailExtensionValue, _ := valueField(thumbnailValue, "extension")
		thumbnailData, decodeErr := decodeImageData(thumbnailDataValue)
		if decodeErr != nil {
			return Image{}, fmt.Errorf("IMAGE thumbnail: %w", decodeErr)
		}
		image.Thumbnail = &Image{Data: thumbnailData, Extension: formatValue(thumbnailExtensionValue)}
	}
	return validateImage(image)
}

func validateImage(image Image) (Image, error) {
	image.Extension = strings.TrimPrefix(strings.ToLower(strings.TrimSpace(image.Extension)), ".")
	if image.Extension == "jpeg" {
		image.Extension = "jpg"
	}
	if len(image.Data) == 0 {
		return Image{}, fmt.Errorf("IMAGE data is empty")
	}
	if _, ok := imageContentTypes[image.Extension]; !ok {
		return Image{}, fmt.Errorf("unsupported IMAGE extension %q", image.Extension)
	}
	if image.Width <= 0 || image.Height <= 0 || math.IsNaN(image.Width) || math.IsNaN(image.Height) || math.IsInf(image.Width, 0) || math.IsInf(image.Height, 0) {
		return Image{}, fmt.Errorf("IMAGE width and height must be finite positive centimeters")
	}
	if image.Width > 100 || image.Height > 100 {
		return Image{}, fmt.Errorf("IMAGE dimensions exceed 100 cm")
	}
	if image.Thumbnail != nil {
		image.Thumbnail.Extension = strings.TrimPrefix(strings.ToLower(strings.TrimSpace(image.Thumbnail.Extension)), ".")
		if image.Thumbnail.Extension == "jpeg" {
			image.Thumbnail.Extension = "jpg"
		}
		if len(image.Thumbnail.Data) == 0 {
			return Image{}, fmt.Errorf("IMAGE thumbnail data is empty")
		}
		if image.Thumbnail.Extension == "svg" || imageContentTypes[image.Thumbnail.Extension] == "" {
			return Image{}, fmt.Errorf("IMAGE thumbnail must be PNG, JPEG, or GIF")
		}
	}
	return image, nil
}

var imageContentTypes = map[string]string{
	"png": "image/png",
	"gif": "image/gif",
	"jpg": "image/jpeg",
	"svg": "image/svg+xml",
}

func decodeImageData(value any) ([]byte, error) {
	switch data := value.(type) {
	case []byte:
		return append([]byte(nil), data...), nil
	case string:
		encoded := strings.TrimSpace(data)
		if comma := strings.Index(encoded, ","); strings.HasPrefix(encoded, "data:") && comma >= 0 {
			encoded = encoded[comma+1:]
		}
		decoded, err := base64.StdEncoding.DecodeString(encoded)
		if err != nil {
			return nil, fmt.Errorf("IMAGE data is not valid base64: %w", err)
		}
		return decoded, nil
	default:
		return nil, fmt.Errorf("IMAGE data must be []byte or base64 text, got %T", value)
	}
}

func linkFromValue(value any) (Link, error) {
	if link, ok := value.(Link); ok {
		return validateLink(link)
	}
	if link, ok := value.(*Link); ok && link != nil {
		return validateLink(*link)
	}
	urlValue, _ := valueField(value, "url")
	labelValue, _ := valueField(value, "label")
	tooltipValue, _ := valueField(value, "tooltip")
	link := Link{URL: formatValue(urlValue), Label: formatValue(labelValue), Tooltip: formatValue(tooltipValue)}
	return validateLink(link)
}

func validateLink(link Link) (Link, error) {
	link.URL = strings.TrimSpace(link.URL)
	if link.URL == "" {
		return Link{}, fmt.Errorf("LINK url is empty")
	}
	if link.Label == "" {
		link.Label = link.URL
	}
	return link, nil
}

func (s *renderState) imageNode(value any) (*xmlNode, error) {
	image, err := imageFromValue(value)
	if err != nil {
		return nil, err
	}
	if int64(len(image.Data)) > s.template.options.MaxOutputBytes {
		return nil, fmt.Errorf("%w: IMAGE data exceeds MaxOutputBytes", ErrSecurityLimit)
	}
	relationshipID, err := s.addImageResource(image.Data, image.Extension)
	if err != nil {
		return nil, err
	}
	blip := `<a:blip r:embed="` + escapeXML(relationshipID) + `"/>`
	if image.Extension == "svg" && image.Thumbnail != nil {
		thumbnailID, thumbnailErr := s.addImageResource(image.Thumbnail.Data, image.Thumbnail.Extension)
		if thumbnailErr != nil {
			return nil, thumbnailErr
		}
		blip = `<a:blip r:embed="` + escapeXML(thumbnailID) + `"><a:extLst><a:ext uri="{96DAC541-7B7A-43D3-8B79-37D633B846F1}"><asvg:svgBlip xmlns:asvg="http://schemas.microsoft.com/office/drawing/2016/SVG/main" r:embed="` + escapeXML(relationshipID) + `"/></a:ext></a:extLst></a:blip>`
	}
	s.counter.resources++
	drawingID := s.counter.resources
	cx := int64(math.Round(image.Width * emuPerCentimeter))
	cy := int64(math.Round(image.Height * emuPerCentimeter))
	rotation := int64(math.Round(image.Rotation * 60000))
	name := "Namat image " + strconv.Itoa(drawingID)
	xmlSource := fmt.Sprintf(`<w:r><w:drawing xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships" xmlns:wp="http://schemas.openxmlformats.org/drawingml/2006/wordprocessingDrawing" xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" xmlns:pic="http://schemas.openxmlformats.org/drawingml/2006/picture"><wp:inline distT="0" distB="0" distL="0" distR="0"><wp:extent cx="%d" cy="%d"/><wp:effectExtent l="0" t="0" r="0" b="0"/><wp:docPr id="%d" name="%s" descr="%s"/><wp:cNvGraphicFramePr><a:graphicFrameLocks noChangeAspect="1"/></wp:cNvGraphicFramePr><a:graphic><a:graphicData uri="http://schemas.openxmlformats.org/drawingml/2006/picture"><pic:pic><pic:nvPicPr><pic:cNvPr id="%d" name="%s" descr="%s"/><pic:cNvPicPr/></pic:nvPicPr><pic:blipFill>%s<a:stretch><a:fillRect/></a:stretch></pic:blipFill><pic:spPr><a:xfrm rot="%d"><a:off x="0" y="0"/><a:ext cx="%d" cy="%d"/></a:xfrm><a:prstGeom prst="rect"><a:avLst/></a:prstGeom></pic:spPr></pic:pic></a:graphicData></a:graphic></wp:inline></w:drawing></w:r>`, cx, cy, drawingID, escapeXML(name), escapeXML(image.Alt), drawingID, escapeXML(name), escapeXML(image.Alt), blip, rotation, cx, cy)
	nodes, err := parseXMLFragment(xmlSource)
	if err != nil || len(nodes) != 1 {
		return nil, fmt.Errorf("build IMAGE drawing: %w", err)
	}
	if image.Caption != "" {
		caption := elementWithPrefix("w", "r")
		caption.Children = append(caption.Children, elementWithPrefix("w", "br"))
		text := elementWithPrefix("w", "t")
		text.Children = append(text.Children, &xmlNode{Type: xmlText, Data: image.Caption})
		caption.Children = append(caption.Children, text)
		nodes[0].Children = append(nodes[0].Children, caption.Children...)
	}
	return nodes[0], nil
}

func (s *renderState) addImageResource(data []byte, extension string) (string, error) {
	partName := s.pkg.uniquePartName("word/media", "namat-image-", extension)
	if err := s.pkg.addPart(partName, data, zip.Deflate); err != nil {
		return "", err
	}
	if err := s.pkg.ensureDefaultContentType(extension, imageContentTypes[extension]); err != nil {
		return "", err
	}
	return s.pkg.addRelationship(s.partName, officeRelationshipBase+"image", "media/"+pathBase(partName), "")
}

func (s *renderState) linkNode(value any) (*xmlNode, error) {
	link, err := linkFromValue(value)
	if err != nil {
		return nil, err
	}
	parsed, err := url.Parse(link.URL)
	if err != nil || parsed.Scheme == "" {
		return nil, fmt.Errorf("LINK url must be absolute")
	}
	allowed := false
	for _, scheme := range s.template.options.AllowedLinkSchemes {
		if strings.EqualFold(parsed.Scheme, scheme) {
			allowed = true
			break
		}
	}
	if !allowed {
		return nil, fmt.Errorf("LINK scheme %q is not allowed", parsed.Scheme)
	}
	relationshipID, err := s.pkg.addRelationship(s.partName, officeRelationshipBase+"hyperlink", link.URL, "External")
	if err != nil {
		return nil, err
	}
	tooltip := ""
	if link.Tooltip != "" {
		tooltip = ` w:tooltip="` + escapeXML(link.Tooltip) + `"`
	}
	nodes, err := parseXMLFragment(`<w:hyperlink xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships" r:id="` + escapeXML(relationshipID) + `" w:history="1"` + tooltip + `><w:r><w:rPr><w:rStyle w:val="Hyperlink"/><w:color w:val="0563C1"/><w:u w:val="single"/></w:rPr><w:t xml:space="preserve">` + escapeXML(link.Label) + `</w:t></w:r></w:hyperlink>`)
	if err != nil || len(nodes) != 1 {
		return nil, fmt.Errorf("build LINK node: %w", err)
	}
	return nodes[0], nil
}

func (s *renderState) htmlNode(value any) (*xmlNode, error) {
	html := formatValue(value)
	if html == "" && s.template.options.RejectNullish {
		return nil, fmt.Errorf("HTML result is empty")
	}
	partName := s.pkg.uniquePartName("word", "namat-html-", "html")
	if err := s.pkg.addPart(partName, []byte(html), zip.Deflate); err != nil {
		return nil, err
	}
	if err := s.pkg.ensureDefaultContentType("html", "text/html"); err != nil {
		return nil, err
	}
	relationshipID, err := s.pkg.addRelationship(s.partName, officeRelationshipBase+"aFChunk", pathBase(partName), "")
	if err != nil {
		return nil, err
	}
	nodes, err := parseXMLFragment(`<w:altChunk xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships" r:id="` + escapeXML(relationshipID) + `"/>`)
	if err != nil || len(nodes) != 1 {
		return nil, fmt.Errorf("build HTML altChunk: %w", err)
	}
	return nodes[0], nil
}

func replaceTextRangeWithNode(paragraph *xmlNode, start, end int, inserted *xmlNode) error {
	type childSpan struct {
		index      int
		start, end int
		text       string
	}
	spans := make([]childSpan, 0, len(paragraph.Children))
	position := 0
	for index, child := range paragraph.Children {
		text := textOfParagraph(child)
		if text == "" {
			continue
		}
		spans = append(spans, childSpan{index: index, start: position, end: position + len(text), text: text})
		position += len(text)
	}
	if start < 0 || end < start || end > position {
		return fmt.Errorf("rich content range is outside paragraph text")
	}
	startSpan, endSpan := -1, -1
	for index, span := range spans {
		if startSpan < 0 && (start < span.end || (start == span.end && index == len(spans)-1)) {
			startSpan = index
		}
		if endSpan < 0 && end <= span.end {
			endSpan = index
			break
		}
	}
	if startSpan < 0 || endSpan < 0 {
		return fmt.Errorf("cannot locate rich content command in paragraph runs")
	}
	first := spans[startSpan]
	last := spans[endSpan]
	prefix := first.text[:start-first.start]
	suffix := last.text[end-last.start:]
	if first.index == last.index {
		original := paragraph.Children[first.index]
		after := original.clone()
		if err := setTextOfNode(original, prefix); err != nil {
			return err
		}
		if err := setTextOfNode(after, suffix); err != nil {
			return err
		}
		children := make([]*xmlNode, 0, len(paragraph.Children)+2)
		children = append(children, paragraph.Children[:first.index+1]...)
		children = append(children, inserted, after)
		children = append(children, paragraph.Children[first.index+1:]...)
		paragraph.Children = children
		return nil
	}
	if err := setTextOfNode(paragraph.Children[first.index], prefix); err != nil {
		return err
	}
	for index := startSpan + 1; index < endSpan; index++ {
		if err := setTextOfNode(paragraph.Children[spans[index].index], ""); err != nil {
			return err
		}
	}
	if err := setTextOfNode(paragraph.Children[last.index], suffix); err != nil {
		return err
	}
	children := make([]*xmlNode, 0, len(paragraph.Children)+1)
	children = append(children, paragraph.Children[:first.index+1]...)
	children = append(children, inserted)
	children = append(children, paragraph.Children[first.index+1:]...)
	paragraph.Children = children
	return nil
}

func expandTextMarkup(root *xmlNode, options Options) error {
	for _, run := range root.descendants("r") {
		children := make([]*xmlNode, 0, len(run.Children))
		for _, child := range run.Children {
			if !child.is("t") {
				children = append(children, child)
				continue
			}
			text := textOfParagraph(child)
			hasLiteralXML := strings.Contains(text, options.LiteralXMLDelimiter)
			if hasLiteralXML && !options.AllowRawXML {
				return fmt.Errorf("literal XML is disabled; set AllowRawXML to enable it")
			}
			if !options.DisableLineBreaks {
				text = strings.ReplaceAll(strings.ReplaceAll(text, "\r\n", "\n"), "\r", "\n")
				text = strings.ReplaceAll(text, "\n", options.LiteralXMLDelimiter+`<w:br/>`+options.LiteralXMLDelimiter)
			}
			if !strings.Contains(text, options.LiteralXMLDelimiter) {
				setTextOfNode(child, text)
				children = append(children, child)
				continue
			}
			expanded, err := literalXMLNodes(child, text, options.LiteralXMLDelimiter)
			if err != nil {
				return err
			}
			children = append(children, expanded...)
		}
		run.Children = children
	}
	return nil
}

func literalXMLNodes(template *xmlNode, text, delimiter string) ([]*xmlNode, error) {
	var result []*xmlNode
	for {
		start := strings.Index(text, delimiter)
		if start < 0 {
			if text != "" {
				copy := template.clone()
				if err := setTextOfNode(copy, text); err != nil {
					return nil, err
				}
				result = append(result, copy)
			}
			return result, nil
		}
		end := strings.Index(text[start+len(delimiter):], delimiter)
		if end < 0 {
			return nil, fmt.Errorf("unterminated literal XML delimiter")
		}
		if start > 0 {
			copy := template.clone()
			if err := setTextOfNode(copy, text[:start]); err != nil {
				return nil, err
			}
			result = append(result, copy)
		}
		end += start + len(delimiter)
		xmlNodes, err := parseXMLFragment(text[start+len(delimiter) : end])
		if err != nil {
			return nil, fmt.Errorf("parse literal XML: %w", err)
		}
		result = append(result, xmlNodes...)
		text = text[end+len(delimiter):]
	}
}

func valueField(value any, name string) (any, bool) {
	if value == nil {
		return nil, false
	}
	v := reflect.ValueOf(value)
	for v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface {
		if v.IsNil() {
			return nil, false
		}
		v = v.Elem()
	}
	switch v.Kind() {
	case reflect.Map:
		if v.Type().Key().Kind() == reflect.String {
			for _, key := range v.MapKeys() {
				if strings.EqualFold(key.String(), name) {
					return v.MapIndex(key).Interface(), true
				}
			}
		}
	case reflect.Struct:
		t := v.Type()
		for index := 0; index < v.NumField(); index++ {
			field := t.Field(index)
			jsonName := strings.Split(field.Tag.Get("json"), ",")[0]
			if strings.EqualFold(field.Name, name) || jsonName == name {
				if v.Field(index).CanInterface() {
					return v.Field(index).Interface(), true
				}
			}
		}
	}
	return nil, false
}

func numericValue(value any) float64 {
	v := reflect.ValueOf(value)
	if !v.IsValid() {
		return 0
	}
	for v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface {
		if v.IsNil() {
			return 0
		}
		v = v.Elem()
	}
	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return float64(v.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return float64(v.Uint())
	case reflect.Float32, reflect.Float64:
		return v.Float()
	default:
		return 0
	}
}

func escapeXML(value string) string {
	var out bytes.Buffer
	_ = xml.EscapeText(&out, []byte(value))
	return out.String()
}

func pathBase(name string) string {
	name = strings.ReplaceAll(name, "\\", "/")
	if index := strings.LastIndex(name, "/"); index >= 0 {
		return name[index+1:]
	}
	return name
}

func elementWithPrefix(prefix, local string) *xmlNode {
	return &xmlNode{Type: xmlElement, Name: xml.Name{Space: prefix, Local: local}}
}
