package namat

import (
	"strings"
	"testing"
)

func TestTemplateParsingEdgeBranches(t *testing.T) {
	options := Options{}.normalized()
	fragmented := mustParseElement(t, "<w:p><w:r><w:t>before [[INS val</w:t></w:r><w:r><w:t>ue</w:t></w:r><w:r><w:t>]] after</w:t></w:r></w:p>")
	normalizeCommandFragments(fragmented, options)
	if got := textOfParagraph(fragmented); got != "before [[INS value]] after" {
		t.Fatalf("normalized fragments = %q", got)
	}
	withoutClose := mustParseElement(t, "<w:p><w:r><w:t>[[INS val</w:t></w:r><w:r><w:t>ue</w:t></w:r></w:p>")
	normalizeCommandFragments(withoutClose, options)
	if got := textOfParagraph(withoutClose); got != "[[INS value" {
		t.Fatalf("unclosed normalized fragments = %q", got)
	}
	if _, err := commandSpans("prefix [[INS value", options); err == nil {
		t.Fatal("unterminated command span succeeded")
	}

	row := mustParseElement(t, "<w:tr><w:tc><w:p><w:r><w:t>[[IF true]]</w:t></w:r></w:p><w:p><w:r><w:t>[[FOR item IN items]]</w:t></w:r></w:p></w:tc></w:tr>")
	if _, _, err := structuralCommand(row, options); err == nil {
		t.Fatal("multiple row structural commands succeeded")
	}

	for _, command := range []string{"[[ELSE]]", "[[END-IF]]", "[[END-FOR]]"} {
		root := elementNode("root")
		root.Children = append(root.Children, commandParagraph(t, command))
		if err := validateNodeStructure(root, options); err == nil {
			t.Fatalf("unmatched %s succeeded", command)
		}
	}
	wrongNesting := elementNode("root")
	wrongNesting.Children = append(wrongNesting.Children,
		commandParagraph(t, "[[FOR item IN items]]"),
		commandParagraph(t, "[[ELSE]]"),
	)
	if err := validateNodeStructure(wrongNesting, options); err == nil {
		t.Fatal("ELSE inside FOR succeeded")
	}
	wrongEndIf := elementNode("root")
	wrongEndIf.Children = append(wrongEndIf.Children,
		commandParagraph(t, "[[FOR item IN items]]"),
		commandParagraph(t, "[[END-IF]]"),
	)
	if err := validateNodeStructure(wrongEndIf, options); err == nil {
		t.Fatal("END-IF inside FOR succeeded")
	}

	recursive := elementNode("root")
	child := elementNode("child")
	child.Children = append(child.Children, commandParagraph(t, "[[ELSE extra]]"))
	recursive.Children = append(recursive.Children, child)
	if err := validateNodeStructure(recursive, options); err == nil || !strings.Contains(err.Error(), "arguments") {
		t.Fatalf("recursive validation error = %v", err)
	}
}
