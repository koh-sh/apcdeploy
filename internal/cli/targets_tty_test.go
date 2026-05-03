package cli

import (
	"bytes"
	"errors"
	"regexp"
	"strings"
	"testing"
)

// stripANSI removes ANSI escape sequences so assertions can inspect the
// underlying text. lipgloss output is full of color codes in TTY mode.
var ansiRE = regexp.MustCompile(`\x1b\[[0-9;]*[A-Za-z]`)

func stripANSI(s string) string { return ansiRE.ReplaceAllString(s, "") }

// newTestTTYTargets builds a ttyTargets bound to a buffer for assertion.
// The animation goroutine still runs; tests should call Close before
// asserting on the final state to ensure deterministic output.
func newTestTTYTargets(t *testing.T, ids []string) (*ttyTargets, *bytes.Buffer) {
	t.Helper()
	buf := &bytes.Buffer{}
	r := &Reporter{outW: &bytes.Buffer{}, errW: buf, outTTY: false, errTTY: true}
	return newTTYTargets(r, ids), buf
}

func TestTTYTargets_TerminalLines(t *testing.T) {
	t.Parallel()

	tt, buf := newTestTTYTargets(t, []string{"r/a/p/dev", "r/a/p/prod"})
	tt.Done("r/a/p/dev", "deployed (12s) — v7, AllAtOnce")
	tt.Fail("r/a/p/prod", errors.New("ConflictException"))
	tt.Close()

	out := stripANSI(buf.String())
	if !strings.Contains(out, "r/a/p/dev") {
		t.Errorf("missing identifier in output:\n%s", out)
	}
	if !strings.Contains(out, "✓ deployed (12s) — v7, AllAtOnce") {
		t.Errorf("missing done summary in output:\n%s", out)
	}
	if !strings.Contains(out, "✗ failed: ConflictException") {
		t.Errorf("missing fail line in output:\n%s", out)
	}
}

func TestTTYTargets_SkipLine(t *testing.T) {
	t.Parallel()

	tt, buf := newTestTTYTargets(t, []string{"x"})
	tt.Skip("x", "skipped (no changes)")
	tt.Close()

	out := stripANSI(buf.String())
	if !strings.Contains(out, "→ skipped (no changes)") {
		t.Errorf("missing skip line in output:\n%s", out)
	}
}

func TestTTYTargets_RunningPhaseAndProgress(t *testing.T) {
	t.Parallel()

	tt, buf := newTestTTYTargets(t, []string{"id"})
	tt.SetPhase("id", "preparing", "")
	tt.SetProgress("id", 0.5, 0)
	tt.Done("id", "complete (3s) — v1, AllAtOnce")
	tt.Close()

	out := stripANSI(buf.String())
	// Final state is "done"; intermediate phase/progress shouldn't matter
	// for terminal assertions, but we should still see the bar rendered at
	// least once at 50%.
	if !strings.Contains(out, "50%") {
		t.Errorf("missing 50%% bar in output:\n%s", out)
	}
	if !strings.Contains(out, "✓ complete (3s) — v1, AllAtOnce") {
		t.Errorf("missing done line in output:\n%s", out)
	}
}

func TestTTYTargets_TerminalSticky(t *testing.T) {
	t.Parallel()

	tt, buf := newTestTTYTargets(t, []string{"id"})
	tt.Done("id", "first")
	tt.Done("id", "second")
	tt.Fail("id", errors.New("late"))
	tt.Close()

	out := stripANSI(buf.String())
	if strings.Contains(out, "second") || strings.Contains(out, "late") {
		t.Errorf("post-terminal calls leaked output:\n%s", out)
	}
}

func TestTTYTargets_EmptyIDsCloseSafe(t *testing.T) {
	t.Parallel()

	// No identifiers — Close must not block on a goroutine that never started.
	tt, buf := newTestTTYTargets(t, nil)
	tt.Close()
	if buf.Len() != 0 {
		t.Errorf("empty Targets should produce no output, got: %q", buf.String())
	}
}

func TestTTYTargets_UnknownIDIgnored(t *testing.T) {
	t.Parallel()

	tt, _ := newTestTTYTargets(t, []string{"known"})
	tt.SetPhase("missing", "preparing", "")
	tt.SetProgress("missing", 0.5, 0)
	tt.Done("missing", "x")
	tt.Fail("missing", errors.New("e"))
	tt.Skip("missing", "r")
	// no panics, no row state for "missing"
	tt.Close()
}
