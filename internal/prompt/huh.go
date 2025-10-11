package prompt

import (
	"errors"

	"github.com/charmbracelet/huh"
)

// HuhPrompter implements Prompter using huh library
type HuhPrompter struct{}

// Ensure HuhPrompter implements the interface
var _ Prompter = (*HuhPrompter)(nil)

func (h *HuhPrompter) Select(message string, options []string) (string, error) {
	// Build huh options from string slice
	huhOptions := make([]huh.Option[string], len(options))
	for i, opt := range options {
		huhOptions[i] = huh.NewOption(opt, opt)
	}

	var result string
	err := huh.NewSelect[string]().
		Title(message).
		Description("Use ↑/↓ to navigate, / to filter, Enter to select").
		Options(huhOptions...).
		Value(&result).
		Run()
	if err != nil {
		// Handle cancellation (Ctrl+C)
		if errors.Is(err, huh.ErrUserAborted) {
			return "", err
		}
		return "", err
	}

	return result, nil
}
