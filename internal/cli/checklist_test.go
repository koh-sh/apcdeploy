package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/koh-sh/apcdeploy/internal/reporter"
)

func TestChecklist_ImplementsInterface(t *testing.T) {
	t.Parallel()

	var (
		_ reporter.Checklist = (*ttyChecklist)(nil)
		_ reporter.Checklist = (*nonTTYChecklist)(nil)
		_ reporter.Checklist = (*silentChecklist)(nil)
	)
}

// TestChecklist_NonTTY_OnlyEmitsCompletionLines verifies the contract that in
// non-TTY mode the checklist stays silent until each item completes — no
// pre-list, no per-item Start announcement.
func TestChecklist_NonTTY_OnlyEmitsCompletionLines(t *testing.T) {
	t.Parallel()

	r, _, errBuf := newTestReporter()
	chk := r.Checklist([]string{"phase one", "phase two"})

	chk.Start(0)
	if errBuf.Len() != 0 {
		t.Errorf("Start should be silent in non-TTY mode; got %q", errBuf.String())
	}
	chk.Done(0, "completed one")

	chk.Start(1)
	chk.Done(1, "completed two")
	chk.Close()

	got := errBuf.String()
	for _, want := range []string{"completed one", "completed two"} {
		if !strings.Contains(got, want) {
			t.Errorf("expected output to contain %q; got %q", want, got)
		}
	}
	// The labels must NOT appear because Start is silent and Done overrides
	// the label with the completion message.
	for _, unwanted := range []string{"phase one", "phase two"} {
		if strings.Contains(got, unwanted) {
			t.Errorf("non-TTY checklist must not echo pending labels; got %q", got)
		}
	}
}

// TestChecklist_NonTTY_FailIsSilent verifies the contract that Fail in
// non-TTY mode emits nothing — root.go's error display covers the failure.
func TestChecklist_NonTTY_FailIsSilent(t *testing.T) {
	t.Parallel()

	r, _, errBuf := newTestReporter()
	chk := r.Checklist([]string{"phase"})

	chk.Start(0)
	chk.Fail(0, "ignored message")
	chk.Close()

	if errBuf.Len() != 0 {
		t.Errorf("Fail should be silent in non-TTY mode; got %q", errBuf.String())
	}
}

// TestChecklist_NonTTY_SkipEmitsInfo verifies that Skip emits an Info line in
// non-TTY mode so early-exit branches (e.g. "no changes") are still visible.
func TestChecklist_NonTTY_SkipEmitsInfo(t *testing.T) {
	t.Parallel()

	r, _, errBuf := newTestReporter()
	chk := r.Checklist([]string{"phase"})

	chk.Start(0)
	chk.Skip(0, "no changes — skipping")
	chk.Close()

	if !strings.Contains(errBuf.String(), "no changes — skipping") {
		t.Errorf("expected Skip to emit info line; got %q", errBuf.String())
	}
}

// TestChecklist_NonTTY_FallsBackToLabelWhenMessageEmpty verifies that Done
// with an empty message uses the original item label.
func TestChecklist_NonTTY_FallsBackToLabelWhenMessageEmpty(t *testing.T) {
	t.Parallel()

	r, _, errBuf := newTestReporter()
	chk := r.Checklist([]string{"label only"})
	chk.Done(0, "")
	chk.Close()

	if !strings.Contains(errBuf.String(), "label only") {
		t.Errorf("expected fallback to label; got %q", errBuf.String())
	}
}

// TestChecklist_NonTTY_OutOfRangeIndexIsNoOp ensures invalid indices don't
// panic or produce output.
func TestChecklist_NonTTY_OutOfRangeIndexIsNoOp(t *testing.T) {
	t.Parallel()

	r, _, errBuf := newTestReporter()
	chk := r.Checklist([]string{"only one"})
	chk.Done(5, "should be ignored")
	chk.Done(-1, "also ignored")
	chk.Close()

	if errBuf.Len() != 0 {
		t.Errorf("out-of-range Done should be silent; got %q", errBuf.String())
	}
}

// TestChecklist_NonTTY_TransitionAfterCloseIsSilent ensures the Close →
// transition path is a no-op so deferred Close + accidental Done from a
// surrounding error path won't double-emit.
func TestChecklist_NonTTY_TransitionAfterCloseIsSilent(t *testing.T) {
	t.Parallel()

	r, _, errBuf := newTestReporter()
	chk := r.Checklist([]string{"phase"})
	chk.Close()
	chk.Done(0, "should be ignored")
	chk.Skip(0, "should be ignored")
	chk.Fail(0, "should be ignored")

	if errBuf.Len() != 0 {
		t.Errorf("transition after Close must be silent; got %q", errBuf.String())
	}
}

