package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestHelpAndVersion(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := run([]string{"help"}, &stdout, &stderr); code != 0 || !bytes.Contains(stdout.Bytes(), []byte("namat render")) {
		t.Fatalf("help code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := run([]string{"version"}, &stdout, &stderr); code != 0 || stdout.String() == "" {
		t.Fatalf("version code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
}

func TestAtomicWriteRefusesImplicitReplacement(t *testing.T) {
	directory := t.TempDir()
	name := filepath.Join(directory, "report.docx")
	if err := atomicWrite(name, []byte("first"), false); err != nil {
		t.Fatal(err)
	}
	if err := atomicWrite(name, []byte("second"), false); err == nil {
		t.Fatal("expected rename failure when destination exists")
	}
	content, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "first" {
		t.Fatalf("existing output changed to %q", content)
	}
	if err := atomicWrite(name, []byte("second"), true); err != nil {
		t.Fatal(err)
	}
	content, _ = os.ReadFile(name)
	if string(content) != "second" {
		t.Fatalf("forced output = %q", content)
	}
}

func TestUnknownCommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := run([]string{"unknown"}, &stdout, &stderr); code != 2 || !bytes.Contains(stderr.Bytes(), []byte("unknown command")) {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
}
