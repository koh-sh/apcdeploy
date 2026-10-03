package diff

import (
	"bytes"
	"io"
	"testing"

	"github.com/koh-sh/apcdeploy/internal/aws"
	"github.com/koh-sh/apcdeploy/internal/batch"
)

// withWarningSink temporarily redirects the package-level warning sink to the
// given writer for the duration of the test, restoring the original on
// cleanup. Tests using this helper MUST NOT run in parallel — the sink is
// package-scoped and concurrent overrides would race.
func withWarningSink(t *testing.T, w io.Writer) {
	t.Helper()
	orig := inProgressWarningSink
	inProgressWarningSink = func() io.Writer { return w }
	t.Cleanup(func() { inProgressWarningSink = orig })
}

func TestDeploymentWarning(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		deployment *aws.DeploymentInfo
		want       string // "" means no warning
	}{
		{
			name:       "nil deployment has no warning",
			deployment: nil,
			want:       "",
		},
		{
			name:       "COMPLETE state has no warning",
			deployment: &aws.DeploymentInfo{DeploymentNumber: 1, State: "COMPLETE"},
			want:       "",
		},
		{
			name:       "DEPLOYING yields a warning",
			deployment: &aws.DeploymentInfo{DeploymentNumber: 42, State: "DEPLOYING"},
			want:       "Deployment #42 is currently DEPLOYING",
		},
		{
			name:       "BAKING yields a warning",
			deployment: &aws.DeploymentInfo{DeploymentNumber: 7, State: "BAKING"},
			want:       "Deployment #7 is currently BAKING",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := deploymentWarning(tt.deployment); got != tt.want {
				t.Errorf("deploymentWarning() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestWriteDeploymentWarnings(t *testing.T) {
	targets := []*batch.Target{
		{Identifier: "us-east-1/app/p/dev"},
		{Identifier: "us-east-1/app/p/stg"},
		{Identifier: "us-east-1/app/p/prod"},
	}

	tests := []struct {
		name     string
		warnings []string
		want     string
	}{
		{
			name:     "no warnings writes nothing",
			warnings: []string{"", "", ""},
			want:     "",
		},
		{
			name:     "single warning is prefixed with its identifier",
			warnings: []string{"", "Deployment #3 is currently BAKING", ""},
			want: "\n" +
				"⚠ us-east-1/app/p/stg: Deployment #3 is currently BAKING\n" +
				"The diff is calculated against the currently deploying version.\n",
		},
		{
			name: "multiple warnings are written as whole blocks in argument order",
			warnings: []string{
				"Deployment #1 is currently DEPLOYING",
				"",
				"Deployment #9 is currently BAKING",
			},
			want: "\n" +
				"⚠ us-east-1/app/p/dev: Deployment #1 is currently DEPLOYING\n" +
				"The diff is calculated against the currently deploying version.\n" +
				"\n" +
				"⚠ us-east-1/app/p/prod: Deployment #9 is currently BAKING\n" +
				"The diff is calculated against the currently deploying version.\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Subtests intentionally do NOT call t.Parallel() — they swap the
			// package-level inProgressWarningSink and would race otherwise.
			var sink bytes.Buffer
			withWarningSink(t, &sink)

			WriteDeploymentWarnings(targets, tt.warnings)

			if got := sink.String(); got != tt.want {
				t.Errorf("sink = %q, want %q", got, tt.want)
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
		// formatDiffs never emits unified-diff file headers, so every line
		// carries exactly one prefix. Content that itself starts with "++"
		// or "--" must still be counted.
		{"added line starting with ++", "+++x\n context", 1, 0},
		{"removed line starting with --", "---x\n context", 0, 1},
		{"mixed lines starting with ++ and --", "+++counter\n---counter\n+added\n-removed", 2, 2},
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
