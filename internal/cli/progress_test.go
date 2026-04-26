package cli

import (
	"strings"
	"testing"
	"time"
)

func TestReporter_ProgressNonTTY_EmitsThresholdSteps(t *testing.T) {
	t.Parallel()

	r, _, errBuf := newTestReporter()
	pb := r.Progress("Deploying")
	pb.Update(10, "Deploying") // below 25, no Step
	pb.Update(30, "Deploying") // crosses 25
	pb.Update(55, "Deploying") // crosses 50
	pb.Update(80, "Deploying") // crosses 75
	pb.Update(100, "Baking")   // crosses 100
	pb.Done("complete")

	got := errBuf.String()
	for _, want := range []string{"Deploying", "complete"} {
		if !strings.Contains(got, want) {
			t.Errorf("non-TTY Progress missing %q; got %q", want, got)
		}
	}
	for _, threshold := range []string{"25%", "50%", "75%", "100%"} {
		if !strings.Contains(got, threshold) {
			t.Errorf("non-TTY Progress missing threshold %q; got %q", threshold, got)
		}
	}
}

func TestReporter_ProgressNonTTY_FailEmitsError(t *testing.T) {
	t.Parallel()

	r, _, errBuf := newTestReporter()
	pb := r.Progress("Deploying")
	pb.Fail("oops")

	if !strings.Contains(errBuf.String(), "oops") {
		t.Errorf("Progress.Fail should emit an Error line; got %q", errBuf.String())
	}
}

func TestReporter_ProgressTTYAnimatesAndCleansUp(t *testing.T) {
	t.Parallel()

	r, _, errBuf := newTTYReporter()
	pb := r.Progress("Deploying")
	pb.Update(40, "Deploying")
	time.Sleep(150 * time.Millisecond)
	pb.Done("done")

	got := errBuf.String()
	if !strings.Contains(got, "Deploying") {
		t.Errorf("expected Progress to render label at least once; got %q", got)
	}
	if !strings.Contains(got, "done") {
		t.Errorf("expected Done() to print success line; got %q", got)
	}
	if !strings.Contains(got, "40.0%") {
		t.Errorf("expected percentage rendered; got %q", got)
	}
}

func TestReporter_ProgressDoubleDoneIsIdempotent(t *testing.T) {
	t.Parallel()

	r, _, errBuf := newTestReporter()
	pb := r.Progress("loading")
	pb.Done("done")
	before := errBuf.Len()
	pb.Done("again")
	if errBuf.Len() != before {
		t.Errorf("second Done should be a no-op; before=%d after=%d", before, errBuf.Len())
	}
}

// TestReporter_ProgressStopEmitsNoLine guards against the regression where
// failure paths called Fail with a duplicate of the propagated error message,
// causing two error lines for one failure. Stop must terminate the bar
// without emitting Step/Success/Error lines so the top-level error formatter
// in cmd/root.go owns the user-facing error line.
func TestReporter_ProgressStopEmitsNoLine(t *testing.T) {
	t.Parallel()

	r, _, errBuf := newTestReporter()
	pb := r.Progress("Deploying")
	// Drain the initial "Step" line emitted on construction in non-TTY mode
	// so we can assert that Stop adds nothing further.
	before := errBuf.Len()
	pb.Stop()
	if errBuf.Len() != before {
		t.Errorf("Stop should not emit any line; before=%d after=%d (added %q)",
			before, errBuf.Len(), errBuf.String()[before:])
	}

	// Subsequent Done/Fail/Stop must all be no-ops.
	pb.Done("late done")
	pb.Fail("late fail")
	pb.Stop()
	if errBuf.Len() != before {
		t.Errorf("post-Stop terminators must be no-ops; got %q", errBuf.String()[before:])
	}
}

func TestReporter_ProgressUpdateClampsOutOfRange(t *testing.T) {
	t.Parallel()

	// Negative and >100 percentages must not produce stray threshold lines
	// (negative would otherwise yield negative thresholds; >100 would emit a
	// duplicate 100% line each call).
	r, _, errBuf := newTestReporter()
	pb := r.Progress("Deploying")
	pb.Update(-50, "neg")
	pb.Update(150, "over")
	pb.Update(150, "over again")
	pb.Done("complete")

	got := errBuf.String()
	// Exactly one 100% threshold line should appear despite multiple >100 updates.
	if strings.Count(got, "100%") != 1 {
		t.Errorf("expected exactly one 100%% threshold line; got %q", got)
	}
	if strings.Contains(got, "-") && strings.Contains(got, "%") {
		// A naive %d format of a negative threshold would produce e.g. "-25%".
		if strings.Contains(got, "-25%") || strings.Contains(got, "-50%") {
			t.Errorf("negative threshold leaked into output: %q", got)
		}
	}
}
