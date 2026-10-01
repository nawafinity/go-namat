package main

import (
	"archive/zip"
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

type failingWriter struct{ err error }

func (w failingWriter) Write([]byte) (int, error) { return 0, w.err }

type fakeTemporaryFile struct {
	name     string
	writeErr error
	short    bool
	syncErr  error
	closeErr error
}

func (f *fakeTemporaryFile) Write(p []byte) (int, error) {
	if f.writeErr != nil {
		return 0, f.writeErr
	}
	if f.short && len(p) > 0 {
		return len(p) - 1, nil
	}
	return len(p), nil
}
func (f *fakeTemporaryFile) Name() string { return f.name }
func (f *fakeTemporaryFile) Sync() error  { return f.syncErr }
func (f *fakeTemporaryFile) Close() error { return f.closeErr }

type fakeFileSystem struct {
	statExists  bool
	statErr     error
	createErr   error
	createErrs  []error
	temporary   *fakeTemporaryFile
	temporaries []*fakeTemporaryFile
	createCalls int
	removeErrs  []error
	removeCalls int
	linkErr     error
	linkCalls   [][2]string
	renameErr   error
	renameErrs  []error
	renameCalls [][2]string
	files       map[string]string
}

func (f *fakeFileSystem) Stat(name string) (os.FileInfo, error) {
	if f.statExists {
		return fakeFileInfo{}, nil
	}
	if f.statErr != nil {
		return nil, f.statErr
	}
	return nil, os.ErrNotExist
}
func (f *fakeFileSystem) CreateTemp(string, string) (temporaryFile, error) {
	call := f.createCalls
	f.createCalls++
	if call < len(f.createErrs) && f.createErrs[call] != nil {
		return nil, f.createErrs[call]
	}
	if f.createErr != nil {
		return nil, f.createErr
	}
	if call < len(f.temporaries) {
		return f.temporaries[call], nil
	}
	if f.temporary == nil {
		f.temporary = &fakeTemporaryFile{name: "temporary"}
	}
	return f.temporary, nil
}
func (f *fakeFileSystem) Link(oldName, newName string) error {
	f.linkCalls = append(f.linkCalls, [2]string{oldName, newName})
	if f.linkErr != nil {
		return f.linkErr
	}
	if f.files != nil {
		if _, exists := f.files[newName]; exists {
			return os.ErrExist
		}
		f.files[newName] = f.files[oldName]
	}
	return nil
}
func (f *fakeFileSystem) Remove(name string) error {
	call := f.removeCalls
	f.removeCalls++
	var err error
	if call < len(f.removeErrs) {
		err = f.removeErrs[call]
	}
	if err == nil && f.files != nil {
		delete(f.files, name)
	}
	return err
}
func (f *fakeFileSystem) Rename(oldName, newName string) error {
	f.renameCalls = append(f.renameCalls, [2]string{oldName, newName})
	call := len(f.renameCalls) - 1
	err := f.renameErr
	if call < len(f.renameErrs) {
		err = f.renameErrs[call]
	}
	if err != nil {
		return err
	}
	if f.files != nil {
		value := f.files[oldName]
		delete(f.files, oldName)
		f.files[newName] = value
	}
	return nil
}

type fakeFileInfo struct{}

func (fakeFileInfo) Name() string       { return "file" }
func (fakeFileInfo) Size() int64        { return 0 }
func (fakeFileInfo) Mode() os.FileMode  { return 0 }
func (fakeFileInfo) ModTime() time.Time { return time.Time{} }
func (fakeFileInfo) IsDir() bool        { return false }
func (fakeFileInfo) Sys() any           { return nil }

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

func TestMainAndRunArgumentEdges(t *testing.T) {
	originalArgs, originalExit := os.Args, exit
	defer func() {
		os.Args = originalArgs
		exit = originalExit
	}()
	os.Args = []string{"namat", "help"}
	code := -1
	exit = func(value int) { code = value }
	main()
	if code != 0 {
		t.Fatalf("main exit code = %d", code)
	}

	for _, args := range [][]string{nil, {"inspect", "--bad"}, {"lint"}, {"lint", "--bad"}, {"metadata"}, {"render", "--bad"}, {"render"}} {
		var stdout, stderr bytes.Buffer
		if code := run(args, &stdout, &stderr); code != 2 {
			t.Fatalf("run(%v) = %d", args, code)
		}
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

func TestInspectMetadataAndRenderCommands(t *testing.T) {
	directory := t.TempDir()
	templatePath := filepath.Join(directory, "template.docx")
	dataPath := filepath.Join(directory, "data.json")
	outputPath := filepath.Join(directory, "output.docx")
	if err := os.WriteFile(templatePath, syntheticTemplate(t), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dataPath, []byte(`{"value":"synthetic"}`), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	if code := run([]string{"inspect", "--json", templatePath}, &stdout, &stderr); code != 0 || !bytes.Contains(stdout.Bytes(), []byte(`"commands":1`)) {
		t.Fatalf("inspect code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := run([]string{"lint", "--data", dataPath, "--json", templatePath}, &stdout, &stderr); code != 0 || !bytes.Contains(stdout.Bytes(), []byte(`"dataValidated":true`)) {
		t.Fatalf("lint code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := run([]string{"metadata", templatePath}, &stdout, &stderr); code != 0 || !bytes.Contains(stdout.Bytes(), []byte(`"Title": ""`)) {
		t.Fatalf("metadata code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := run([]string{"render", "--data", dataPath, "--out", outputPath, templatePath}, &stdout, &stderr); code != 0 {
		t.Fatalf("render code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	report, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(readZIPPart(t, report, "word/document.xml"), []byte("synthetic")) {
		t.Fatal("rendered package does not contain the synthetic value")
	}
}

func TestInspectFailureAndOutputEdges(t *testing.T) {
	directory := t.TempDir()
	validPath := filepath.Join(directory, "valid.docx")
	invalidPath := filepath.Join(directory, "invalid.docx")
	compileInvalidPath := filepath.Join(directory, "compile-invalid.docx")
	if err := os.WriteFile(validPath, syntheticTemplate(t), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(invalidPath, []byte("bad"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(compileInvalidPath, syntheticTemplateCommand(t, "[[@]]"), 0o600); err != nil {
		t.Fatal(err)
	}

	for _, args := range [][]string{{"inspect"}, {"inspect", "--details", filepath.Join(directory, "missing.docx")}, {"inspect", invalidPath}, {"inspect", compileInvalidPath}} {
		var stdout, stderr bytes.Buffer
		if code := run(args, &stdout, &stderr); code == 0 {
			t.Fatalf("run(%v) unexpectedly succeeded", args)
		}
	}

	var stdout, stderr bytes.Buffer
	if code := run([]string{"inspect", validPath}, &stdout, &stderr); code != 0 || !strings.Contains(stdout.String(), "insert: 1") {
		t.Fatalf("human inspect code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	wantErr := errors.New("write")
	if code := inspect([]string{"--json", validPath}, failingWriter{err: wantErr}, &stderr); code != 1 {
		t.Fatalf("inspect writer code = %d", code)
	}
}

func TestMetadataFailureAndWriterEdges(t *testing.T) {
	directory := t.TempDir()
	validPath := filepath.Join(directory, "valid.docx")
	invalidPath := filepath.Join(directory, "invalid.docx")
	_ = os.WriteFile(validPath, syntheticTemplate(t), 0o600)
	_ = os.WriteFile(invalidPath, []byte("bad"), 0o600)
	var stdout, stderr bytes.Buffer
	for _, args := range [][]string{{"metadata"}, {"metadata", filepath.Join(directory, "missing")}, {"metadata", invalidPath}} {
		stdout.Reset()
		stderr.Reset()
		if code := run(args, &stdout, &stderr); code == 0 {
			t.Fatalf("run(%v) unexpectedly succeeded", args)
		}
	}
	if code := metadata([]string{validPath}, failingWriter{err: errors.New("write")}, &stderr); code != 1 {
		t.Fatalf("metadata writer code = %d", code)
	}
}

func TestRenderFailureEdges(t *testing.T) {
	directory := t.TempDir()
	validTemplate := filepath.Join(directory, "valid.docx")
	invalidTemplate := filepath.Join(directory, "invalid.docx")
	validData := filepath.Join(directory, "data.json")
	invalidData := filepath.Join(directory, "invalid.json")
	existingOutput := filepath.Join(directory, "existing.docx")
	_ = os.WriteFile(validTemplate, syntheticTemplate(t), 0o600)
	_ = os.WriteFile(invalidTemplate, []byte("bad"), 0o600)
	_ = os.WriteFile(validData, []byte("{\"value\":\"ok\"}"), 0o600)
	_ = os.WriteFile(invalidData, []byte("{"), 0o600)
	_ = os.WriteFile(existingOutput, []byte("existing"), 0o600)

	tests := [][]string{
		{"render", "--data", validData, "--out", existingOutput, validTemplate},
		{"render", "--data", validData, "--out", filepath.Join(directory, "out1.docx"), filepath.Join(directory, "missing.docx")},
		{"render", "--data", filepath.Join(directory, "missing.json"), "--out", filepath.Join(directory, "out2.docx"), validTemplate},
		{"render", "--data", invalidData, "--out", filepath.Join(directory, "out3.docx"), validTemplate},
		{"render", "--data", validData, "--out", filepath.Join(directory, "out4.docx"), invalidTemplate},
		{"render", "--data", validData, "--out", filepath.Join(directory, "missing", "out.docx"), validTemplate},
	}
	for _, args := range tests {
		var stdout, stderr bytes.Buffer
		if code := run(args, &stdout, &stderr); code != 1 {
			t.Fatalf("run(%v) = %d, stderr=%q", args, code, stderr.String())
		}
	}
	var stdout, stderr bytes.Buffer
	if code := render([]string{"--data", validData, "--out", string([]byte{0}), validTemplate}, &stdout, &stderr); code != 1 {
		t.Fatalf("invalid output path code = %d", code)
	}
}

func TestAtomicWriteInjectedFailures(t *testing.T) {
	wantErr := errors.New("failure")
	tests := []struct {
		name    string
		fs      *fakeFileSystem
		replace bool
	}{
		{"existing", &fakeFileSystem{statExists: true}, false},
		{"stat", &fakeFileSystem{statErr: wantErr}, false},
		{"create", &fakeFileSystem{createErr: wantErr}, true},
		{"write", &fakeFileSystem{temporary: &fakeTemporaryFile{name: "tmp", writeErr: wantErr}}, true},
		{"short write", &fakeFileSystem{temporary: &fakeTemporaryFile{name: "tmp", short: true}}, true},
		{"sync", &fakeFileSystem{temporary: &fakeTemporaryFile{name: "tmp", syncErr: wantErr}}, true},
		{"close", &fakeFileSystem{temporary: &fakeTemporaryFile{name: "tmp", closeErr: wantErr}}, true},
		{"link", &fakeFileSystem{temporary: &fakeTemporaryFile{name: "tmp"}, linkErr: wantErr}, false},
		{"linked temporary cleanup", &fakeFileSystem{temporary: &fakeTemporaryFile{name: "tmp"}, removeErrs: []error{wantErr}}, false},
		{"replace rename", &fakeFileSystem{temporary: &fakeTemporaryFile{name: "tmp"}, renameErr: wantErr}, true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := atomicWriteWithFS(test.fs, "report.docx", []byte("data"), test.replace); err == nil {
				t.Fatal("expected atomic write error")
			}
		})
	}
	fs := &fakeFileSystem{temporary: &fakeTemporaryFile{name: "tmp"}}
	if err := atomicWriteWithFS(fs, "report.docx", []byte("data"), true); err != nil {
		t.Fatalf("successful injected write: %v", err)
	}
}

func TestAtomicWriteWithoutForceDoesNotOverwriteRacingDestination(t *testing.T) {
	fs := &fakeFileSystem{
		temporary: &fakeTemporaryFile{name: "new.tmp"},
		linkErr:   os.ErrExist,
		files: map[string]string{
			"new.tmp":     "new",
			"report.docx": "racing writer",
		},
	}
	err := atomicWriteWithFS(fs, "report.docx", []byte("new"), false)
	if !errors.Is(err, os.ErrExist) {
		t.Fatalf("atomicWriteWithFS error = %v, want destination-exists failure", err)
	}
	if got := fs.files["report.docx"]; got != "racing writer" {
		t.Fatalf("racing destination was overwritten with %q", got)
	}
	if len(fs.renameCalls) != 0 {
		t.Fatalf("non-replacing write called Rename: %#v", fs.renameCalls)
	}
}

func TestAtomicWriteReportsShortWrite(t *testing.T) {
	fs := &fakeFileSystem{temporary: &fakeTemporaryFile{name: "tmp", short: true}}
	if err := atomicWriteWithFS(fs, "report.docx", []byte("data"), false); !errors.Is(err, io.ErrShortWrite) {
		t.Fatalf("atomicWriteWithFS error = %v, want io.ErrShortWrite", err)
	}
}

func TestAtomicReplaceRestoresOriginalWhenInstallingNewFileFails(t *testing.T) {
	directErr := errors.New("destination exists")
	installErr := errors.New("install failed")
	fs := &fakeFileSystem{
		statExists: true,
		temporaries: []*fakeTemporaryFile{
			{name: "new.tmp"},
			{name: "backup.tmp"},
		},
		renameErrs: []error{directErr, nil, installErr, nil},
		files: map[string]string{
			"new.tmp":     "new",
			"backup.tmp":  "placeholder",
			"report.docx": "original",
		},
	}

	err := atomicWriteWithFS(fs, "report.docx", []byte("new"), true)
	if !errors.Is(err, installErr) {
		t.Fatalf("atomicWriteWithFS error = %v, want install failure", err)
	}
	wantCalls := [][2]string{
		{"new.tmp", "report.docx"},
		{"report.docx", "backup.tmp"},
		{"new.tmp", "report.docx"},
		{"backup.tmp", "report.docx"},
	}
	if !reflect.DeepEqual(fs.renameCalls, wantCalls) {
		t.Fatalf("rename calls = %#v, want restore sequence %#v", fs.renameCalls, wantCalls)
	}
	if got := fs.files["report.docx"]; got != "original" {
		t.Fatalf("restored output = %q, want original content", got)
	}
	if _, exists := fs.files["backup.tmp"]; exists {
		t.Fatal("backup still exists after restoring the original output")
	}
}

func TestAtomicReplaceFallbackFailures(t *testing.T) {
	directErr := errors.New("destination exists")
	wantErr := errors.New("failure")
	newAndBackup := func(backup *fakeTemporaryFile) []*fakeTemporaryFile {
		return []*fakeTemporaryFile{{name: "new.tmp"}, backup}
	}
	tests := []struct {
		name string
		fs   *fakeFileSystem
	}{
		{
			name: "destination stat",
			fs: &fakeFileSystem{
				statErr:    wantErr,
				temporary:  &fakeTemporaryFile{name: "new.tmp"},
				renameErrs: []error{directErr},
			},
		},
		{
			name: "backup creation",
			fs: &fakeFileSystem{
				statExists:  true,
				temporaries: newAndBackup(&fakeTemporaryFile{name: "unused"}),
				createErrs:  []error{nil, wantErr},
				renameErrs:  []error{directErr},
			},
		},
		{
			name: "backup close",
			fs: &fakeFileSystem{
				statExists:  true,
				temporaries: newAndBackup(&fakeTemporaryFile{name: "backup.tmp", closeErr: wantErr}),
				renameErrs:  []error{directErr},
			},
		},
		{
			name: "backup placeholder removal",
			fs: &fakeFileSystem{
				statExists:  true,
				temporaries: newAndBackup(&fakeTemporaryFile{name: "backup.tmp"}),
				removeErrs:  []error{wantErr},
				renameErrs:  []error{directErr},
			},
		},
		{
			name: "move original to backup",
			fs: &fakeFileSystem{
				statExists:  true,
				temporaries: newAndBackup(&fakeTemporaryFile{name: "backup.tmp"}),
				renameErrs:  []error{directErr, wantErr},
			},
		},
		{
			name: "restore original",
			fs: &fakeFileSystem{
				statExists:  true,
				temporaries: newAndBackup(&fakeTemporaryFile{name: "backup.tmp"}),
				renameErrs:  []error{directErr, nil, errors.New("install failed"), wantErr},
			},
		},
		{
			name: "backup cleanup",
			fs: &fakeFileSystem{
				statExists:  true,
				temporaries: newAndBackup(&fakeTemporaryFile{name: "backup.tmp"}),
				removeErrs:  []error{nil, wantErr},
				renameErrs:  []error{directErr, nil, nil},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := atomicWriteWithFS(test.fs, "report.docx", []byte("new"), true)
			if !errors.Is(err, wantErr) {
				t.Fatalf("atomicWriteWithFS error = %v, want injected failure", err)
			}
			if test.name == "restore original" && !strings.Contains(err.Error(), "backup.tmp") {
				t.Fatalf("restore failure does not identify recoverable backup: %v", err)
			}
		})
	}
}

func TestAtomicReplaceFallbackSucceeds(t *testing.T) {
	fs := &fakeFileSystem{
		statExists: true,
		temporaries: []*fakeTemporaryFile{
			{name: "new.tmp"},
			{name: "backup.tmp"},
		},
		renameErrs: []error{errors.New("destination exists"), nil, nil},
	}
	if err := atomicWriteWithFS(fs, "report.docx", []byte("new"), true); err != nil {
		t.Fatalf("atomicWriteWithFS: %v", err)
	}
	if len(fs.renameCalls) != 3 {
		t.Fatalf("rename calls = %#v, want direct attempt plus backup replacement", fs.renameCalls)
	}
}

func TestOSFileSystemRemove(t *testing.T) {
	path := filepath.Join(t.TempDir(), "remove-me")
	if err := os.WriteFile(path, []byte("temporary"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := (osFileSystem{}).Remove(path); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("removed file stat error = %v", err)
	}
}

func TestPrintFailureDetailModes(t *testing.T) {
	for _, details := range []bool{false, true} {
		var output bytes.Buffer
		if code := printFailure(&output, "failed", errors.New("detail"), details); code != 1 {
			t.Fatalf("printFailure code = %d", code)
		}
		if strings.Contains(output.String(), "detail") != details {
			t.Fatalf("details=%v output=%q", details, output.String())
		}
	}
}

func syntheticTemplate(t *testing.T) []byte {
	return syntheticTemplateCommand(t, "[[value]]")
}

func syntheticTemplateCommand(t *testing.T, command string) []byte {
	t.Helper()
	parts := map[string]string{
		"[Content_Types].xml": `<?xml version="1.0"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/><Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/></Types>`,
		"_rels/.rels":         `<?xml version="1.0"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/></Relationships>`,
		"word/document.xml":   `<?xml version="1.0"?><w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body><w:p><w:r><w:t>` + command + `</w:t></w:r></w:p><w:sectPr/></w:body></w:document>`,
	}
	var output bytes.Buffer
	writer := zip.NewWriter(&output)
	for _, name := range []string{"[Content_Types].xml", "_rels/.rels", "word/document.xml"} {
		part, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := part.Write([]byte(parts[name])); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return output.Bytes()
}

func readZIPPart(t *testing.T, data []byte, name string) []byte {
	t.Helper()
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range reader.File {
		if file.Name != name {
			continue
		}
		stream, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		content, readErr := io.ReadAll(stream)
		closeErr := stream.Close()
		if readErr != nil {
			t.Fatal(readErr)
		}
		if closeErr != nil {
			t.Fatal(closeErr)
		}
		return content
	}
	t.Fatalf("ZIP part %q not found", name)
	return nil
}
