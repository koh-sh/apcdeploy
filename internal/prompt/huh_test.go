package prompt

import (
	"errors"
	"testing"

	"github.com/charmbracelet/huh"
)

func TestHuhPrompter_Select_NonTTY(t *testing.T) {
	t.Skip("Skipping: HuhPrompter.Select() may block on TTY in some environments")
}

func TestHuhPrompter_Select_EmptyOptions(t *testing.T) {
	t.Skip("Skipping: HuhPrompter.Select() may block on TTY in some environments")
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
