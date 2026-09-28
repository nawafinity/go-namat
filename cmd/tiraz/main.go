// Command tiraz provides local utilities for validating DOCX templates.
package main

import (
	"flag"
	"fmt"
	"os"
	"sort"

	"github.com/tiraz-go/tiraz"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "inspect":
		inspect(os.Args[2:])
	case "help", "-h", "--help":
		usage()
	default:
		fmt.Fprintln(os.Stderr, "tiraz: unknown command")
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: tiraz inspect [--details] template.docx")
}

func inspect(args []string) {
	flags := flag.NewFlagSet("inspect", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	details := flags.Bool("details", false, "include expression text in validation errors")
	if err := flags.Parse(args); err != nil {
		os.Exit(2)
	}
	if flags.NArg() != 1 {
		usage()
		os.Exit(2)
	}

	content, err := os.ReadFile(flags.Arg(0))
	if err != nil {
		fmt.Fprintln(os.Stderr, "tiraz: cannot read template")
		if *details {
			fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(1)
	}
	commands, err := tiraz.ListCommands(content, tiraz.Options{})
	if err != nil {
		validationFailure(err, *details)
	}
	if _, err := tiraz.Compile(content, tiraz.Options{}); err != nil {
		validationFailure(err, *details)
	}

	counts := make(map[string]int)
	for _, command := range commands {
		counts[string(command.Type)]++
	}
	types := make([]string, 0, len(counts))
	for commandType := range counts {
		types = append(types, commandType)
	}
	sort.Strings(types)

	fmt.Printf("valid DOCX template: %d commands\n", len(commands))
	for _, commandType := range types {
		fmt.Printf("%s: %d\n", commandType, counts[commandType])
	}
}

func validationFailure(err error, details bool) {
	fmt.Fprintln(os.Stderr, "tiraz: template validation failed")
	if details {
		fmt.Fprintln(os.Stderr, err)
	}
	os.Exit(1)
}
