package engine

import "testing"

func TestTemplateParsingEdgeBranches(t *testing.T) {
	options := Options{}.normalized()
	fragmented := mustParseElement(t, "<w:p><w:r><w:t>before [[val</w:t></w:r><w:r><w:t>ue</w:t></w:r><w:r><w:t>]] after</w:t></w:r></w:p>")
	normalizeCommandFragments(fragmented, options)
	if got := textOfParagraph(fragmented); got != "before [[value]] after" {
		t.Fatalf("normalized fragments = %q", got)
	}
	withoutClose := mustParseElement(t, "<w:p><w:r><w:t>[[val</w:t></w:r><w:r><w:t>ue</w:t></w:r></w:p>")
	normalizeCommandFragments(withoutClose, options)
	if got := textOfParagraph(withoutClose); got != "[[value" {
		t.Fatalf("unclosed normalized fragments = %q", got)
	}
	if _, err := commandSpans("prefix [[value", options); err == nil {
		t.Fatal("unterminated command span succeeded")
	}

	row := mustParseElement(t, "<w:tr><w:tc><w:p><w:r><w:t>[[#if true]]</w:t></w:r></w:p><w:p><w:r><w:t>[[#each items as item]]</w:t></w:r></w:p></w:tc></w:tr>")
	if _, _, err := structuralCommand(row, options); err == nil {
		t.Fatal("multiple row structural commands succeeded")
	}

	for _, command := range []string{"[[#else]]", "[[/if]]", "[[/each]]"} {
		root := elementNode("root")
		root.Children = append(root.Children, commandParagraph(t, command))
		if err := validateNodeStructure(root, options); err == nil {
			t.Fatalf("unmatched %s succeeded", command)
		}
	}
	wrongNesting := elementNode("root")
	wrongNesting.Children = append(wrongNesting.Children,
		commandParagraph(t, "[[#each items as item]]"),
		commandParagraph(t, "[[#else]]"),
	)
	if err := validateNodeStructure(wrongNesting, options); err == nil {
		t.Fatal("ELSE inside FOR succeeded")
	}
	wrongEndIf := elementNode("root")
	wrongEndIf.Children = append(wrongEndIf.Children,
		commandParagraph(t, "[[#each items as item]]"),
		commandParagraph(t, "[[/if]]"),
	)
	if err := validateNodeStructure(wrongEndIf, options); err == nil {
		t.Fatal("END-IF inside FOR succeeded")
	}

	recursive := elementNode("root")
	child := elementNode("child")
	child.Children = append(child.Children, commandParagraph(t, "[[#else extra]]"))
	recursive.Children = append(recursive.Children, child)
	if err := validateNodeStructure(recursive, options); err == nil {
		t.Fatalf("recursive validation error = %v", err)
	}
}

func TestNestedTableMarkersDoNotBelongToOuterRow(t *testing.T) {
	options := Options{}.normalized()
	outer := mustParseElement(t, `<w:tr><w:tc><w:p><w:r><w:t>outer content</w:t></w:r></w:p><w:tbl>
<w:tr><w:tc><w:p><w:r><w:t>[[#each items as item]]</w:t></w:r></w:p></w:tc></w:tr>
<w:tr><w:tc><w:p><w:r><w:t>[[/each]]</w:t></w:r></w:p></w:tc></w:tr>
</w:tbl></w:tc></w:tr>`)
	if command, ok, err := structuralCommand(outer, options); err != nil || ok {
		t.Fatalf("outer row structural command = %#v, %v, %v", command, ok, err)
	}
	nestedRows := outer.descendants("tr")
	if len(nestedRows) != 3 {
		t.Fatalf("row count = %d, want 3", len(nestedRows))
	}
	if command, ok, err := structuralCommand(nestedRows[1], options); err != nil || !ok || command.Type != CommandEach {
		t.Fatalf("nested row structural command = %#v, %v, %v", command, ok, err)
	}
}
