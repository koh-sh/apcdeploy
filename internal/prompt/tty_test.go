package prompt

import (
	"errors"
	"testing"
)

func TestCheckTTY(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		wantErr bool
	}{
		{
			name: "check TTY availability",
			// In CI/test environments, stdin is typically not a TTY
			// We expect an error in most test scenarios
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := CheckTTY()
			if tt.wantErr && err == nil {
				t.Error("CheckTTY() expected error but got nil")
			}
			if tt.wantErr && err != nil && !errors.Is(err, ErrNoTTY) {
				t.Errorf("CheckTTY() error = %v, want ErrNoTTY", err)
			}
		})
	}
}

func TestErrNoTTY(t *testing.T) {
	t.Parallel()

	// Verify ErrNoTTY contains expected message
	expectedMsg := "interactive mode requires a TTY"
	if ErrNoTTY.Error() != expectedMsg {
		t.Errorf("ErrNoTTY.Error() = %q, want %q", ErrNoTTY.Error(), expectedMsg)
	}
}
