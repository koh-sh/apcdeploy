package diff

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/koh-sh/apcdeploy/internal/aws"
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

func TestDisplayDeploymentWarning(t *testing.T) {
	tests := []struct {
		name         string
		deployment   *aws.DeploymentInfo
		wantContains string // substring expected on stderr; "" means no output
	}{
		{
			name:         "nil deployment is silent",
			deployment:   nil,
			wantContains: "",
		},
		{
			name:         "COMPLETE state is silent",
			deployment:   &aws.DeploymentInfo{DeploymentNumber: 1, State: "COMPLETE"},
			wantContains: "",
		},
		{
			name:         "DEPLOYING surfaces a notice",
			deployment:   &aws.DeploymentInfo{DeploymentNumber: 42, State: "DEPLOYING"},
			wantContains: "Deployment #42 is currently DEPLOYING",
		},
		{
			name:         "BAKING surfaces a notice",
			deployment:   &aws.DeploymentInfo{DeploymentNumber: 7, State: "BAKING"},
			wantContains: "Deployment #7 is currently BAKING",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Subtests intentionally do NOT call t.Parallel() — they swap the
			// package-level inProgressWarningSink and would race otherwise.
			var sink bytes.Buffer
			withWarningSink(t, &sink)

			displayDeploymentWarning(tt.deployment)

			got := sink.String()
			if tt.wantContains == "" {
				if got != "" {
					t.Errorf("expected silent sink, got %q", got)
				}
				return
			}
			if !strings.Contains(got, tt.wantContains) {
				t.Errorf("expected sink to contain %q; got %q", tt.wantContains, got)
			}
			if !strings.Contains(got, "⚠") {
				t.Errorf("expected sink to contain warning glyph; got %q", got)
			}
		})
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
		name           string
		added, removed int
		want           string
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
