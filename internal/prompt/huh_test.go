package prompt

import (
	"testing"
)

func TestHuhPrompter_ImplementsInterface(t *testing.T) {
	t.Parallel()

	// Compile-time check that HuhPrompter implements Prompter
	var _ Prompter = (*HuhPrompter)(nil)
}
