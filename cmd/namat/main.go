// Command namat validates and renders DOCX templates locally.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/nawafinity/go-namat"
)

var version = "dev"
var exit = os.Exit

func main() { exit(run(os.Args[1:], os.Stdout, os.Stderr)) }

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		usage(stderr)
		return 2
	}
	switch args[0] {
	case "inspect":
		return inspect(args[1:], stdout, stderr)
	case "lint":
		return lint(args[1:], stdout, stderr)
	case "metadata":
		return metadata(args[1:], stdout, stderr)
	case "render":
		return render(args[1:], stdout, stderr)
	case "version", "--version", "-version":
		fmt.Fprintln(stdout, version)
		return 0
	case "help", "-h", "--help":
		usage(stdout)
		return 0
	default:
		fmt.Fprintln(stderr, "namat: unknown command")
		usage(stderr)
		return 2
	}
}

func usage(output io.Writer) {
	fmt.Fprintln(output, "usage:")
	fmt.Fprintln(output, "  namat inspect [--details] [--json] template.docx")
	fmt.Fprintln(output, "  namat lint [--data data.json] [--json] template.docx")
	fmt.Fprintln(output, "  namat metadata template.docx")
	fmt.Fprintln(output, "  namat render --data data.json --out report.docx [--force] template.docx")
	fmt.Fprintln(output, "  namat version")
}

func lint(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("lint", flag.ContinueOnError)
	flags.SetOutput(stderr)
	dataPath := flags.String("data", "", "optional JSON sample used to validate field and type behavior")
	jsonOutput := flags.Bool("json", false, "write the lint result as JSON")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if flags.NArg() != 1 {
		usage(stderr)
		return 2
	}
	templateBytes, err := os.ReadFile(flags.Arg(0))
	if err != nil {
		return printFailure(stderr, "cannot read template", err, true)
	}
	compiled, err := namat.Compile(templateBytes, namat.Options{CollectErrors: true})
	if err != nil {
		return printFailure(stderr, "template validation failed", err, true)
	}
	dataValidated := false
	if *dataPath != "" {
		dataBytes, readErr := os.ReadFile(*dataPath)
		if readErr != nil {
			return printFailure(stderr, "cannot read data", readErr, true)
		}
		data, decodeErr := namat.DecodeJSON(bytes.NewReader(dataBytes))
		if decodeErr != nil {
			return printFailure(stderr, "invalid JSON data", decodeErr, true)
		}
		if _, renderErr := compiled.Render(context.Background(), data); renderErr != nil {
			return printFailure(stderr, "data validation failed", renderErr, true)
		}
		dataValidated = true
	}
	if *jsonOutput {
		payload := struct {
			Valid         bool `json:"valid"`
			Commands      int  `json:"commands"`
			DataValidated bool `json:"dataValidated"`
		}{Valid: true, Commands: len(compiled.Commands()), DataValidated: dataValidated}
		if err := json.NewEncoder(stdout).Encode(payload); err != nil {
			fmt.Fprintln(stderr, "namat: write output:", err)
			return 1
		}
		return 0
	}
	if dataValidated {
		fmt.Fprintf(stdout, "valid v1 template and data: %d commands\n", len(compiled.Commands()))
	} else {
		fmt.Fprintf(stdout, "valid v1 template: %d commands\n", len(compiled.Commands()))
	}
	return 0
}

func inspect(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("inspect", flag.ContinueOnError)
	flags.SetOutput(stderr)
	details := flags.Bool("details", false, "include expression text in validation errors")
	jsonOutput := flags.Bool("json", false, "write command counts as JSON")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if flags.NArg() != 1 {
		usage(stderr)
		return 2
	}
	content, err := os.ReadFile(flags.Arg(0))
	if err != nil {
		return printFailure(stderr, "cannot read template", err, *details)
	}
	commands, err := namat.ListCommands(content, namat.Options{})
	if err != nil {
		return printFailure(stderr, "template validation failed", err, *details)
	}
	if _, err := namat.Compile(content, namat.Options{CollectErrors: true}); err != nil {
		return printFailure(stderr, "template validation failed", err, *details)
	}
	counts := make(map[string]int)
	for _, command := range commands {
		counts[string(command.Type)]++
	}
	if *jsonOutput {
		payload := struct {
			Valid    bool           `json:"valid"`
			Commands int            `json:"commands"`
			ByType   map[string]int `json:"byType"`
		}{Valid: true, Commands: len(commands), ByType: counts}
		if err := json.NewEncoder(stdout).Encode(payload); err != nil {
			fmt.Fprintln(stderr, "namat: write output:", err)
			return 1
		}
		return 0
	}
	types := make([]string, 0, len(counts))
	for commandType := range counts {
		types = append(types, commandType)
	}
	sort.Strings(types)
	fmt.Fprintf(stdout, "valid DOCX template: %d commands\n", len(commands))
	for _, commandType := range types {
		fmt.Fprintf(stdout, "%s: %d\n", commandType, counts[commandType])
	}
	return 0
}

func metadata(args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		usage(stderr)
		return 2
	}
	content, err := os.ReadFile(args[0])
	if err != nil {
		fmt.Fprintln(stderr, "namat: cannot read document:", err)
		return 1
	}
	value, err := namat.GetMetadata(content)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	encoder := json.NewEncoder(stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		fmt.Fprintln(stderr, "namat: write output:", err)
		return 1
	}
	return 0
}

