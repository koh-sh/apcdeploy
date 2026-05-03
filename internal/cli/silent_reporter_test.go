package cli

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/koh-sh/apcdeploy/internal/reporter"
)

func TestSilentReporter_ImplementsInterface(t *testing.T) {
	t.Parallel()

	var _ reporter.Reporter = (*SilentReporter)(nil)
}

func TestNewSilentReporter(t *testing.T) {
	t.Parallel()

	if NewSilentReporter() == nil {
		t.Fatal("NewSilentReporter() returned nil")
	}
}

// newTestSilentReporter wires the silent reporter to in-memory buffers so we
// can assert on what it does and does NOT emit.
func newTestSilentReporter() (*SilentReporter, *bytes.Buffer, *bytes.Buffer) {
	var out, errBuf bytes.Buffer
	r := &SilentReporter{outW: &out, errW: &errBuf}
	return r, &out, &errBuf
}

func TestSilentReporter_SuppressesHumanKinds(t *testing.T) {
	t.Parallel()

	r, out, errBuf := newTestSilentReporter()
	r.Step("a")
	r.Success("b")
	r.Info("c")
	r.Warn("d")
	r.Header("e")
	r.Box("title", []string{"line"})
	r.Table([]string{"H"}, [][]string{{"v"}})

	if out.Len() != 0 {
		t.Errorf("silent reporter wrote to stdout: %q", out.String())
	}
	if errBuf.Len() != 0 {
		t.Errorf("silent reporter wrote to stderr: %q", errBuf.String())
	}
}

func TestSilentReporter_PreservesError(t *testing.T) {
	t.Parallel()

	r, _, errBuf := newTestSilentReporter()
	r.Error("fatal")
	if !strings.Contains(errBuf.String(), "fatal") {
		t.Errorf("Error should always reach stderr; got %q", errBuf.String())
	}
}

func TestSilentReporter_PreservesStdoutPayloads(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		invoke func(r *SilentReporter)
		want   string
	}{
		{"data", func(r *SilentReporter) { r.Data([]byte("payload")) }, "payload"},
		{"diff", func(r *SilentReporter) { r.Diff([]byte("+x\n")) }, "+x\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r, out, _ := newTestSilentReporter()
			tt.invoke(r)
			if out.String() != tt.want {
				t.Errorf("stdout = %q, want %q", out.String(), tt.want)
			}
		})
	}
}

func TestSilentReporter_SpinDoneIsSilent(t *testing.T) {
	t.Parallel()

	r, _, errBuf := newTestSilentReporter()
	sp := r.Spin("starting")
	sp.Done("finished")
	if errBuf.Len() != 0 {
		t.Errorf("silent spinner Done should be silent; got %q", errBuf.String())
	}
}

func TestSilentReporter_SpinFailEmitsError(t *testing.T) {
	t.Parallel()

	r, _, errBuf := newTestSilentReporter()
	sp := r.Spin("starting")
	sp.Fail("crashed")
	if !strings.Contains(errBuf.String(), "crashed") {
		t.Errorf("silent spinner Fail should surface via Error; got %q", errBuf.String())
	}
}

func TestSilentReporter_SpinStopIsSilent(t *testing.T) {
	t.Parallel()

	r, _, errBuf := newTestSilentReporter()
	sp := r.Spin("starting")
	sp.Stop()
	if errBuf.Len() != 0 {
		t.Errorf("silent Spinner.Stop must be silent; got %q", errBuf.String())
	}

	// Done/Fail after Stop must be no-ops so scripts don't see a stray
	// Error line via Spinner.Fail.
	sp.Done("late")
	sp.Fail("late")
	if errBuf.Len() != 0 {
		t.Errorf("Done/Fail after Stop must be no-ops; got %q", errBuf.String())
	}
}

func TestSilentReporter_ProgressDoneIsSilent(t *testing.T) {
	t.Parallel()

	r, _, errBuf := newTestSilentReporter()
	pb := r.Progress("starting")
	pb.Update(50, "halfway")
	pb.Done("finished")
	if errBuf.Len() != 0 {
		t.Errorf("silent progress should be silent; got %q", errBuf.String())
	}
	// Update after Done must remain silent.
	pb.Update(99, "post-done")
	if errBuf.Len() != 0 {
		t.Errorf("silent progress Update must always be silent; got %q", errBuf.String())
	}
}

func TestSilentReporter_ProgressFailEmitsError(t *testing.T) {
	t.Parallel()

	r, _, errBuf := newTestSilentReporter()
	pb := r.Progress("starting")
	pb.Fail("crashed")
	if !strings.Contains(errBuf.String(), "crashed") {
		t.Errorf("silent progress Fail should surface via Error; got %q", errBuf.String())
	}

	// Second Fail is a no-op.
	before := errBuf.Len()
	pb.Fail("again")
	if errBuf.Len() != before {
		t.Errorf("second Fail should be a no-op; before=%d after=%d", before, errBuf.Len())
	}
}

func TestSilentReporter_ProgressStopIsSilent(t *testing.T) {
	t.Parallel()

	r, _, errBuf := newTestSilentReporter()
	pb := r.Progress("starting")
	pb.Stop()
	if errBuf.Len() != 0 {
		t.Errorf("silent progress Stop should be silent; got %q", errBuf.String())
	}

	// Subsequent Fail must NOT surface an error after Stop.
	pb.Fail("late")
	if errBuf.Len() != 0 {
		t.Errorf("Fail after Stop must be a no-op; got %q", errBuf.String())
	}
}

func TestSilentReporter_TargetsSuppressed(t *testing.T) {
	t.Parallel()

	r, out, errBuf := newTestSilentReporter()
	tg := r.Targets([]string{"id"})
	tg.SetPhase("id", "preparing", "")
	tg.SetProgress("id", 0.5, 0)
	tg.Done("id", "deployed (1s) — v1, AllAtOnce")
	tg.Skip("id", "no changes")
	tg.Close()

	if out.Len() != 0 {
		t.Errorf("Targets must not write to stdout: %q", out.String())
	}
	if errBuf.Len() != 0 {
		t.Errorf("Targets non-fail kinds must be silent: %q", errBuf.String())
	}
}

func TestSilentReporter_TargetsFailEmitsError(t *testing.T) {
	t.Parallel()

	r, _, errBuf := newTestSilentReporter()
	tg := r.Targets([]string{"id"})
	tg.Fail("id", errors.New("boom"))
	tg.Close()

	if !strings.Contains(errBuf.String(), "boom") {
		t.Errorf("Fail should surface error message; got %q", errBuf.String())
	}
}

func TestSilentReporter_TargetsFailNilErrorIsSilent(t *testing.T) {
	t.Parallel()

	r, _, errBuf := newTestSilentReporter()
	tg := r.Targets([]string{"id"})
	tg.Fail("id", nil)
	tg.Close()

	if errBuf.Len() != 0 {
		t.Errorf("Fail with nil error must be silent; got %q", errBuf.String())
	}
}
