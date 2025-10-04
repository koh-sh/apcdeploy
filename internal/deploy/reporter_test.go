package deploy

import "testing"

func TestProgressReporterInterface(t *testing.T) {
	// Verify mockReporter implements ProgressReporter
	var _ ProgressReporter = (*mockReporter)(nil)

	reporter := &mockReporter{}

	// Test Progress
	reporter.Progress("test progress")
	if len(reporter.messages) != 1 {
		t.Errorf("expected 1 message, got %d", len(reporter.messages))
	}
	if reporter.messages[0] != "progress: test progress" {
		t.Errorf("expected 'progress: test progress', got '%s'", reporter.messages[0])
	}

	// Test Success
	reporter.Success("test success")
	if len(reporter.messages) != 2 {
		t.Errorf("expected 2 messages, got %d", len(reporter.messages))
	}
	if reporter.messages[1] != "success: test success" {
		t.Errorf("expected 'success: test success', got '%s'", reporter.messages[1])
	}

	// Test Warning
	reporter.Warning("test warning")
	if len(reporter.messages) != 3 {
		t.Errorf("expected 3 messages, got %d", len(reporter.messages))
	}
	if reporter.messages[2] != "warning: test warning" {
		t.Errorf("expected 'warning: test warning', got '%s'", reporter.messages[2])
	}
}
