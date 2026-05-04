package cli

import (
	"os"
	"testing"
)

// TestTerminalCols covers the boundaries that the Targets renderer
// relies on for its truncation guard: a nil file and a non-TTY file
// must both return 0 ("unknown — do not truncate") rather than panic.
// The TTY-positive branch needs a real PTY which the unit-test sandbox
// doesn't provide; that path is exercised in practice via the e2e
// run command and validated by TestTTYTargets_TruncatesLongFields with
// an injected cols value.
func TestTerminalCols(t *testing.T) {
	t.Parallel()

	if got := TerminalCols(nil); got != 0 {
		t.Errorf("TerminalCols(nil) = %d, want 0", got)
	}

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	t.Cleanup(func() { _ = r.Close(); _ = w.Close() })

	if got := TerminalCols(r); got != 0 {
		t.Errorf("TerminalCols(non-TTY pipe) = %d, want 0", got)
	}
}

// TestTerminalColsOf covers the io.Writer wrapper used by ttyTargets:
// a *bytes.Buffer (and any non-*os.File) must return 0 so tests that
// inject a buffer don't accidentally enable truncation based on a
// stale cols value.
func TestTerminalColsOf(t *testing.T) {
	t.Parallel()

	// terminalColsOf accepts io.Writer; a non-*os.File path returns 0.
	type stubWriter struct{}
	if got := terminalColsOf(stubWriterImpl{}); got != 0 {
		t.Errorf("terminalColsOf(non-file) = %d, want 0", got)
	}
	_ = stubWriter{}
}

type stubWriterImpl struct{}

func (stubWriterImpl) Write(p []byte) (int, error) { return len(p), nil }
