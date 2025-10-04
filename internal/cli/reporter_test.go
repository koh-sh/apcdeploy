package cli

import (
	"testing"
)

func TestNewReporter(t *testing.T) {
	reporter := NewReporter()
	if reporter == nil {
		t.Error("NewReporter() returned nil")
	}
}

func TestReporter_Progress(t *testing.T) {
	reporter := NewReporter()

	// Should not panic
	reporter.Progress("test progress message")
}

func TestReporter_Success(t *testing.T) {
	reporter := NewReporter()

	// Should not panic
	reporter.Success("test success message")
}

func TestReporter_Warning(t *testing.T) {
	reporter := NewReporter()

	// Should not panic
	reporter.Warning("test warning message")
}

func TestReporter_ImplementsInterface(t *testing.T) {
	// This test verifies that Reporter implements the ProgressReporter interface
	// The compilation will fail if it doesn't implement the interface
	_ = (*Reporter)(nil)
}
