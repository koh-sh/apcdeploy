package cli

import (
	"testing"

	"github.com/koh-sh/apcdeploy/internal/reporter"
)

func TestNewReporter(t *testing.T) {
	t.Parallel()

	reporter := NewReporter()
	if reporter == nil {
		t.Error("NewReporter() returned nil")
	}
}

func TestReporter_Methods(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		method  func(*Reporter, string)
		message string
	}{
		{
			name: "Progress should not panic",
			method: func(r *Reporter, msg string) {
				r.Progress(msg)
			},
			message: "test progress message",
		},
		{
			name: "Success should not panic",
			method: func(r *Reporter, msg string) {
				r.Success(msg)
			},
			message: "test success message",
		},
		{
			name: "Warning should not panic",
			method: func(r *Reporter, msg string) {
				r.Warning(msg)
			},
			message: "test warning message",
		},
		{
			name: "Progress with empty message",
			method: func(r *Reporter, msg string) {
				r.Progress(msg)
			},
			message: "",
		},
		{
			name: "Success with empty message",
			method: func(r *Reporter, msg string) {
				r.Success(msg)
			},
			message: "",
		},
		{
			name: "Warning with empty message",
			method: func(r *Reporter, msg string) {
				r.Warning(msg)
			},
			message: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := NewReporter()
			// Should not panic when called
			tt.method(r, tt.message)
		})
	}
}

func TestReporter_ImplementsInterface(t *testing.T) {
	t.Parallel()

	// This test verifies that Reporter implements the ProgressReporter interface
	// The compilation will fail if it doesn't implement the interface
	var _ reporter.ProgressReporter = (*Reporter)(nil)
}
