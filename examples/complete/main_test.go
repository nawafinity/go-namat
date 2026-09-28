package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestCompleteExample(t *testing.T) {
	originalArgs, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	originalProcessArgs := os.Args
	directory := t.TempDir()
	if err := os.Chdir(directory); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.Chdir(originalArgs)
		os.Args = originalProcessArgs
	}()

	os.Args = []string{"complete"}
	main()
	if _, err := os.Stat(filepath.Join(directory, "namat-demo.docx")); err != nil {
		t.Fatalf("default report: %v", err)
	}

	custom := filepath.Join(directory, "custom.docx")
	os.Args = []string{"complete", custom}
	main()
	if _, err := os.Stat(custom); err != nil {
		t.Fatalf("custom report: %v", err)
	}
	if len(templateDOCX()) == 0 || len(bannerPNG()) == 0 {
		t.Fatal("example assets are empty")
	}
}

func TestCheckPanicsOnError(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("check did not panic")
		}
	}()
	check(errors.New("failure"))
}