func render(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("render", flag.ContinueOnError)
	flags.SetOutput(stderr)
	dataPath := flags.String("data", "", "JSON data file")
	outputPath := flags.String("out", "", "output DOCX file")
	force := flags.Bool("force", false, "replace an existing output file")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if flags.NArg() != 1 || *dataPath == "" || *outputPath == "" {
		usage(stderr)
		return 2
	}
	if !*force {
		if _, err := os.Stat(*outputPath); err == nil {
			fmt.Fprintln(stderr, "namat: output already exists; pass --force to replace it")
			return 1
		} else if !os.IsNotExist(err) {
			fmt.Fprintln(stderr, "namat: inspect output path:", err)
			return 1
		}
	}
	template, err := os.ReadFile(flags.Arg(0))
	if err != nil {
		fmt.Fprintln(stderr, "namat: cannot read template:", err)
		return 1
	}
	dataBytes, err := os.ReadFile(*dataPath)
	if err != nil {
		fmt.Fprintln(stderr, "namat: cannot read data:", err)
		return 1
	}
	data, err := namat.DecodeJSON(bytes.NewReader(dataBytes))
	if err != nil {
		fmt.Fprintln(stderr, "namat: invalid JSON data:", err)
		return 1
	}
	report, err := namat.CreateReport(context.Background(), template, data, namat.Options{})
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if err := atomicWrite(*outputPath, report, *force); err != nil {
		fmt.Fprintln(stderr, "namat: write report:", err)
		return 1
	}
	fmt.Fprintln(stdout, *outputPath)
	return 0
}

func atomicWrite(name string, data []byte, replace bool) error {
	return atomicWriteWithFS(osFileSystem{}, name, data, replace)
}

type temporaryFile interface {
	io.Writer
	Name() string
	Sync() error
	Close() error
}

type fileSystem interface {
	Stat(string) (os.FileInfo, error)
	CreateTemp(string, string) (temporaryFile, error)
	Link(string, string) error
	Remove(string) error
	Rename(string, string) error
}

type osFileSystem struct{}

func (osFileSystem) Stat(name string) (os.FileInfo, error) { return os.Stat(name) }
func (osFileSystem) CreateTemp(directory, pattern string) (temporaryFile, error) {
	return os.CreateTemp(directory, pattern)
}
func (osFileSystem) Link(oldName, newName string) error   { return os.Link(oldName, newName) }
func (osFileSystem) Remove(name string) error             { return os.Remove(name) }
func (osFileSystem) Rename(oldName, newName string) error { return os.Rename(oldName, newName) }

func atomicWriteWithFS(fs fileSystem, name string, data []byte, replace bool) error {
	if !replace {
		if _, err := fs.Stat(name); err == nil {
			return fmt.Errorf("destination already exists")
		} else if !os.IsNotExist(err) {
			return err
		}
	}
	directory := filepath.Dir(name)
	temporary, err := fs.CreateTemp(directory, ".namat-*.tmp")
	if err != nil {
		return err
	}
	temporaryName := temporary.Name()
	committed := false
	defer func() {
		_ = temporary.Close()
		if !committed {
			_ = fs.Remove(temporaryName)
		}
	}()
	written, err := temporary.Write(data)
	if err != nil {
		return err
	}
	if written != len(data) {
		return io.ErrShortWrite
	}
	if err := temporary.Sync(); err != nil {
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if !replace {
		// Linking a sibling temporary file creates the destination atomically and
		// fails if another process created it after the initial Stat check. A
		// plain Rename would overwrite that racing file on Unix.
		if err := fs.Link(temporaryName, name); err != nil {
			return err
		}
		if err := fs.Remove(temporaryName); err != nil {
			return fmt.Errorf("report created but cannot remove temporary file %q: %w", temporaryName, err)
		}
		committed = true
		return nil
	}

	// os.Rename replaces an existing destination atomically on Unix. Windows
	// may reject that operation, so try the atomic path first and use a sibling
	// backup only when the destination still exists.
	directErr := fs.Rename(temporaryName, name)
	if directErr == nil {
		committed = true
		return nil
	}
	if _, statErr := fs.Stat(name); statErr != nil {
		if os.IsNotExist(statErr) {
			return directErr
		}
		return errors.Join(directErr, statErr)
	}

	backup, err := fs.CreateTemp(directory, ".namat-backup-*.tmp")
	if err != nil {
		return errors.Join(directErr, err)
	}
	backupName := backup.Name()
	if err := backup.Close(); err != nil {
		_ = fs.Remove(backupName)
		return errors.Join(directErr, err)
	}
	if err := fs.Remove(backupName); err != nil {
		return errors.Join(directErr, err)
	}
	if err := fs.Rename(name, backupName); err != nil {
		return errors.Join(directErr, err)
	}
	if err := fs.Rename(temporaryName, name); err != nil {
		if restoreErr := fs.Rename(backupName, name); restoreErr != nil {
			return errors.Join(err, fmt.Errorf("original output remains at %q: %w", backupName, restoreErr))
		}
		return err
	}
	committed = true
	if err := fs.Remove(backupName); err != nil {
		return fmt.Errorf("report replaced but cannot remove backup %q: %w", backupName, err)
	}
	return nil
}

func printFailure(output io.Writer, message string, err error, details bool) int {
	fmt.Fprintln(output, "namat:", message)
	if details {
		fmt.Fprintln(output, err)
	}
	return 1
}
