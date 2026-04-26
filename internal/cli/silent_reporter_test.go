package cli

import (
	"bytes"
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
