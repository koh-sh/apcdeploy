package diff

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/koh-sh/apcdeploy/internal/aws"
	mockreporter "github.com/koh-sh/apcdeploy/internal/reporter/testing"
)

// withWarningSink temporarily redirects the package-level warning sink to the
// given writer for the duration of the test, restoring the original on
// cleanup. Tests using this helper MUST NOT run in parallel — the sink is
// package-scoped and concurrent overrides would race.
func withWarningSink(t *testing.T, w io.Writer) {
	t.Helper()
	orig := inProgressWarningSink
	inProgressWarningSink = w
	t.Cleanup(func() { inProgressWarningSink = orig })
}

func TestDisplay(t *testing.T) {
	const id = "us-east-1/app/prof/env"

	tests := []struct {
		name             string
		result           *Result
		deployment       *aws.DeploymentInfo
		wantStdout       string
		wantDoneSummary  string // substring expected in the done transition's Summary
		wantWarn         bool
		wantWarnText     string
	}{
		{
			name:             "no changes - finalises Targets with 'no changes'",
			result:           &Result{HasChanges: false, FileName: "data.json"},
			deployment:       &aws.DeploymentInfo{DeploymentNumber: 1, ConfigurationVersion: "1", State: "COMPLETE"},
			wantStdout:       "",
			wantDoneSummary:  "no changes",
		},
		{
			name:             "has changes - emits diff payload and 'diff (...)' summary",
			result:           &Result{HasChanges: true, UnifiedDiff: "+added\n-removed\n", FileName: "data.json"},
			deployment:       &aws.DeploymentInfo{DeploymentNumber: 1, ConfigurationVersion: "1", State: "COMPLETE"},
			wantStdout:       "+added\n-removed\n",
			wantDoneSummary:  "diff (",
		},
		{
			name:             "DEPLOYING surfaces an in-progress notice on stderr",
			result:           &Result{HasChanges: true, UnifiedDiff: "+a\n-b\n", FileName: "data.json"},
			deployment:       &aws.DeploymentInfo{DeploymentNumber: 42, ConfigurationVersion: "1", State: "DEPLOYING"},
			wantStdout:       "+a\n-b\n",
			wantDoneSummary:  "diff (",
			wantWarn:         true,
			wantWarnText:     "Deployment #42 is currently DEPLOYING",
		},
		{
			name:             "BAKING surfaces an in-progress notice on stderr",
			result:           &Result{HasChanges: true, UnifiedDiff: "+a\n-b\n", FileName: "data.json"},
			deployment:       &aws.DeploymentInfo{DeploymentNumber: 7, ConfigurationVersion: "1", State: "BAKING"},
			wantStdout:       "+a\n-b\n",
			wantDoneSummary:  "diff (",
			wantWarn:         true,
			wantWarnText:     "Deployment #7 is currently BAKING",
		},
		{
			name:             "no changes with DEPLOYING still warns",
			result:           &Result{HasChanges: false, FileName: "data.json"},
			deployment:       &aws.DeploymentInfo{DeploymentNumber: 99, ConfigurationVersion: "1", State: "DEPLOYING"},
			wantStdout:       "",
			wantDoneSummary:  "no changes",
			wantWarn:         true,
			wantWarnText:     "Deployment #99 is currently DEPLOYING",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Subtests intentionally do NOT call t.Parallel() — they swap the
			// package-level inProgressWarningSink and would race otherwise.
			var stderrBuf bytes.Buffer
			withWarningSink(t, &stderrBuf)

			r := &mockreporter.MockReporter{}
			tg := r.Targets([]string{id})
			display(r, tg, id, tt.result, tt.deployment)
			tg.Close()

			if got := string(r.Stdout); got != tt.wantStdout {
				t.Errorf("stdout payload = %q, want %q", got, tt.wantStdout)
			}

			foundDone := false
			for _, call := range r.TargetsCalls {
				for _, tr := range call.Transitions {
					if tr.Kind == "done" && strings.Contains(tr.Summary, tt.wantDoneSummary) {
						foundDone = true
					}
				}
			}
			if !foundDone {
				t.Errorf("expected Targets.Done summary containing %q; got %+v", tt.wantDoneSummary, r.TargetsCalls)
			}

			gotStderr := stderrBuf.String()
			if tt.wantWarn {
				if !strings.Contains(gotStderr, tt.wantWarnText) {
					t.Errorf("expected stderr to contain %q; got %q", tt.wantWarnText, gotStderr)
				}
				if !strings.Contains(gotStderr, "⚠") {
					t.Errorf("expected stderr to contain warning glyph; got %q", gotStderr)
				}
			} else if strings.Contains(gotStderr, "is currently") {
				t.Errorf("unexpected in-progress notice on stderr: %q", gotStderr)
			}

			// The notice must NEVER go through Reporter.Warn.
			for _, msg := range r.Messages {
				if strings.HasPrefix(msg, "warn:") && strings.Contains(msg, "is currently") {
					t.Errorf("in-progress notice leaked into Reporter.Warn: %q", msg)
				}
			}
		})
	}
}

func TestDisplay_NilDeployment(t *testing.T) {
	const id = "us-east-1/app/prof/env"

	var stderrBuf bytes.Buffer
	withWarningSink(t, &stderrBuf)

	r := &mockreporter.MockReporter{}
	tg := r.Targets([]string{id})
	display(
		r,
		tg,
		id,
		&Result{HasChanges: true, UnifiedDiff: "+a\n", FileName: "data.json"},
		nil,
	)
	tg.Close()

	// No in-progress notice when deployment is nil.
	if got := stderrBuf.String(); strings.Contains(got, "is currently") {
		t.Errorf("did not expect in-progress notice for nil deployment; got %q", got)
	}
}

func Test_countChanges(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		diff        string
		wantAdded   int
		wantRemoved int
	}{
		{"empty diff", "", 0, 0},
		{"additions only", "+added line 1\n+added line 2", 2, 0},
		{"deletions only", "-removed line 1\n-removed line 2", 0, 2},
		{"mixed changes", "+added\n-removed\n context", 1, 1},
		{"ignore file headers", "--- a/file.json\n+++ b/file.json\n+added\n-removed", 1, 1},
		{"multiple", "+line1\n+line2\n-line3\n-line4\n-line5", 2, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			added, removed := countChanges(tt.diff)
			if added != tt.wantAdded {
				t.Errorf("countChanges() added = %v, want %v", added, tt.wantAdded)
			}
			if removed != tt.wantRemoved {
				t.Errorf("countChanges() removed = %v, want %v", removed, tt.wantRemoved)
			}
		})
	}
}

func Test_ensureTrailingNewline(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want string
	}{
		{"empty", "", ""},
		{"already has newline", "abc\n", "abc\n"},
		{"missing newline", "abc", "abc\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := ensureTrailingNewline(tt.in); got != tt.want {
				t.Errorf("ensureTrailingNewline(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestFormatDiffSummary(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		added, removed int
		want          string
	}{
		{"single line", 1, 0, "diff (1 line changed: +1 -0)"},
		{"singular removed", 0, 1, "diff (1 line changed: +0 -1)"},
		{"plural", 2, 3, "diff (5 lines changed: +2 -3)"},
		{"zero", 0, 0, "diff (0 lines changed: +0 -0)"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := formatDiffSummary(tt.added, tt.removed); got != tt.want {
				t.Errorf("formatDiffSummary(%d,%d) = %q, want %q", tt.added, tt.removed, got, tt.want)
			}
		})
	}
}
