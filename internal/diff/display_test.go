package diff

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/koh-sh/apcdeploy/internal/aws"
	"github.com/koh-sh/apcdeploy/internal/config"
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
	tests := []struct {
		name         string
		result       *Result
		cfg          *config.Config
		resources    *aws.ResolvedResources
		deployment   *aws.DeploymentInfo
		wantStdout   string
		wantSuccess  bool // expect Success("No changes detected")
		wantWarn     bool
		wantWarnText string // substring expected in the in-progress notice on stderr
	}{
		{
			name: "no changes - emits success and no diff payload",
			result: &Result{
				HasChanges:  false,
				UnifiedDiff: "",
				FileName:    "data.json",
			},
			cfg:         &config.Config{Application: "app", Environment: "env"},
			resources:   &aws.ResolvedResources{Profile: &aws.ProfileInfo{Name: "prof"}},
			deployment:  &aws.DeploymentInfo{DeploymentNumber: 1, ConfigurationVersion: "1", State: "COMPLETE"},
			wantStdout:  "",
			wantSuccess: true,
		},
		{
			name: "has changes - emits diff payload to stdout and Info summary",
			result: &Result{
				HasChanges:  true,
				UnifiedDiff: "+added\n-removed\n",
				FileName:    "data.json",
			},
			cfg:        &config.Config{Application: "app", Environment: "env"},
			resources:  &aws.ResolvedResources{Profile: &aws.ProfileInfo{Name: "prof"}},
			deployment: &aws.DeploymentInfo{DeploymentNumber: 1, ConfigurationVersion: "1", State: "COMPLETE"},
			wantStdout: "+added\n-removed\n",
		},
		{
			name: "deployment in DEPLOYING surfaces an in-progress notice on stderr",
			result: &Result{
				HasChanges:  true,
				UnifiedDiff: "+new line\n-old line\n",
				FileName:    "data.json",
			},
			cfg:          &config.Config{Application: "app", Environment: "env"},
			resources:    &aws.ResolvedResources{Profile: &aws.ProfileInfo{Name: "prof"}},
			deployment:   &aws.DeploymentInfo{DeploymentNumber: 42, ConfigurationVersion: "1", State: "DEPLOYING"},
			wantStdout:   "+new line\n-old line\n",
			wantWarn:     true,
			wantWarnText: "Deployment #42 is currently DEPLOYING",
		},
		{
			name: "deployment in BAKING surfaces an in-progress notice on stderr",
			result: &Result{
				HasChanges:  true,
				UnifiedDiff: "+a\n-b\n",
				FileName:    "data.json",
			},
			cfg:          &config.Config{Application: "app", Environment: "env"},
			resources:    &aws.ResolvedResources{Profile: &aws.ProfileInfo{Name: "prof"}},
			deployment:   &aws.DeploymentInfo{DeploymentNumber: 7, ConfigurationVersion: "1", State: "BAKING"},
			wantStdout:   "+a\n-b\n",
			wantWarn:     true,
			wantWarnText: "Deployment #7 is currently BAKING",
		},
		{
			name: "no changes with DEPLOYING still warns",
			result: &Result{
				HasChanges:  false,
				UnifiedDiff: "",
				FileName:    "data.json",
			},
			cfg:          &config.Config{Application: "app", Environment: "env"},
			resources:    &aws.ResolvedResources{Profile: &aws.ProfileInfo{Name: "prof"}},
			deployment:   &aws.DeploymentInfo{DeploymentNumber: 99, ConfigurationVersion: "1", State: "DEPLOYING"},
			wantStdout:   "",
			wantSuccess:  true,
			wantWarn:     true,
			wantWarnText: "Deployment #99 is currently DEPLOYING",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Subtests intentionally do NOT call t.Parallel() — they swap the
			// package-level inProgressWarningSink and would race otherwise.
			var stderrBuf bytes.Buffer
			withWarningSink(t, &stderrBuf)

			r := &mockreporter.MockReporter{}
			display(r, tt.result, tt.cfg, tt.resources, tt.deployment)

			if got := string(r.Stdout); got != tt.wantStdout {
				t.Errorf("stdout payload = %q, want %q", got, tt.wantStdout)
			}

			if tt.wantSuccess && !r.HasMessage("success: No changes detected") {
				t.Errorf("expected Success(\"No changes detected\"); messages=%v", r.Messages)
			}

			// The in-progress warning bypasses Reporter.Warn (contract
			// exception) and goes directly to stderr — assert against the
			// stderr buffer rather than the Mock messages.
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

			// The notice must NEVER go through Reporter.Warn — even when
			// emitted, it stays out of the Mock's message log.
			for _, msg := range r.Messages {
				if strings.HasPrefix(msg, "warn:") && strings.Contains(msg, "is currently") {
					t.Errorf("in-progress notice leaked into Reporter.Warn: %q", msg)
				}
			}
		})
	}
}

func TestDisplay_NilDeployment(t *testing.T) {
	var stderrBuf bytes.Buffer
	withWarningSink(t, &stderrBuf)

	r := &mockreporter.MockReporter{}
	display(
		r,
		&Result{HasChanges: true, UnifiedDiff: "+a\n", FileName: "data.json"},
		&config.Config{Application: "app", Environment: "env"},
		&aws.ResolvedResources{Profile: &aws.ProfileInfo{Name: "prof"}},
		nil,
	)

	// "(none)" must appear as the Remote Version cell when no prior deployment.
	var found bool
	for _, table := range r.Tables {
		for _, row := range table.Rows {
			if len(row) >= 2 && row[0] == "Remote Version" && row[1] == "(none)" {
				found = true
			}
		}
	}
	if !found {
		t.Errorf("expected metadata table to contain Remote Version=(none); tables=%+v", r.Tables)
	}

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
