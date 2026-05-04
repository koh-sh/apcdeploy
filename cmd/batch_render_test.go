package cmd

import (
	"bytes"
	"errors"
	"io"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/koh-sh/apcdeploy/internal/batch"
	reportertest "github.com/koh-sh/apcdeploy/internal/reporter/testing"
)

// captureStderr swaps os.Stderr for a pipe, runs fn, and returns the
// captured bytes. Used by the batch render tests because the helpers
// write directly to os.Stderr (no Reporter primitive carries plain
// summary lines — see batch_render.go for the rationale).
//
// Tests run sequentially around the swap because os.Stderr is process
// global; callers must not use t.Parallel().
var captureMu sync.Mutex

func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	captureMu.Lock()
	defer captureMu.Unlock()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	orig := os.Stderr
	os.Stderr = w

	done := make(chan []byte, 1)
	go func() {
		buf, _ := io.ReadAll(r)
		done <- buf
	}()

	fn()
	_ = w.Close()
	os.Stderr = orig

	return string(<-done)
}

func TestRenderBatchSummary_HiddenForSingleTarget(t *testing.T) {
	silent = false
	t.Cleanup(func() { silent = false })

	out := captureStderr(t, func() {
		renderBatchSummary(&reportertest.MockReporter{}, batch.Summary{OK: 1}, 1, summaryConfig{noopVerb: "no-op"})
	})
	if out != "" {
		t.Errorf("N=1 should not emit a summary line; got %q", out)
	}
}

func TestRenderBatchSummary_RendersForMultipleTargets(t *testing.T) {
	silent = false
	t.Cleanup(func() { silent = false })

	out := captureStderr(t, func() {
		renderBatchSummary(&reportertest.MockReporter{},
			batch.Summary{OK: 2, NoOp: 1, Failed: 0, Elapsed: 12 * time.Second},
			3,
			summaryConfig{noopVerb: "no-op", withElapsed: true},
		)
	})
	if !strings.Contains(out, "2 ok, 1 no-op, 0 failed (12s)") {
		t.Errorf("summary missing expected counts/elapsed; got %q", out)
	}
}

func TestRenderBatchSummary_SkippedFoldedIntoNoOp(t *testing.T) {
	silent = false
	t.Cleanup(func() { silent = false })

	// Fail-fast skip is "no-op" from the user's perspective: the target
	// didn't change anything because the batch short-circuited it.
	// Counter 2 (NoOp) collapses NoOp + Skipped.
	out := captureStderr(t, func() {
		renderBatchSummary(&reportertest.MockReporter{},
			batch.Summary{OK: 1, NoOp: 1, Skipped: 1, Failed: 1},
			4,
			summaryConfig{noopVerb: "no-op"},
		)
	})
	if !strings.Contains(out, "1 ok, 2 no-op, 1 failed") {
		t.Errorf("Skipped should fold into no-op count; got %q", out)
	}
}

func TestRenderBatchSummary_SilentSuppresses(t *testing.T) {
	silent = true
	t.Cleanup(func() { silent = false })

	out := captureStderr(t, func() {
		renderBatchSummary(&reportertest.MockReporter{},
			batch.Summary{OK: 2, Failed: 1, Errors: []batch.TargetError{
				{Identifier: "us-east-1/a/b/c", Err: errors.New("boom")},
			}},
			3,
			summaryConfig{noopVerb: "no-op"},
		)
	})
	if out != "" {
		t.Errorf("--silent must suppress everything; got %q", out)
	}
}

func TestRenderBatchSummary_ErrorsSection(t *testing.T) {
	silent = false
	t.Cleanup(func() { silent = false })

	out := captureStderr(t, func() {
		renderBatchSummary(&reportertest.MockReporter{},
			batch.Summary{OK: 1, Failed: 1, Errors: []batch.TargetError{
				{Identifier: "us-east-1/a/b/c", Err: errors.New("boom")},
			}},
			2,
			summaryConfig{noopVerb: "no-op"},
		)
	})
	for _, want := range []string{"Errors:", "us-east-1/a/b/c", "boom"} {
		if !strings.Contains(out, want) {
			t.Errorf("Errors: section missing %q; got %q", want, out)
		}
	}
}

func TestRenderErrorsSection_NoOpWhenEmpty(t *testing.T) {
	out := captureStderr(t, func() {
		renderErrorsSection(batch.Summary{OK: 3})
	})
	if out != "" {
		t.Errorf("no failures should yield no Errors: section; got %q", out)
	}
}

// Sanity: the captureStderr helper itself works (smoke).
func TestCaptureStderr_Smoke(t *testing.T) {
	out := captureStderr(t, func() {
		_, _ = os.Stderr.Write([]byte("hello"))
	})
	if !bytes.Equal([]byte(out), []byte("hello")) {
		t.Errorf("captureStderr round-trip = %q, want %q", out, "hello")
	}
}

func TestRequireSingleConfig(t *testing.T) {
	tests := []struct {
		name   string
		input  []string
		wantOK bool
		want   string
		errSub string
	}{
		{"empty defaults to apcdeploy.yml", nil, true, defaultConfigFile, ""},
		{"single value passes through", []string{"my.yml"}, true, "my.yml", ""},
		{"two values rejected", []string{"a.yml", "b.yml"}, false, "", "does not support multiple"},
		{"three values rejected", []string{"a.yml", "b.yml", "c.yml"}, false, "", "got 3"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configFiles = tt.input
			t.Cleanup(func() { configFiles = []string{defaultConfigFile} })
			got, err := requireSingleConfig("test")
			if tt.wantOK {
				if err != nil {
					t.Fatalf("err = %v, want nil", err)
				}
				if got != tt.want {
					t.Errorf("path = %q, want %q", got, tt.want)
				}
			} else {
				if err == nil {
					t.Fatal("err = nil, want error")
				}
				if !strings.Contains(err.Error(), tt.errSub) {
					t.Errorf("err = %q, want substring %q", err.Error(), tt.errSub)
				}
			}
		})
	}
}

func TestFlushDiffPayloads(t *testing.T) {
	tests := []struct {
		name     string
		targets  []*batch.Target
		payloads [][]byte
		want     string
	}{
		{
			name: "skips empty payloads",
			targets: []*batch.Target{
				{Identifier: "us-east-1/a/b/dev"},
				{Identifier: "us-east-1/a/b/stg"},
				{Identifier: "us-east-1/a/b/prod"},
			},
			payloads: [][]byte{
				nil,
				[]byte("--- a\n+++ b\n@@ -1 +1 @@\n-x\n+y\n"),
				nil,
			},
			want: "=== us-east-1/a/b/stg ===\n--- a\n+++ b\n@@ -1 +1 @@\n-x\n+y\n",
		},
		{
			name: "headers each non-empty payload in argument order with blank-line separator",
			targets: []*batch.Target{
				{Identifier: "id-A"},
				{Identifier: "id-B"},
			},
			payloads: [][]byte{
				[]byte("diff-A\n"),
				[]byte("diff-B\n"),
			},
			want: "=== id-A ===\ndiff-A\n\n=== id-B ===\ndiff-B\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rep := &reportertest.MockReporter{}
			flushDiffPayloads(rep, tt.targets, tt.payloads)
			if got := string(rep.Stdout); got != tt.want {
				t.Errorf("stdout = %q\nwant   %q", got, tt.want)
			}
		})
	}
}
