package engine

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"strings"
)

type xmlNodeType uint8

const (
	xmlDocument xmlNodeType = iota
	xmlElement
	xmlText
	xmlComment
	xmlDirective
	xmlProcInst
)

type xmlNode struct {
	Type     xmlNodeType
	Name     xml.Name
	Attrs    []xml.Attr
	Data     string
	Target   string
	Children []*xmlNode
}

type xmlAttribute struct {
	name  string
	value string
}

func elementNode(name string, attributes ...xmlAttribute) *xmlNode {
	node := &xmlNode{Type: xmlElement, Name: xml.Name{Local: name}}
	for _, attribute := range attributes {
		node.Attrs = append(node.Attrs, xml.Attr{Name: xml.Name{Local: attribute.name}, Value: attribute.value})
	}
	return node
}

func parseXML(data []byte) (*xmlNode, error) {
	decoder := xml.NewDecoder(bytes.NewReader(data))
	root := &xmlNode{Type: xmlDocument}
	stack := []*xmlNode{root}
	for {
		token, err := decoder.RawToken()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		parent := stack[len(stack)-1]
		switch t := token.(type) {
		case xml.StartElement:
			node := &xmlNode{Type: xmlElement, Name: t.Name, Attrs: append([]xml.Attr(nil), t.Attr...)}
			parent.Children = append(parent.Children, node)
			stack = append(stack, node)
		case xml.EndElement:
			if len(stack) == 1 {
				return nil, fmt.Errorf("unexpected closing element %s", qname(t.Name))
			}
			open := stack[len(stack)-1]
			if open.Name != t.Name {
				return nil, fmt.Errorf("closing element %s does not match %s", qname(t.Name), qname(open.Name))
			}
			stack = stack[:len(stack)-1]
		case xml.CharData:
			parent.Children = append(parent.Children, &xmlNode{Type: xmlText, Data: string(t)})
		case xml.Comment:
			parent.Children = append(parent.Children, &xmlNode{Type: xmlComment, Data: string(t)})
		case xml.Directive:
			parent.Children = append(parent.Children, &xmlNode{Type: xmlDirective, Data: string(t)})
		case xml.ProcInst:
			parent.Children = append(parent.Children, &xmlNode{Type: xmlProcInst, Target: t.Target, Data: string(t.Inst)})
		}
	}
	if len(stack) != 1 {
		return nil, fmt.Errorf("unclosed XML element %s", qname(stack[len(stack)-1].Name))
	}
	return root, nil
}

func (n *xmlNode) bytes() []byte {
	var out bytes.Buffer
	writeXML(&out, n)
	return out.Bytes()
}

func writeXML(out *bytes.Buffer, n *xmlNode) {
	switch n.Type {
	case xmlDocument:
		for _, child := range n.Children {
			writeXML(out, child)
		}
	case xmlElement:
		out.WriteByte('<')
		out.WriteString(qname(n.Name))
		for _, attr := range n.Attrs {
			out.WriteByte(' ')
			out.WriteString(qname(attr.Name))
			out.WriteString(`="`)
			_ = xml.EscapeText(out, []byte(attr.Value))
			out.WriteByte('"')
		}
		out.WriteByte('>')
		for _, child := range n.Children {
			writeXML(out, child)
		}
		out.WriteString("</")
		out.WriteString(qname(n.Name))
		out.WriteByte('>')
	case xmlText:
		_ = xml.EscapeText(out, []byte(n.Data))
	case xmlComment:
		out.WriteString("<!--")
		out.WriteString(n.Data)
		out.WriteString("-->")
	case xmlDirective:
		out.WriteString("<!")
		out.WriteString(n.Data)
		out.WriteByte('>')
	case xmlProcInst:
		out.WriteString("<?")
		out.WriteString(n.Target)
		if n.Data != "" {
			out.WriteByte(' ')
			out.WriteString(n.Data)
		}
		out.WriteString("?>")
	}
}

func qname(name xml.Name) string {
	if name.Space == "" {
		return name.Local
	}
	return name.Space + ":" + name.Local
}

func (n *xmlNode) clone() *xmlNode {
	if n == nil {
		return nil
	}
	copyNode := &xmlNode{Type: n.Type, Name: n.Name, Data: n.Data, Target: n.Target, Attrs: append([]xml.Attr(nil), n.Attrs...)}
	copyNode.Children = make([]*xmlNode, len(n.Children))
	for i, child := range n.Children {
		copyNode.Children[i] = child.clone()
	}
	return copyNode
}

func (n *xmlNode) is(local string) bool {
	return n != nil && n.Type == xmlElement && n.Name.Local == local
}

func (n *xmlNode) descendants(local string) []*xmlNode {
	var result []*xmlNode
	var walk func(*xmlNode)
	walk = func(current *xmlNode) {
		if current.is(local) {
			result = append(result, current)
		}
		for _, child := range current.Children {
			walk(child)
		}
	}
	walk(n)
	return result
}

func textOfParagraph(paragraph *xmlNode) string {
	var out strings.Builder
	for _, text := range paragraph.descendants("t") {
		for _, child := range text.Children {
			if child.Type == xmlText {
				out.WriteString(child.Data)
			}
		}
	}
	return out.String()
}

func textNodes(paragraph *xmlNode) []*xmlNode {
	var result []*xmlNode
	for _, element := range paragraph.descendants("t") {
		for _, child := range element.Children {
			if child.Type == xmlText {
				result = append(result, child)
			}
		}
	}
	return result
}

func setTextOfNode(node *xmlNode, value string) error {
	nodes := textNodes(node)
	if len(nodes) == 0 {
		if value == "" {
			return nil
		}
		return fmt.Errorf("OOXML node contains no text element")
	}
	nodes[0].Data = value
	for _, text := range nodes[1:] {
		text.Data = ""
	}
	return nil
}

func parseXMLFragment(source string) ([]*xmlNode, error) {
	wrapper := `<namat:fragment xmlns:namat="urn:namat" xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships" xmlns:wp="http://schemas.openxmlformats.org/drawingml/2006/wordprocessingDrawing" xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" xmlns:pic="http://schemas.openxmlformats.org/drawingml/2006/picture">` + source + `</namat:fragment>`
	root, err := parseXML([]byte(wrapper))
	if err != nil {
		return nil, err
	}
	fragment := firstElement(root)
	return fragment.Children, nil
}
