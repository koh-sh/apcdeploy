package prompt

import "errors"

// ErrUserCancelled is returned when the user cancels the prompt
var ErrUserCancelled = errors.New("operation cancelled")

// Prompter provides an interface for prompting users for input
type Prompter interface {
	// Select displays a list of options and returns the selected value
	Select(message string, options []string) (string, error)
}
