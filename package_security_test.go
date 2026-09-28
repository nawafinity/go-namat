package namat

import (
	"archive/zip"
	"bytes"
	"errors"
	"strings"
	"testing"
)

func TestPackageRejectsTraversalAndDuplicateParts(t *testing.T) {
	makeZip := func(names []string) []byte {
		var out bytes.Buffer
		writer := zip.NewWriter(&out)
		for _, name := range names {
			stream, err := writer.Create(name)
			if err != nil {
				t.Fatal(err)
			}
			_, _ = stream.Write([]byte("x"))
		}
		if err := writer.Close(); err != nil {
			t.Fatal(err)
		}
		return out.Bytes()
	}

	if _, err := readPackageWithLimits(makeZip([]string{"../escape.xml"}), 1024, 4096, 10); err == nil || !strings.Contains(err.Error(), "invalid") || !errors.Is(err, ErrInvalidTemplate) {
		t.Fatalf("traversal error = %v", err)
	}
	if _, err := readPackageWithLimits(makeZip([]string{"word/document.xml", "word/document.xml"}), 1024, 4096, 10); err == nil || !strings.Contains(err.Error(), "duplicate") || !errors.Is(err, ErrInvalidTemplate) {
		t.Fatalf("duplicate error = %v", err)
	}
}

func TestPackageRejectsNonZIPAsInvalidTemplate(t *testing.T) {
	_, err := readPackage([]byte("not a DOCX"))
	if err == nil || !errors.Is(err, ErrInvalidTemplate) {
		t.Fatalf("error = %v, want invalid-template category", err)
	}
}

func TestPackageLimitsPartsAndUncompressedSize(t *testing.T) {
	template := testDOCX(t, map[string][]byte{"word/document.xml": []byte(wordDocument(`<w:p/>`)), "word/large.bin": bytes.Repeat([]byte("a"), 2048)})
	if _, err := readPackageWithLimits(template, 1024, 4096, 10); err == nil || !strings.Contains(err.Error(), "size limit") || !errors.Is(err, ErrSecurityLimit) {
		t.Fatalf("part-size error = %v", err)
	}
	if _, err := readPackageWithLimits(template, 4096, 1024, 10); err == nil || !strings.Contains(err.Error(), "uncompressed") || !errors.Is(err, ErrSecurityLimit) {
		t.Fatalf("total-size error = %v", err)
	}
	if _, err := readPackageWithLimits(template, 4096, 8192, 2); err == nil || !strings.Contains(err.Error(), "parts") || !errors.Is(err, ErrSecurityLimit) {
		t.Fatalf("part-count error = %v", err)
	}
}
