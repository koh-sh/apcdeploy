package diff

import (
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
