package core

import (
	"fmt"
)

// CommandError is an error used to signal different error situations in command handling.
type CommandError struct {
	s         string
	userError bool
}

func (c CommandError) Error() string {
	return c.s
}

func (c CommandError) isUserError() bool {
	return c.userError
}

// NewUserError creates a new user input related error
func NewUserError(a ...interface{}) CommandError {
	return CommandError{s: fmt.Sprintln(a...), userError: true}
}

// NewSystemError creates a new system related error
func NewSystemError(a ...interface{}) CommandError {
	return CommandError{s: fmt.Sprintln(a...), userError: false}
}

// NewSystemErrorF creates a new system related error with formatting
func NewSystemErrorF(format string, a ...interface{}) CommandError {
	return CommandError{s: fmt.Sprintf(format, a...), userError: false}
}
