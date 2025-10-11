package diff

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func Test_displayColorizedDiff(t *testing.T) {
	tests := []struct {
		name string
		diff string
	}{
		{
			name: "empty diff",
			diff: "",
		},
		{
			name: "additions only",
			diff: "+added line 1\n+added line 2",
		},
		{
			name: "deletions only",
			diff: "-removed line 1\n-removed line 2",
		},
		{
			name: "diff headers",
			diff: "@@ -1,3 +1,3 @@\ncontext line",
		},
		{
			name: "mixed changes",
			diff: "@@ -1,3 +1,3 @@\n context line\n-removed line\n+added line",
		},
		{
			name: "empty lines in diff",
			diff: "+added\n\n-removed",
		},
		{
			name: "context lines",
			diff: " context line 1\n context line 2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Just verify it doesn't panic
			displayColorizedDiff(tt.diff)
		})
	}
}

func TestDisplaySilent(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		result     *Result
		wantOutput bool
		wantText   string
	}{
		{
			name: "no changes - should produce no output",
			result: &Result{
				HasChanges:  false,
				UnifiedDiff: "",
				FileName:    "data.json",
			},
			wantOutput: false,
		},
		{
			name: "has changes - should show diff only",
			result: &Result{
				HasChanges:  true,
				UnifiedDiff: "+added line\n-removed line",
				FileName:    "data.json",
			},
			wantOutput: true,
			wantText:   "added line",
		},
		{
			name: "has changes with multiple lines",
			result: &Result{
				HasChanges:  true,
				UnifiedDiff: "@@ -1,3 +1,3 @@\n context\n+new line\n-old line",
				FileName:    "config.yaml",
			},
			wantOutput: true,
			wantText:   "new line",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			output := captureOutput(func() {
				DisplaySilent(tt.result)
			})

			if tt.wantOutput {
				if output == "" {
					t.Error("DisplaySilent() expected output but got empty string")
				}
				if tt.wantText != "" && !strings.Contains(output, tt.wantText) {
					t.Errorf("DisplaySilent() output missing %q\nGot:\n%s", tt.wantText, output)
				}
			} else if output != "" {
				t.Errorf("DisplaySilent() expected no output but got:\n%s", output)
			}
		})
	}
}

func Test_countChanges(t *testing.T) {
	tests := []struct {
		name        string
		diff        string
		wantAdded   int
		wantRemoved int
	}{
		{
			name:        "empty diff",
			diff:        "",
			wantAdded:   0,
			wantRemoved: 0,
		},
		{
			name:        "additions only",
			diff:        "+added line 1\n+added line 2",
			wantAdded:   2,
			wantRemoved: 0,
		},
		{
			name:        "deletions only",
			diff:        "-removed line 1\n-removed line 2",
			wantAdded:   0,
			wantRemoved: 2,
		},
		{
			name:        "mixed changes",
			diff:        "+added\n-removed\n context",
			wantAdded:   1,
			wantRemoved: 1,
		},
		{
			name:        "ignore file headers",
			diff:        "--- a/file.json\n+++ b/file.json\n+added\n-removed",
			wantAdded:   1,
			wantRemoved: 1,
		},
		{
			name:        "multiple additions and deletions",
			diff:        "+line1\n+line2\n-line3\n-line4\n-line5",
			wantAdded:   2,
			wantRemoved: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

// Helper functions
func captureOutput(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	return buf.String()
}
