package cli

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"
)

// newTestPlainTargets builds a plainTargets bound to a buffer for assertion.
func newTestPlainTargets(t *testing.T, ids []string) (*plainTargets, *bytes.Buffer) {
	t.Helper()
	buf := &bytes.Buffer{}
	r := &Reporter{outW: &bytes.Buffer{}, errW: buf, outTTY: false, errTTY: false}
	return newPlainTargets(r, ids), buf
}

func TestPlainTargets_PhaseTransitions(t *testing.T) {
	t.Parallel()

	pt, buf := newTestPlainTargets(t, []string{"r/a/p/dev"})
	defer pt.Close()

	pt.SetPhase("r/a/p/dev", "preparing", "")
	pt.SetPhase("r/a/p/dev", "preparing", "") // duplicate suppressed
	pt.SetPhase("r/a/p/dev", "comparing", "")

	out := buf.String()
	if !strings.Contains(out, "r/a/p/dev: preparing\n") {
		t.Errorf("missing preparing line: %q", out)
	}
	if !strings.Contains(out, "r/a/p/dev: comparing\n") {
		t.Errorf("missing comparing line: %q", out)
	}
	if got := strings.Count(out, "preparing"); got != 1 {
		t.Errorf("expected exactly 1 'preparing' line, got %d in %q", got, out)
	}
}

func TestPlainTargets_ProgressDecimation(t *testing.T) {
	t.Parallel()

	pt, buf := newTestPlainTargets(t, []string{"id"})
	defer pt.Close()

	for _, p := range []float64{0.1, 0.25, 0.30, 0.5, 0.6, 0.75, 0.99, 1.0} {
		pt.SetProgress("id", p, 0)
	}

	out := buf.String()
	for _, want := range []string{"id: deploying 25%", "id: deploying 50%", "id: deploying 75%", "id: deploying 100%"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing line %q in:\n%s", want, out)
		}
	}
	// 0.1 should not produce a 0% line.
	if strings.Contains(out, "deploying 0%") {
		t.Errorf("unexpected 0%% line in: %s", out)
	}
}

func TestPlainTargets_TerminalStates(t *testing.T) {
	t.Parallel()

	pt, buf := newTestPlainTargets(t, []string{"a", "b", "c"})
	defer pt.Close()

	pt.Done("a", "deployed (12s) — v7, AllAtOnce")
	pt.Fail("b", errors.New("boom"))
	pt.Skip("c", "skipped (no changes)")

	out := buf.String()
	if !strings.Contains(out, "a: ✓ deployed (12s) — v7, AllAtOnce\n") {
		t.Errorf("missing done line: %q", out)
	}
	if !strings.Contains(out, "b: ✗ failed: boom\n") {
		t.Errorf("missing fail line: %q", out)
	}
	if !strings.Contains(out, "c: → skipped (no changes)\n") {
		t.Errorf("missing skip line: %q", out)
	}
}

func TestPlainTargets_TerminalStateSticky(t *testing.T) {
	t.Parallel()

	pt, buf := newTestPlainTargets(t, []string{"id"})
	defer pt.Close()

	pt.Done("id", "first")
	pt.Done("id", "second")              // ignored
	pt.SetPhase("id", "deploying", "")   // ignored
	pt.SetProgress("id", 0.5, 0)         // ignored
	pt.Fail("id", errors.New("ignored")) // ignored

	out := buf.String()
	if got := strings.Count(out, "✓"); got != 1 {
		t.Errorf("expected exactly 1 success line, got %d in %q", got, out)
	}
	if strings.Contains(out, "second") || strings.Contains(out, "deploying") || strings.Contains(out, "ignored") {
		t.Errorf("post-terminal calls leaked output: %q", out)
	}
}

func TestPlainTargets_UnknownIDIgnored(t *testing.T) {
	t.Parallel()

	pt, buf := newTestPlainTargets(t, []string{"known"})
	defer pt.Close()

	pt.SetPhase("missing", "preparing", "")
	pt.SetProgress("missing", 0.5, 0)
	pt.Done("missing", "x")
	pt.Fail("missing", errors.New("e"))
	pt.Skip("missing", "r")

	if buf.Len() != 0 {
		t.Errorf("unknown id should not emit output, got: %q", buf.String())
	}
}

func TestPlainTargets_EtaIgnoredInSetProgress(t *testing.T) {
	t.Parallel()

	// SetProgress's eta is intentionally ignored in non-TTY mode (CI logs
	// stay clean — output.md §6.2 keeps lines to the threshold percent only).
	pt, buf := newTestPlainTargets(t, []string{"id"})
	defer pt.Close()

	pt.SetProgress("id", 0.5, 5*time.Minute)

	out := buf.String()
	if strings.Contains(out, "min left") {
		t.Errorf("eta should not appear in non-TTY output: %q", out)
	}
}
