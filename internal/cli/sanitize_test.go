package cli

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/koh-sh/apcdeploy/internal/reporter"
)

// injected is a terminal control sequence (clear screen) that remote AppConfig
// strings could carry. None of the human-facing kinds may write it raw; each
// must render it as the visible escape `\x1b[2J` instead.
const (
	injected        = "evil\x1b[2J"
	injectedEscaped = `evil\x1b[2J`
)

func assertNeutralised(t *testing.T, got string) {
	t.Helper()
	if strings.Contains(got, "\x1b[2J") {
		t.Errorf("output contains the raw control sequence: %q", got)
	}
	if !strings.Contains(got, injectedEscaped) {
		t.Errorf("output does not contain the visible escape %q: %q", injectedEscaped, got)
	}
}

func TestReporter_NeutralisesControlSequences(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		invoke func(r *Reporter)
	}{
		{"step", func(r *Reporter) { r.Step(injected) }},
		{"success", func(r *Reporter) { r.Success(injected) }},
		{"info", func(r *Reporter) { r.Info(injected) }},
		{"warn", func(r *Reporter) { r.Warn(injected) }},
		{"error", func(r *Reporter) { r.Error(injected) }},
		{"header", func(r *Reporter) { r.Header(injected) }},
		{"box title", func(r *Reporter) { r.Box(injected, []string{"line"}) }},
		{"box line", func(r *Reporter) { r.Box("T", []string{injected}) }},
		{"table header", func(r *Reporter) { r.Table([]string{injected}, [][]string{{"v"}}) }},
		{"table cell", func(r *Reporter) { r.Table([]string{"Field", "Value"}, [][]string{{"Description", injected}}) }},
		{"spinner done", func(r *Reporter) { r.Spin("working").Done(injected) }},
		{"spinner fail", func(r *Reporter) { r.Spin("working").Fail(injected) }},
	}

	for _, tty := range []bool{false, true} {
		for _, tt := range tests {
			name := tt.name
			if tty {
				name += " tty"
			}
			t.Run(name, func(t *testing.T) {
				t.Parallel()
				var out, errBuf bytes.Buffer
				r := &Reporter{outW: &out, errW: &errBuf, outTTY: tty, errTTY: tty}
				tt.invoke(r)
				assertNeutralised(t, errBuf.String())
			})
		}
	}
}

func TestReporter_SpinTTYNeutralisesLabel(t *testing.T) {
	t.Parallel()

	var out, errBuf bytes.Buffer
	r := &Reporter{outW: &out, errW: &errBuf, outTTY: true, errTTY: true}
	s := r.Spin(injected)
	s.Update(injected)
	s.Stop()
	assertNeutralised(t, errBuf.String())
}

func TestReporter_ErrorKeepsLineBreaks(t *testing.T) {
	t.Parallel()

	r, _, errBuf := newTestReporter()
	r.Error("schema validation failed:\n  - /a: bad")
	if !strings.Contains(errBuf.String(), "failed:\n  - /a: bad") {
		t.Errorf("multi-line error lost its layout: %q", errBuf.String())
	}
}

func TestReporter_DiffTTYNeutralisesButPipedIsRaw(t *testing.T) {
	t.Parallel()

	payload := []byte("-" + injected + "\n+\tok\n")

	tty, ttyOut, _ := newTTYReporter()
	tty.Diff(payload)
	assertNeutralised(t, ttyOut.String())
	if strings.Contains(ttyOut.String(), `\tok`) {
		t.Errorf("TTY diff must not escape tabs in content: %q", ttyOut.String())
	}

	piped, pipedOut, _ := newTestReporter()
	piped.Diff(payload)
	if pipedOut.String() != string(payload) {
		t.Errorf("piped diff must be byte-for-byte: got %q, want %q", pipedOut.String(), payload)
	}
}

func TestReporter_DataIsRaw(t *testing.T) {
	t.Parallel()

	for _, tty := range []bool{false, true} {
		var out, errBuf bytes.Buffer
		r := &Reporter{outW: &out, errW: &errBuf, outTTY: tty, errTTY: tty}
		r.Data([]byte(injected))
		if out.String() != injected {
			t.Errorf("Data (tty=%v) must be byte-for-byte: got %q", tty, out.String())
		}
	}
}

func TestSilentReporter_NeutralisesErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		invoke func(r *SilentReporter)
	}{
		{"error", func(r *SilentReporter) { r.Error(injected) }},
		{"spinner fail", func(r *SilentReporter) { r.Spin("x").Fail(injected) }},
		{"targets fail", func(r *SilentReporter) { r.Targets([]string{"id"}).Fail("id", errors.New(injected)) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var out, errBuf bytes.Buffer
			r := &SilentReporter{outW: &out, errW: &errBuf}
			tt.invoke(r)
			assertNeutralised(t, errBuf.String())
		})
	}
}

func TestTargets_NeutraliseControlSequences(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		ids    []string
		invoke func(tg reporter.Targets, id string)
	}{
		{"identifier", []string{injected}, func(tg reporter.Targets, id string) {
			tg.Done(id, "deployed")
		}},
		{"phase detail", []string{"id"}, func(tg reporter.Targets, id string) {
			tg.SetPhase(id, "deploying", injected)
		}},
		{"done summary", []string{"id"}, func(tg reporter.Targets, id string) {
			tg.Done(id, injected)
		}},
		{"fail error", []string{"id"}, func(tg reporter.Targets, id string) {
			tg.Fail(id, errors.New(injected))
		}},
		{"skip reason", []string{"id"}, func(tg reporter.Targets, id string) {
			tg.Skip(id, injected)
		}},
	}

	for _, tt := range tests {
		t.Run("plain "+tt.name, func(t *testing.T) {
			t.Parallel()
			tg, buf := newTestPlainTargets(t, tt.ids)
			tt.invoke(tg, tt.ids[0])
			tg.Close()
			assertNeutralised(t, buf.String())
		})
		t.Run("tty "+tt.name, func(t *testing.T) {
			t.Parallel()
			tg, buf := newTestTTYTargets(t, tt.ids)
			tt.invoke(tg, tt.ids[0])
			tg.Close()
			assertNeutralised(t, buf.String())
		})
	}
}

func TestTargets_FailErrorLayout(t *testing.T) {
	t.Parallel()

	err := errors.New("schema validation failed:\n  - /a: bad")

	// Non-TTY lines are appended to a log, so a multi-line error keeps its
	// layout there.
	plain, plainBuf := newTestPlainTargets(t, []string{"id"})
	plain.Fail("id", err)
	plain.Close()
	if !strings.Contains(plainBuf.String(), "failed:\n  - /a: bad") {
		t.Errorf("plain Fail lost the error layout: %q", plainBuf.String())
	}

	// A TTY row is redrawn in place and must stay on one line.
	tty, ttyBuf := newTestTTYTargets(t, []string{"id"})
	tty.Fail("id", err)
	tty.Close()
	if !strings.Contains(stripANSI(ttyBuf.String()), `failed:\n`) {
		t.Errorf("TTY Fail row must render the error on one line: %q", ttyBuf.String())
	}
}
