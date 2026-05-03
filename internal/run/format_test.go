package run

import (
	"testing"
	"time"
)

func TestFormatElapsed(t *testing.T) {
	t.Parallel()

	tests := []struct {
		d    time.Duration
		want string
	}{
		{0, "0s"},
		{1500 * time.Millisecond, "2s"},
		{45 * time.Second, "45s"},
		{59 * time.Second, "59s"},
		{time.Minute, "1m"},
		{61 * time.Second, "1m 1s"},
		{8*time.Minute + 15*time.Second, "8m 15s"},
		{60 * time.Minute, "60m"},
	}
	for _, tt := range tests {
		if got := formatElapsed(tt.d); got != tt.want {
			t.Errorf("formatElapsed(%v) = %q, want %q", tt.d, got, tt.want)
		}
	}
}

func TestFormatRunSummary(t *testing.T) {
	t.Parallel()

	// "started" never includes elapsed (no wait flag → no deploy duration to
	// quote). Other verbs include elapsed when start is non-zero.
	fixed := time.Now().Add(-12 * time.Second)

	tests := []struct {
		name     string
		verb     string
		start    time.Time
		version  int32
		strategy string
		addendum string
		want     string
	}{
		{
			name: "started omits elapsed and includes deployment addendum",
			verb: "started", version: 42, strategy: "AppConfig.AllAtOnce", addendum: "deployment #5",
			want: "started — v42, AppConfig.AllAtOnce, deployment #5",
		},
		{
			name: "deployed includes elapsed and addendum",
			verb: "deployed", start: fixed, version: 42, strategy: "Linear50PercentEvery30Seconds", addendum: "baking started",
			want: "deployed (12s) — v42, Linear50PercentEvery30Seconds, baking started",
		},
		{
			name: "complete with no addendum",
			verb: "complete", start: fixed, version: 7, strategy: "Canary10Percent20Minutes",
			want: "complete (12s) — v7, Canary10Percent20Minutes",
		},
		{
			name: "no version inserts strategy after em-dash",
			verb: "deployed", start: fixed, strategy: "AllAtOnce",
			want: "deployed (12s) — AllAtOnce",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := formatRunSummary(tt.verb, tt.start, tt.version, tt.strategy, tt.addendum); got != tt.want {
				t.Errorf("formatRunSummary() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRemainingSeconds(t *testing.T) {
	t.Parallel()

	now := time.Now()
	tests := []struct {
		name     string
		deadline time.Time
		wantMin  int
		wantMax  int
	}{
		{"future deadline", now.Add(30 * time.Second), 25, 30},
		{"past deadline clamps to 1", now.Add(-1 * time.Second), 1, 1},
		{"current time clamps to 1", now, 1, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := remainingSeconds(tt.deadline)
			if got < tt.wantMin || got > tt.wantMax {
				t.Errorf("remainingSeconds(%v) = %d, want in [%d, %d]", tt.deadline, got, tt.wantMin, tt.wantMax)
			}
		})
	}
}
