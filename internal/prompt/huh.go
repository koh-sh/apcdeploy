package prompt

import (
	"errors"

	"charm.land/huh/v2"
	"github.com/koh-sh/apcdeploy/internal/sanitize"
)

// HuhPrompter implements Prompter using huh library
type HuhPrompter struct{}

// Ensure HuhPrompter implements the interface
var _ Prompter = (*HuhPrompter)(nil)

func (h *HuhPrompter) Select(message string, options []string) (string, error) {
	var result string
	err := huh.NewSelect[string]().
		Title(message).
		Description("Use ↑/↓ to navigate, / to filter, Enter to select").
		Options(selectOptions(options)...).
		Value(&result).
		Run()
	if err != nil {
		// Handle cancellation (Ctrl+C) and convert to our standard error
		if errors.Is(err, huh.ErrUserAborted) {
			return "", ErrUserCancelled
		}
		return "", err
	}

	return result, nil
}

// selectOptions builds huh options from the given strings. The options are
// remote AppConfig names that other principals can set, and huh writes labels
// to the terminal verbatim (in accessible mode, enabled by TERM=dumb, straight
// to stdout), so each label is neutralised. The value stays the exact name for
// the exact-match resolver.
func selectOptions(options []string) []huh.Option[string] {
	huhOptions := make([]huh.Option[string], len(options))
	for i, opt := range options {
		huhOptions[i] = huh.NewOption(sanitize.Line(opt), opt)
	}
	return huhOptions
}

func (h *HuhPrompter) Input(message string, placeholder string) (string, error) {
	var result string
	err := huh.NewInput().
		Title(message).
		Placeholder(placeholder).
		Value(&result).
		Run()
	if err != nil {
		// Handle cancellation (Ctrl+C) and convert to our standard error
		if errors.Is(err, huh.ErrUserAborted) {
			return "", ErrUserCancelled
		}
		return "", err
	}

	return result, nil
}

// CheckTTY verifies that stdin is connected to a terminal
func (h *HuhPrompter) CheckTTY() error {
	return CheckTTY()
}
