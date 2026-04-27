package cli

import (
	"strings"
	"testing"
)

func TestStateBadge(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		state string
	}{
		{"complete", "COMPLETE"},
		{"deploying", "DEPLOYING"},
		{"baking", "BAKING"},
		{"rolled back", "ROLLED_BACK"},
		{"rolling back", "ROLLING_BACK"},
		{"unknown stays plain", "VALIDATING"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := StateBadge(tt.state)
			if !strings.Contains(got, tt.state) {
				t.Errorf("StateBadge(%q) = %q, want to contain %q", tt.state, got, tt.state)
			}
		})
	}
}

func TestHeadingAndSubtleText(t *testing.T) {
	t.Parallel()

	if !strings.Contains(HeadingText("name"), "name") {
		t.Errorf("HeadingText must preserve raw text")
	}
	if !strings.Contains(SubtleText("id"), "id") {
		t.Errorf("SubtleText must preserve raw text")
	}
}
