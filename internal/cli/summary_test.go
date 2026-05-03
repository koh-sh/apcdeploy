package cli

import (
	"testing"
	"time"
)

func TestFormatElapsed(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		d    time.Duration
		want string
	}{
		{"zero", 0, "0s"},
		{"sub-second rounds up", 1500 * time.Millisecond, "2s"},
		{"45s", 45 * time.Second, "45s"},
		{"59s", 59 * time.Second, "59s"},
		{"exactly one minute", time.Minute, "1m"},
		{"1m 1s", 61 * time.Second, "1m 1s"},
		{"8m 15s", 8*time.Minute + 15*time.Second, "8m 15s"},
		{"60m no seconds", 60 * time.Minute, "60m"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := FormatElapsed(tt.d); got != tt.want {
				t.Errorf("FormatElapsed(%v) = %q, want %q", tt.d, got, tt.want)
			}
		})
	}
}

func TestFormatDeploymentSummary(t *testing.T) {
	t.Parallel()

	// All non-"started" verbs reference the same fixed "8s ago" start so
	// elapsed is deterministic across cases.
	fixed := time.Now().Add(-8 * time.Second)

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
			want: "deployed (8s) — v42, Linear50PercentEvery30Seconds, baking started",
		},
		{
			name: "complete with no addendum",
			verb: "complete", start: fixed, version: 7, strategy: "Canary10Percent20Minutes",
			want: "complete (8s) — v7, Canary10Percent20Minutes",
		},
		{
			name: "no version inserts strategy after em-dash",
			verb: "deployed", start: fixed, strategy: "AllAtOnce",
			want: "deployed (8s) — AllAtOnce",
		},
		{
			name: "started with no strategy / no addendum / no version",
			verb: "started",
			want: "started",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := FormatDeploymentSummary(tt.verb, tt.start, tt.version, tt.strategy, tt.addendum); got != tt.want {
				t.Errorf("FormatDeploymentSummary() = %q, want %q", got, tt.want)
			}
		})
	}
}
