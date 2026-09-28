package engine

import (
	"errors"
	"fmt"
	"strings"
)

var (
	// ErrInvalidTemplate identifies an invalid DOCX package or template shape.
	ErrInvalidTemplate = errors.New("invalid template")
	// ErrCommandSyntax identifies a malformed command or expression.
	ErrCommandSyntax = errors.New("command syntax error")
	// ErrCommandExecution identifies a command that failed during rendering.
	ErrCommandExecution = errors.New("command execution error")
	// ErrNullishResult identifies a rejected nil command result.
	ErrNullishResult = errors.New("nullish command result")
	// ErrObjectResult identifies an object accidentally used as text.
	ErrObjectResult = errors.New("object command result")
	// ErrSecurityLimit identifies a configured resource or safety limit.
	ErrSecurityLimit = errors.New("security limit exceeded")
)

// Error describes a template failure and the package part where it occurred.
type Error struct {
	Part    string
	Command string
	Err     error
}

func (e *Error) Error() string {
	switch {
	case e.Part != "" && e.Command != "":
		return fmt.Sprintf("namat: %s: command %q: %v", e.Part, e.Command, e.Err)
	case e.Part != "":
		return fmt.Sprintf("namat: %s: %v", e.Part, e.Err)
	default:
		return fmt.Sprintf("namat: %v", e.Err)
	}
}

func (e *Error) Unwrap() error { return e.Err }

// MultiError contains independent validation errors found when CollectErrors
// is enabled.
type MultiError struct {
	Errors []error
}

func (e *MultiError) Error() string {
	if e == nil || len(e.Errors) == 0 {
		return "namat: no errors"
	}
	parts := make([]string, len(e.Errors))
	for index, err := range e.Errors {
		parts[index] = err.Error()
	}
	return fmt.Sprintf("namat: %d validation errors: %s", len(parts), strings.Join(parts, "; "))
}

// Unwrap allows errors.Is and errors.As to inspect every contained error.
func (e *MultiError) Unwrap() []error {
	if e == nil {
		return nil
	}
	return e.Errors
}
