package cli

import (
	"strings"
	"testing"
	"time"
)

func TestRenderBar(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		percent   float64
		fullCount int
	}{
		{name: "0%", percent: 0, fullCount: 0},
		{name: "25%", percent: 0.25, fullCount: 5},
		{name: "50%", percent: 0.5, fullCount: 10},
		{name: "100%", percent: 1, fullCount: 20},
		{name: "clamped above 100%", percent: 1.5, fullCount: 20},
		{name: "clamped below 0%", percent: -0.5, fullCount: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := renderBar(tt.percent)
			if gotFull := strings.Count(got, "█"); gotFull != tt.fullCount {
				t.Errorf("renderBar(%v): █ count = %d, want %d", tt.percent, gotFull, tt.fullCount)
			}
			gotEmpty := strings.Count(got, "░")
			wantEmpty := targetsBarWidth - tt.fullCount
			if gotEmpty != wantEmpty {
				t.Errorf("renderBar(%v): ░ count = %d, want %d", tt.percent, gotEmpty, wantEmpty)
			}
		})
	}
}

func TestClampPercent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		percent float64
		want    int
	}{
		{0, 0},
		{0.5, 50},
		{1, 100},
		{-0.1, 0},
		{1.5, 100},
		{0.333, 33},
	}
	for _, tt := range tests {
		if got := clampPercent(tt.percent); got != tt.want {
			t.Errorf("clampPercent(%v) = %d, want %d", tt.percent, got, tt.want)
		}
	}
}

func TestFormatETA(t *testing.T) {
	t.Parallel()

	tests := []struct {
		d    time.Duration
		want string
	}{
		{0, ""},
		{-time.Second, ""},
		{30 * time.Second, "(~30 sec left)"},
		{59 * time.Second, "(~59 sec left)"},
		{90 * time.Second, "(~2 min left)"},
		{5 * time.Minute, "(~5 min left)"},
	}
	for _, tt := range tests {
		if got := formatETA(tt.d); got != tt.want {
			t.Errorf("formatETA(%v) = %q, want %q", tt.d, got, tt.want)
		}
	}
}

func TestPercentThreshold(t *testing.T) {
	t.Parallel()

	tests := []struct {
		percent float64
		want    int
	}{
		{0, 0},
		{0.1, 0},
		{0.25, 25},
		{0.49, 25},
		{0.5, 50},
		{0.74, 50},
		{0.75, 75},
		{0.99, 75},
		{1, 100},
	}
	for _, tt := range tests {
		if got := percentThreshold(tt.percent); got != tt.want {
			t.Errorf("percentThreshold(%v) = %d, want %d", tt.percent, got, tt.want)
		}
	}
}

func TestIDColumnWidth(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		ids  []string
		want int
	}{
		{name: "empty", ids: nil, want: 3},
		{name: "single", ids: []string{"a"}, want: 4},
		{name: "longest wins", ids: []string{"abc", "ab", "abcdef"}, want: 9},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := idColumnWidth(tt.ids); got != tt.want {
				t.Errorf("idColumnWidth(%v) = %d, want %d", tt.ids, got, tt.want)
			}
		})
	}
}

func TestPadID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		id    string
		width int
		want  string
	}{
		{"abc", 6, "abc   "},
		{"abc", 3, "abc"},
		{"abc", 2, "abc"},
		{"", 4, "    "},
	}
	for _, tt := range tests {
		if got := padID(tt.id, tt.width); got != tt.want {
			t.Errorf("padID(%q, %d) = %q, want %q", tt.id, tt.width, got, tt.want)
		}
	}
}
