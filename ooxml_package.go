package namat

import (
	"archive/zip"
	"fmt"
	"path"
	"strconv"
	"strings"
)

const (
	relationshipNamespace  = "http://schemas.openxmlformats.org/package/2006/relationships"
	officeRelationshipBase = "http://schemas.openxmlformats.org/officeDocument/2006/relationships/"
)

func (p *docxPackage) addPart(name string, data []byte, method uint16) error {
	name = strings.TrimPrefix(strings.ReplaceAll(name, "\\", "/"), "/")
	if name == "" || strings.Contains(name, "../") {
		return fmt.Errorf("invalid OOXML part name %q", name)
	}
	if _, exists := p.Parts[name]; exists {
		return fmt.Errorf("OOXML part already exists: %s", name)
	}
	header := zip.FileHeader{Name: name, Method: method}
	p.Parts[name] = &packagePart{Header: header, Data: append([]byte(nil), data...)}
	p.Order = append(p.Order, name)
	return nil
}

func (p *docxPackage) uniquePartName(directory, stem, extension string) string {
	extension = strings.TrimPrefix(strings.ToLower(extension), ".")
	for index := 1; ; index++ {
		name := path.Join(directory, stem+strconv.Itoa(index)+"."+extension)
		if _, exists := p.Parts[name]; !exists {
			return name
		}
	}
}

func (p *docxPackage) addRelationship(sourcePart, relationshipType, target, targetMode string) (string, error) {
	relsName := relationshipPartName(sourcePart)
	var root *xmlNode
	if part, exists := p.Parts[relsName]; exists {
		parsed, err := parseXML(part.Data)
		if err != nil {
			return "", fmt.Errorf("parse relationships for %s: %w", sourcePart, err)
		}
		root = parsed
	} else {
		parsed, err := parseXML([]byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Relationships xmlns="` + relationshipNamespace + `"></Relationships>`))
		if err != nil {
			return "", err
		}
		root = parsed
	}
	relationships := firstElement(root)
	if relationships == nil || !relationships.is("Relationships") {
		return "", fmt.Errorf("invalid relationships part for %s", sourcePart)
	}
	used := make(map[string]bool)
	for _, child := range relationships.Children {
		if child.is("Relationship") {
			used[attributeValue(child, "Id")] = true
		}
	}
	var id string
	for index := 1; ; index++ {
		candidate := "rIdNamat" + strconv.Itoa(index)
		if !used[candidate] {
			id = candidate
			break
		}
	}
	attrs := []xmlAttribute{
		{name: "Id", value: id},
		{name: "Type", value: relationshipType},
		{name: "Target", value: target},
	}
	if targetMode != "" {
		attrs = append(attrs, xmlAttribute{name: "TargetMode", value: targetMode})
	}
	relationships.Children = append(relationships.Children, elementNode("Relationship", attrs...))
	content, err := root.bytes()
	if err != nil {
		return "", err
	}
	if part, exists := p.Parts[relsName]; exists {
		part.Data = content
	} else if err := p.addPart(relsName, content, zip.Deflate); err != nil {
		return "", err
	}
	return id, nil
}

func (p *docxPackage) ensureDefaultContentType(extension, contentType string) error {
	extension = strings.TrimPrefix(strings.ToLower(extension), ".")
	part := p.Parts["[Content_Types].xml"]
	root, err := parseXML(part.Data)
	if err != nil {
		return fmt.Errorf("parse [Content_Types].xml: %w", err)
	}
	types := firstElement(root)
	if types == nil || !types.is("Types") {
		return fmt.Errorf("invalid [Content_Types].xml")
	}
	for _, child := range types.Children {
		if child.is("Default") && strings.EqualFold(attributeValue(child, "Extension"), extension) {
			if existing := attributeValue(child, "ContentType"); existing != contentType {
				return fmt.Errorf("extension %s already maps to content type %s", extension, existing)
			}
			return nil
		}
	}
	types.Children = append(types.Children, elementNode("Default",
		xmlAttribute{name: "Extension", value: extension},
		xmlAttribute{name: "ContentType", value: contentType},
	))
	content, err := root.bytes()
	if err != nil {
		return err
	}
	part.Data = content
	return nil
}

func relationshipPartName(sourcePart string) string {
	directory, filename := path.Split(strings.ReplaceAll(sourcePart, "\\", "/"))
	return path.Join(strings.TrimSuffix(directory, "/"), "_rels", filename+".rels")
}

func firstElement(root *xmlNode) *xmlNode {
	for _, child := range root.Children {
		if child.Type == xmlElement {
			return child
		}
	}
	return nil
}

func attributeValue(node *xmlNode, local string) string {
	for _, attribute := range node.Attrs {
		if attribute.Name.Local == local {
			return attribute.Value
		}
	}
	return ""
}
