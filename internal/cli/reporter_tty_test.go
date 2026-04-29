package cli

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

// newTTYReporter mirrors newTestReporter but with both streams flagged as
// TTY, so the lipgloss / spinner / diff-color branches are exercised.
func newTTYReporter() (*Reporter, *bytes.Buffer, *bytes.Buffer) {
	var out, errBuf bytes.Buffer
	r := &Reporter{
		outW:   &out,
		errW:   &errBuf,
		outTTY: true,
		errTTY: true,
	}
	return r, &out, &errBuf
}

func TestReporter_TTYStderrKinds(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		invoke func(r *Reporter)
		want   string
	}{
		{"header tty", func(r *Reporter) { r.Header("Section") }, "Section"},
		{"box tty with title", func(r *Reporter) { r.Box("Title", []string{"a", "b"}) }, "Title"},
		{"box tty no title", func(r *Reporter) { r.Box("", []string{"only line"}) }, "only line"},
		{"table tty", func(r *Reporter) { r.Table([]string{"H1"}, [][]string{{"v1"}}) }, "v1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r, out, errBuf := newTTYReporter()
			tt.invoke(r)
			if out.Len() != 0 {
				t.Errorf("stderr-only kind unexpectedly wrote to stdout: %q", out.String())
			}
			if !strings.Contains(errBuf.String(), tt.want) {
				t.Errorf("stderr = %q, want to contain %q", errBuf.String(), tt.want)
			}
		})
	}
}

func TestReporter_TTYDiffColorizes(t *testing.T) {
	t.Parallel()

	r, out, _ := newTTYReporter()
	r.Diff([]byte("--- a\n+++ b\n@@ hunk @@\n+added\n-removed\n context\n"))

	got := out.String()
	for _, want := range []string{"--- a", "+++ b", "@@ hunk @@", "+added", "-removed", "context"} {
		if !strings.Contains(got, want) {
			t.Errorf("colorized diff missing %q; got %q", want, got)
		}
	}
}

func TestReporter_NonTTYDiffIsRaw(t *testing.T) {
	t.Parallel()

	r, out, _ := newTestReporter()
	payload := []byte("+x\n-y\n")
	r.Diff(payload)
	if out.String() != string(payload) {
		t.Errorf("non-TTY diff should be raw bytes; got %q want %q", out.String(), string(payload))
	}
}

func TestReporter_SpinTTYAnimatesAndCleansUp(t *testing.T) {
	t.Parallel()

	r, _, errBuf := newTTYReporter()
	sp := r.Spin("loading")
	// Allow the animation goroutine to render at least one frame.
	time.Sleep(150 * time.Millisecond)
	sp.Done("loaded")

	got := errBuf.String()
	if !strings.Contains(got, "loading") {
		t.Errorf("expected spinner to render the message at least once; got %q", got)
	}
	if !strings.Contains(got, "loaded") {
		t.Errorf("expected Done() to print success line; got %q", got)
	}
}

func TestReporter_SpinTTYFail(t *testing.T) {
	t.Parallel()

	r, _, errBuf := newTTYReporter()
	sp := r.Spin("loading")
	time.Sleep(50 * time.Millisecond)
	sp.Fail("failed")

	if !strings.Contains(errBuf.String(), "failed") {
		t.Errorf("expected Fail() to print error line; got %q", errBuf.String())
	}
}

func TestReporter_SpinTTYStopIsSilentAndIdempotent(t *testing.T) {
	t.Parallel()

	r, _, errBuf := newTTYReporter()
	sp := r.Spin("loading")
	sp.Stop()

	got := errBuf.String()
	// Stop should clear the spinner line and emit no completion line. Any
	// frames already drawn are erased by the terminate path's "\r\033[K".
	if strings.Contains(got, "loaded") || strings.Contains(got, "failed") {
		t.Errorf("Stop must not emit a completion line; got %q", got)
	}

	// Subsequent terminations must be no-ops.
	sp.Done("late")
	sp.Fail("late")
	if strings.Contains(errBuf.String(), "late") {
		t.Errorf("Done/Fail after Stop must be no-ops; got %q", errBuf.String())
	}
}

func TestVisibleWidth(t *testing.T) {
	t.Parallel()

	tests := []struct {
		in   string
		want int
	}{
		{"", 0},
		{"abc", 3},
		{"日本語", 3},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			t.Parallel()
			if got := visibleWidth(tt.in); got != tt.want {
				t.Errorf("visibleWidth(%q) = %d, want %d", tt.in, got, tt.want)
			}
		})
	}
}

func TestColorizeDiffLine(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want string // substring expected in output
	}{
		{"add", "+added\n", "+added"},
		{"del", "-removed\n", "-removed"},
		{"hunk", "@@ -1 +1 @@\n", "@@"},
		{"meta plus", "+++ b/file\n", "+++ b/file"},
		{"meta minus", "--- a/file\n", "--- a/file"},
		{"plain context", " context\n", "context"},
		{"no trailing newline", "+abc", "+abc"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := colorizeDiffLine(tt.in)
			if !strings.Contains(got, tt.want) {
				t.Errorf("colorizeDiffLine(%q) = %q, want to contain %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestIsTerminal_NilFile(t *testing.T) {
	t.Parallel()

	if IsTerminal(nil) {
		t.Error("IsTerminal(nil) should return false")
	}
}
