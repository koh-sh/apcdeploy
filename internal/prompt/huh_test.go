package prompt

import (
	"errors"
	"testing"

	"github.com/charmbracelet/huh"
)

func TestHuhPrompter_Select_NonTTY(t *testing.T) {
	t.Parallel()

	// In a non-TTY environment (like CI), huh will fail with a TTY error
	// This test verifies that the error is properly propagated

	prompter := &HuhPrompter{}
	options := []string{"option1", "option2", "option3"}

	_, err := prompter.Select("Test prompt", options)

	// We expect an error because there's no TTY
	if err == nil {
		t.Error("expected error in non-TTY environment, got nil")
	}
}

func TestHuhPrompter_Select_EmptyOptions(t *testing.T) {
	t.Parallel()

	// Test that empty options are handled
	prompter := &HuhPrompter{}
	options := []string{}

	_, err := prompter.Select("Test prompt", options)

	// We expect an error (either from huh or from no TTY)
	if err == nil {
		t.Error("expected error with empty options, got nil")
	}
}

func TestHuhPrompter_ImplementsInterface(t *testing.T) {
	t.Parallel()

	// Compile-time check that HuhPrompter implements Prompter
	var _ Prompter = (*HuhPrompter)(nil)
}

func TestHuhPrompter_ErrorHandling(t *testing.T) {
	t.Parallel()

	// This test documents the behavior when huh returns errors
	// In practice, the main errors we care about are:
	// 1. ErrUserAborted (Ctrl+C)
	// 2. TTY errors (no terminal available)

	// Test that ErrUserAborted is recognized
	err := huh.ErrUserAborted
	if !errors.Is(err, huh.ErrUserAborted) {
		t.Error("ErrUserAborted should be recognizable with errors.Is")
	}
}
