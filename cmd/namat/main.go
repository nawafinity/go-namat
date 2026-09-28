// Command namat validates and renders DOCX templates locally.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/nawafinity/go-namat"
)

var version = "dev"

func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr)) }

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		usage(stderr)
		return 2
	}
	switch args[0] {
	case "inspect":
		return inspect(args[1:], stdout, stderr)
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
	fmt.Fprintln(output, "  namat metadata template.docx")
	fmt.Fprintln(output, "  namat render --data data.json --out report.docx [--force] template.docx")
	fmt.Fprintln(output, "  namat version")
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
	var data any
	if err := json.Unmarshal(dataBytes, &data); err != nil {
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
	if !replace {
		if _, err := os.Stat(name); err == nil {
			return fmt.Errorf("destination already exists")
		} else if !os.IsNotExist(err) {
			return err
		}
	}
	directory := filepath.Dir(name)
	temporary, err := os.CreateTemp(directory, ".namat-*.tmp")
	if err != nil {
		return err
	}
	temporaryName := temporary.Name()
	committed := false
	defer func() {
		_ = temporary.Close()
		if !committed {
			_ = os.Remove(temporaryName)
		}
	}()
	if _, err := temporary.Write(data); err != nil {
		return err
	}
	if err := temporary.Sync(); err != nil {
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if replace {
		if err := os.Remove(name); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	if err := os.Rename(temporaryName, name); err != nil {
		return err
	}
	committed = true
	return nil
}

func printFailure(output io.Writer, message string, err error, details bool) int {
	fmt.Fprintln(output, "namat:", message)
	if details {
		fmt.Fprintln(output, err)
	}
	return 1
}
