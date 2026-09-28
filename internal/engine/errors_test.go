package engine

import (
	"errors"
	"strings"
	"testing"
)

func TestStructuredErrorsExposeCategories(t *testing.T) {
	inner := errors.New("synthetic failure")
	templateErr := &Error{Part: "word/document.xml", Command: "INS value", Err: inner}
	if !errors.Is(templateErr, inner) || !strings.Contains(templateErr.Error(), "word/document.xml") {
		t.Fatalf("unexpected template error: %v", templateErr)
	}

	multiple := &MultiError{Errors: []error{templateErr, ErrCommandSyntax}}
	if !errors.Is(multiple, inner) || !errors.Is(multiple, ErrCommandSyntax) {
		t.Fatalf("MultiError does not unwrap its entries: %v", multiple)
	}
	if !strings.Contains(multiple.Error(), "2 validation errors") {
		t.Fatalf("unexpected MultiError text: %q", multiple.Error())
	}
}
