package cli

import (
	"testing"

	"github.com/koh-sh/apcdeploy/internal/reporter"
)

func TestSilentReporter_ImplementsInterface(t *testing.T) {
	t.Parallel()
	var _ reporter.ProgressReporter = (*SilentReporter)(nil)
}

func TestNewSilentReporter(t *testing.T) {
	t.Parallel()
	r := NewSilentReporter()
	if r == nil {
		t.Fatal("NewSilentReporter() returned nil")
	}
}

func TestSilentReporter_Methods(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		method func(*SilentReporter, string)
	}{
		{
			name: "Progress should not panic",
			method: func(r *SilentReporter, msg string) {
				r.Progress(msg)
			},
		},
		{
			name: "Success should not panic",
			method: func(r *SilentReporter, msg string) {
				r.Success(msg)
			},
		},
		{
			name: "Warning should not panic",
			method: func(r *SilentReporter, msg string) {
				r.Warning(msg)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := NewSilentReporter()
			// Should not panic when called
			tt.method(r, "test message")
		})
	}
}
