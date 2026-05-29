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
		// Full-width CJK characters each occupy 2 terminal cells.
		// "東京" is 2 runes but 4 display cells → column width = 4+3 = 7.
		{name: "CJK double-width", ids: []string{"東京"}, want: 7},
		// Mix of ASCII and CJK: "ab東京" = 2 + 4 = 6 cells.
		{name: "mixed ascii+CJK", ids: []string{"ab東京", "abcde"}, want: 9},
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

func TestVisibleWidthWideChars(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		s    string
		want int
	}{
		{name: "empty", s: "", want: 0},
		{name: "ascii", s: "hello", want: 5},
		// Full-width CJK: each rune is 2 display cells.
		{name: "CJK two runes", s: "東京", want: 4},
		{name: "mixed ascii+CJK", s: "ab東京", want: 6},
		// ANSI escape sequences must count as 0 cells.
		{name: "ANSI sequences stripped", s: "\x1b[31mred\x1b[0m", want: 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := visibleWidth(tt.s); got != tt.want {
				t.Errorf("visibleWidth(%q) = %d, want %d", tt.s, got, tt.want)
			}
		})
	}
}

func TestSanitizeIdentifier(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want string
	}{
		{"plain ascii unchanged", "us-east-1/my-app/my-profile/dev", "us-east-1/my-app/my-profile/dev"},
		{"empty unchanged", "", ""},
		{"CSI clear screen stripped", "us-east-1/\x1b[2Jevil/p/e", "us-east-1/evil/p/e"},
		{"CSI cursor move stripped", "a\x1b[1Ab", "ab"},
		{"OSC sequence stripped", "a\x1b]0;title\x07b", "ab"},
		{"OSC with ST stripped", "a\x1b]0;title\x1b\\b", "ab"},
		{"DCS stripped", "a\x1bP1;2|payload\x1b\\b", "ab"},
		{"SGR color stripped", "a\x1b[31;1mb\x1b[0m", "ab"},
		{"bare ESC pair stripped", "a\x1bMb", "ab"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := sanitizeIdentifier(tt.in); got != tt.want {
				t.Errorf("sanitizeIdentifier(%q) = %q, want %q", tt.in, got, tt.want)
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
		// "東京" is 2 runes but 4 display cells; padding to width=7 needs 3 spaces.
		{"東京", 7, "東京   "},
		// "東京" display width 4, target width 4 → no padding.
		{"東京", 4, "東京"},
	}
	for _, tt := range tests {
		if got := padID(tt.id, tt.width); got != tt.want {
			t.Errorf("padID(%q, %d) = %q, want %q", tt.id, tt.width, got, tt.want)
		}
	}
}

func TestTruncateCells(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		in    string
		limit int
		want  string
	}{
		{"empty input", "", 10, ""},
		{"limit zero returns empty", "abc", 0, ""},
		{"limit negative returns empty", "abc", -1, ""},
		{"shorter than limit passes through", "abc", 10, "abc"},
		{"equal to limit passes through", "abc", 3, "abc"},
		{"limit one returns ellipsis only", "abc", 1, "…"},
		{"truncates to limit with ellipsis", "abcdef", 4, "abc…"},
		{"long ASCII", strings.Repeat("a", 100), 5, "aaaa…"},
		// Full-width CJK: each rune = 2 cells.
		// limit=4 cells: fits "あ" (2) + "…" (1) = 3 cells (not 4), or "あい" = 4 cells exactly.
		// "あいうえお" with limit=4: "あい" = 4 cells, fits exactly → no truncation needed? No —
		// len("あいうえお") = 5 runes = 10 cells > 4, so truncate: budget=4, "あ"=2 cells fits,
		// remaining 2 cells: can we fit "い" (2 cells)? that would be 4 total but leaves 0 for "…".
		// We need limit-1=3 cells for content: "あ"=2 fits, "い" doesn't (2 > 1). So result = "あ…".
		{"CJK truncates at cell boundary", "あいうえお", 4, "あ…"},
		// limit=5 cells: content budget=4 cells. "あ"=2, "い"=2 → 4 cells. Result: "あい…".
		{"CJK limit 5", "あいうえお", 5, "あい…"},
		// "東京" = 4 cells, limit=4 → fits exactly, no truncation.
		{"CJK fits exactly", "東京", 4, "東京"},
		// Mixed: "ab東" = 4 cells, limit=4 → fits.
		{"mixed fits exactly", "ab東", 4, "ab東"},
		// Mixed: "ab東京" = 6 cells, limit=5 → content budget=4: "ab"=2, "東"=2 → 4 cells → "ab東…".
		{"mixed truncates", "ab東京", 5, "ab東…"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := truncateCells(tt.in, tt.limit); got != tt.want {
				t.Errorf("truncateCells(%q, %d) = %q, want %q", tt.in, tt.limit, got, tt.want)
			}
		})
	}
}
