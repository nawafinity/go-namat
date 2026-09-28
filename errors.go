package namat

import "fmt"

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