// TestChecklist_NonTTY_DoubleDoneIsIdempotent ensures repeated transitions on
// the same item don't double-emit.
func TestChecklist_NonTTY_DoubleDoneIsIdempotent(t *testing.T) {
	t.Parallel()

	r, _, errBuf := newTestReporter()
	chk := r.Checklist([]string{"phase"})
	chk.Done(0, "first")
	chk.Done(0, "second")
	chk.Close()

	if strings.Count(errBuf.String(), "first") != 1 {
		t.Errorf("expected exactly one 'first' line; got %q", errBuf.String())
	}
	if strings.Contains(errBuf.String(), "second") {
		t.Errorf("expected second Done to be ignored; got %q", errBuf.String())
	}
}

// TestChecklist_TTY_RendersInitialPendingBlock verifies the TTY path prints
// every item up-front so users see the full plan before work starts.
func TestChecklist_TTY_RendersInitialPendingBlock(t *testing.T) {
	t.Parallel()

	r, _, errBuf := newTTYReporter()
	chk := r.Checklist([]string{"alpha", "beta", "gamma"})
	chk.Close()

	got := errBuf.String()
	for _, label := range []string{"alpha", "beta", "gamma"} {
		if !strings.Contains(got, label) {
			t.Errorf("expected initial render to contain %q; got %q", label, got)
		}
	}
}

// TestChecklist_TTY_TransitionsRedraw verifies that state changes are
// reflected in the output. Close synchronizes the animation goroutine via
// <-c.done plus a final redraw, so we don't need a sleep for the Done message
// to land in the buffer.
func TestChecklist_TTY_TransitionsRedraw(t *testing.T) {
	t.Parallel()

	r, _, errBuf := newTTYReporter()
	chk := r.Checklist([]string{"phase"})
	chk.Start(0)
	chk.Done(0, "all done")
	chk.Close()

	if !strings.Contains(errBuf.String(), "all done") {
		t.Errorf("expected Done message in output; got %q", errBuf.String())
	}
}

// TestChecklist_TTY_FailAndSkipRender verifies that Fail and Skip transitions
// render with their respective messages (visible because the redraw rewrites
// the line).
func TestChecklist_TTY_FailAndSkipRender(t *testing.T) {
	t.Parallel()

	r, _, errBuf := newTTYReporter()
	chk := r.Checklist([]string{"a", "b"})
	chk.Fail(0, "boom")
	chk.Skip(1, "later")
	chk.Close()

	got := errBuf.String()
	if !strings.Contains(got, "boom") {
		t.Errorf("expected Fail message in TTY output; got %q", got)
	}
	if !strings.Contains(got, "later") {
		t.Errorf("expected Skip message in TTY output; got %q", got)
	}
}

// TestChecklist_TTY_DoubleCloseIsIdempotent verifies repeated Close calls are
// safe — callers commonly defer Close after also explicitly calling it.
func TestChecklist_TTY_DoubleCloseIsIdempotent(t *testing.T) {
	t.Parallel()

	r, _, _ := newTTYReporter()
	chk := r.Checklist([]string{"x"})
	chk.Close()
	chk.Close() // must not panic or block
}

// TestSilentReporter_Checklist verifies the silent variant suppresses
// everything except Fail (which forwards to Error so scripts still see fatal
// failures).
func TestSilentReporter_Checklist(t *testing.T) {
	t.Parallel()

	var errBuf bytes.Buffer
	r := &SilentReporter{outW: &bytes.Buffer{}, errW: &errBuf}
	chk := r.Checklist([]string{"a", "b"})

	chk.Start(0)
	chk.Done(0, "done a")
	chk.Skip(1, "skipped b")
	chk.Close()

	if errBuf.Len() != 0 {
		t.Errorf("silent checklist Start/Done/Skip must be silent; got %q", errBuf.String())
	}

	chk2 := r.Checklist([]string{"x"})
	chk2.Fail(0, "boom")
	if !strings.Contains(errBuf.String(), "boom") {
		t.Errorf("silent checklist Fail must forward to Error; got %q", errBuf.String())
	}
}

// TestSilentReporter_Checklist_OutOfRangeFailIsSafe ensures invalid indices
// don't panic on the silent variant.
func TestSilentReporter_Checklist_OutOfRangeFailIsSafe(t *testing.T) {
	t.Parallel()

	r := &SilentReporter{outW: &bytes.Buffer{}, errW: &bytes.Buffer{}}
	chk := r.Checklist([]string{"only"})
	chk.Fail(99, "ignored")
	chk.Fail(-1, "also ignored")
	chk.Close()
}
