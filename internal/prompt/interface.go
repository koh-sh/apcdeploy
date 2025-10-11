package prompt

// Prompter provides an interface for prompting users for input
type Prompter interface {
	// Select displays a list of options and returns the selected value
	Select(message string, options []string) (string, error)
}
