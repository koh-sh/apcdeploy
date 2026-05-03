package cli

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/koh-sh/apcdeploy/internal/reporter"
)

// SilentReporter is the --silent variant of Reporter. It suppresses every
// human-facing kind (Step / Success / Info / Warn / Header / Box / Table /
// Spin) and only forwards Error to stderr and Data / Diff to stdout, so
// scripts still receive errors and payloads.
type SilentReporter struct {
	outW io.Writer
	errW io.Writer
}

var _ reporter.Reporter = (*SilentReporter)(nil)

// NewSilentReporter constructs a SilentReporter bound to os.Stdout / os.Stderr.
func NewSilentReporter() *SilentReporter {
	return &SilentReporter{
		outW: os.Stdout,
		errW: os.Stderr,
	}
}

func (r *SilentReporter) Step(string)    {}
func (r *SilentReporter) Success(string) {}
func (r *SilentReporter) Info(string)    {}
func (r *SilentReporter) Warn(string)    {}

// Error is the one stderr kind that is preserved in silent mode.
func (r *SilentReporter) Error(msg string) {
	fmt.Fprintf(r.errW, "%s %s\n", symError, msg)
}

func (r *SilentReporter) Header(string)              {}
func (r *SilentReporter) Box(string, []string)       {}
func (r *SilentReporter) Table([]string, [][]string) {}

// Spin returns a no-op spinner. Done/Fail are silent; only Fail forwards to
// Error so that fatal failures still surface in scripts.
func (r *SilentReporter) Spin(string) reporter.Spinner {
	return &silentSpinner{r: r}
}

// Targets returns a no-op Targets handle. Done/Skip/SetPhase/SetProgress are
// silent; only Fail forwards to Error so that fatal failures still surface
// in scripts.
func (r *SilentReporter) Targets([]string) reporter.Targets {
	return &silentTargets{r: r}
}

// Data writes a machine-readable payload to stdout. Always emitted.
func (r *SilentReporter) Data(p []byte) {
	_, _ = r.outW.Write(p)
}

// Diff writes a unified diff payload to stdout. Always emitted as raw bytes
// (no color) so piped consumers receive clean text.
func (r *SilentReporter) Diff(p []byte) {
	_, _ = r.outW.Write(p)
}

type silentSpinner struct {
	r        *SilentReporter
	finished bool
}

func (s *silentSpinner) Update(string) {}

func (s *silentSpinner) Done(string) {
	if s.finished {
		return
	}
	s.finished = true
}

func (s *silentSpinner) Fail(msg string) {
	if s.finished {
		return
	}
	s.finished = true
	s.r.Error(msg)
}

func (s *silentSpinner) Stop() {
	if s.finished {
		return
	}
	s.finished = true
}

var _ reporter.Targets = (*silentTargets)(nil)

type silentTargets struct {
	r      *SilentReporter
	closed bool
}

func (t *silentTargets) SetPhase(string, string, string)            {}
func (t *silentTargets) SetProgress(string, float64, time.Duration) {}
func (t *silentTargets) Done(string, string)                        {}
func (t *silentTargets) Skip(string, string)                        {}

func (t *silentTargets) Fail(_ string, err error) {
	if t.closed || err == nil {
		return
	}
	t.r.Error(err.Error())
}

func (t *silentTargets) Close() {
	if t.closed {
		return
	}
	t.closed = true
}
