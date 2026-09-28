package namat

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestCommittedTestsUseEnglishAndProductNeutralData(t *testing.T) {
	forbidden := []string{"o" + "eg", "data." + "gov.sa/" + "oe", "od" + "oe"}
	err := filepath.WalkDir(".", func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if path == ".git" || path == "dist" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, "_test.go") && !strings.Contains(filepath.ToSlash(path), "/testdata/") {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		lower := strings.ToLower(string(content))
		for _, marker := range forbidden {
			if strings.Contains(lower, marker) {
				t.Errorf("%s contains product-specific marker %q", path, marker)
			}
		}
		if utf8.Valid(content) {
			for _, r := range string(content) {
				if isArabicScriptRune(r) {
					t.Errorf("%s contains a literal Arabic-script rune; use English test source and Unicode escapes for multilingual cases", path)
					break
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func isArabicScriptRune(r rune) bool {
	return r >= 0x0600 && r <= 0x06ff ||
		r >= 0x0750 && r <= 0x077f ||
		r >= 0x0870 && r <= 0x089f ||
		r >= 0x08a0 && r <= 0x08ff ||
		r >= 0xfb50 && r <= 0xfdff ||
		r >= 0xfe70 && r <= 0xfeff
}

func TestFeatureCoverageMatrixReferencesExistingTests(t *testing.T) {
	matrix, err := os.ReadFile("docs/FEATURE_COVERAGE.md")
	if err != nil {
		t.Fatal(err)
	}
	referencePattern := regexp.MustCompile(`(?:Test|Fuzz)[A-Za-z0-9_]+`)
	references := referencePattern.FindAllString(string(matrix), -1)
	if len(references) < 40 {
		t.Fatalf("feature matrix has only %d test references; expected at least 40", len(references))
	}

	var sources strings.Builder
	err = filepath.WalkDir(".", func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if path == ".git" || path == "dist" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, "_test.go") {
			return nil
		}
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		sources.Write(content)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, reference := range references {
		if !strings.Contains(sources.String(), "func "+reference+"(") {
			t.Errorf("feature matrix references missing test %s", reference)
		}
	}
}
