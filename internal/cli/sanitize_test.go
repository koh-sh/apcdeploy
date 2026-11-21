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
		// ttyOnly marks output that exists only on a TTY (the animated
		// spinner label).
		ttyOnly bool
	}{
		{"step", func(r *Reporter) { r.Step(injected) }, false},
		{"success", func(r *Reporter) { r.Success(injected) }, false},
		{"info", func(r *Reporter) { r.Info(injected) }, false},
		{"warn", func(r *Reporter) { r.Warn(injected) }, false},
		{"error", func(r *Reporter) { r.Error(injected) }, false},
		{"header", func(r *Reporter) { r.Header(injected) }, false},
		{"box title", func(r *Reporter) { r.Box(injected, []string{"line"}) }, false},
		{"box line", func(r *Reporter) { r.Box("T", []string{injected}) }, false},
		{"table header", func(r *Reporter) { r.Table([]string{injected}, [][]string{{"v"}}) }, false},
		{"table cell", func(r *Reporter) { r.Table([]string{"Field", "Value"}, [][]string{{"Description", injected}}) }, false},
		{"spinner done", func(r *Reporter) { r.Spin("working").Done(injected) }, false},
		{"spinner fail", func(r *Reporter) { r.Spin("working").Fail(injected) }, false},
		{"spinner label", func(r *Reporter) {
			s := r.Spin(injected)
			s.Update(injected)
			s.Stop()
		}, true},
	}

	for _, tty := range []bool{false, true} {
		for _, tt := range tests {
			if tt.ttyOnly && !tty {
				continue
			}
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

// TestReporter_Payloads covers stdout payloads: they stay byte-for-byte,
// except the TTY diff render, which is for a human and is neutralised.
func TestReporter_Payloads(t *testing.T) {
	t.Parallel()

	diff := []byte("-" + injected + "\n+\tok\n")

	tests := []struct {
		name   string
		tty    bool
		invoke func(r *Reporter)
		check  func(t *testing.T, out string)
	}{
		{
			name:   "diff tty is neutralised",
			tty:    true,
			invoke: func(r *Reporter) { r.Diff(diff) },
			check: func(t *testing.T, out string) {
				assertNeutralised(t, out)
				if strings.Contains(out, `\tok`) {
					t.Errorf("TTY diff must not escape tabs in content: %q", out)
				}
			},
		},
		{name: "diff piped is raw", invoke: func(r *Reporter) { r.Diff(diff) }, check: assertExact(string(diff))},
		{name: "data piped is raw", invoke: func(r *Reporter) { r.Data([]byte(injected)) }, check: assertExact(injected)},
		{name: "data tty is raw", tty: true, invoke: func(r *Reporter) { r.Data([]byte(injected)) }, check: assertExact(injected)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var out, errBuf bytes.Buffer
			r := &Reporter{outW: &out, errW: &errBuf, outTTY: tt.tty, errTTY: tt.tty}
			tt.invoke(r)
			tt.check(t, out.String())
		})
	}
}

func assertExact(want string) func(t *testing.T, out string) {
	return func(t *testing.T, out string) {
		t.Helper()
		if out != want {
			t.Errorf("payload must be byte-for-byte: got %q, want %q", out, want)
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

// TestMultiLineErrorLayout covers where a multi-line error keeps its line
// breaks: lines appended to a log do, a TTY Targets row (redrawn in place)
// renders it on one line.
func TestMultiLineErrorLayout(t *testing.T) {
	t.Parallel()

	const msg = "schema validation failed:\n  - /a: bad"

	tests := []struct {
		name   string
		render func(t *testing.T) string
		want   string
	}{
		{
			name: "reporter error keeps line breaks",
			render: func(t *testing.T) string {
				r, _, errBuf := newTestReporter()
				r.Error(msg)
				return errBuf.String()
			},
			want: "failed:\n  - /a: bad",
		},
		{
			name: "plain targets fail keeps line breaks",
			render: func(t *testing.T) string {
				tg, buf := newTestPlainTargets(t, []string{"id"})
				tg.Fail("id", errors.New(msg))
				tg.Close()
				return buf.String()
			},
			want: "failed:\n  - /a: bad",
		},
		{
			name: "tty targets fail stays on one line",
			render: func(t *testing.T) string {
				tg, buf := newTestTTYTargets(t, []string{"id"})
				tg.Fail("id", errors.New(msg))
				tg.Close()
				return stripANSI(buf.String())
			},
			want: `failed:\n`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.render(t); !strings.Contains(got, tt.want) {
				t.Errorf("output %q does not contain %q", got, tt.want)
			}
		})
	}
}
