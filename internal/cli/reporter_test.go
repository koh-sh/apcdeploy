package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/koh-sh/apcdeploy/internal/reporter"
)

func TestNewReporter(t *testing.T) {
	t.Parallel()

	if NewReporter() == nil {
		t.Error("NewReporter() returned nil")
	}
}

func TestReporter_ImplementsInterface(t *testing.T) {
	t.Parallel()

	var _ reporter.Reporter = (*Reporter)(nil)
}

// newTestReporter builds a Reporter that writes to in-memory buffers and
// reports both streams as non-TTY, so output is plain-text and predictable.
func newTestReporter() (*Reporter, *bytes.Buffer, *bytes.Buffer) {
	var out, errBuf bytes.Buffer
	r := &Reporter{
		outW:   &out,
		errW:   &errBuf,
		outTTY: false,
		errTTY: false,
	}
	return r, &out, &errBuf
}

func TestReporter_StderrKinds(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		invoke func(r *Reporter)
		want   string
	}{
		{"step", func(r *Reporter) { r.Step("starting") }, "starting"},
		{"success", func(r *Reporter) { r.Success("done") }, "done"},
		{"info", func(r *Reporter) { r.Info("note") }, "note"},
		{"warn", func(r *Reporter) { r.Warn("careful") }, "careful"},
		{"error", func(r *Reporter) { r.Error("boom") }, "boom"},
		{"header", func(r *Reporter) { r.Header("Title") }, "Title"},
		{"box", func(r *Reporter) { r.Box("T", []string{"line1", "line2"}) }, "line1"},
		{"table", func(r *Reporter) { r.Table([]string{"A"}, [][]string{{"v"}}) }, "v"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r, out, err := newTestReporter()
			tt.invoke(r)
			if out.Len() != 0 {
				t.Errorf("stderr-only kind unexpectedly wrote to stdout: %q", out.String())
			}
			if !strings.Contains(err.String(), tt.want) {
				t.Errorf("stderr = %q, want to contain %q", err.String(), tt.want)
			}
		})
	}
}

func TestReporter_StdoutKinds(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		invoke func(r *Reporter)
		want   string
	}{
		{"data", func(r *Reporter) { r.Data([]byte("payload")) }, "payload"},
		{"diff", func(r *Reporter) { r.Diff([]byte("+added\n-removed\n")) }, "+added"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r, out, err := newTestReporter()
			tt.invoke(r)
			if err.Len() != 0 {
				t.Errorf("stdout-only kind unexpectedly wrote to stderr: %q", err.String())
			}
			if !strings.Contains(out.String(), tt.want) {
				t.Errorf("stdout = %q, want to contain %q", out.String(), tt.want)
			}
		})
	}
}

func TestReporter_SpinNonTTY_DoneEmitsSuccess(t *testing.T) {
	t.Parallel()

	r, _, errBuf := newTestReporter()
	sp := r.Spin("loading")
	sp.Done("loaded")

	got := errBuf.String()
	if !strings.Contains(got, "loading") {
		t.Errorf("Spin should emit a Step line in non-TTY mode; got %q", got)
	}
	if !strings.Contains(got, "loaded") {
		t.Errorf("Spinner.Done should emit a Success line; got %q", got)
	}
}

func TestReporter_SpinNonTTY_FailEmitsError(t *testing.T) {
	t.Parallel()

	r, _, errBuf := newTestReporter()
	sp := r.Spin("loading")
	sp.Fail("oops")

	got := errBuf.String()
	if !strings.Contains(got, "oops") {
		t.Errorf("Spinner.Fail should emit an Error line; got %q", got)
	}
}

func TestReporter_DoubleDoneIsIdempotent(t *testing.T) {
	t.Parallel()

	r, _, errBuf := newTestReporter()
	sp := r.Spin("loading")
	sp.Done("loaded")
	before := errBuf.Len()
	sp.Done("again")
	if errBuf.Len() != before {
		t.Errorf("second Done should be a no-op; before=%d after=%d", before, errBuf.Len())
	}
}
