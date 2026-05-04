package prompt

import (
	"testing"
)

func TestErrNoTTY(t *testing.T) {
	t.Parallel()

	// Verify ErrNoTTY contains expected message
	expectedMsg := "interactive mode requires a TTY"
	if ErrNoTTY.Error() != expectedMsg {
		t.Errorf("ErrNoTTY.Error() = %q, want %q", ErrNoTTY.Error(), expectedMsg)
	}
}
