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
		{"sub-second rounds up at 0.5s", 1500 * time.Millisecond, "2s"},
		{"sub-second rounds down below 0.5s", 1499 * time.Millisecond, "1s"},
		{"under 0.5s rounds to zero", 499 * time.Millisecond, "0s"},
		{"45s", 45 * time.Second, "45s"},
		{"59s", 59 * time.Second, "59s"},
		{"exactly one minute", time.Minute, "1m"},
		{"1m 1s preserves seconds", 61 * time.Second, "1m 1s"},
		{"6m + sub-second jitter rounds nearest", 6*time.Minute + 400*time.Millisecond, "6m"},
		{"6m + 600ms jitter rounds up to 6m 1s", 6*time.Minute + 600*time.Millisecond, "6m 1s"},
		{"8m 15s preserves seconds", 8*time.Minute + 15*time.Second, "8m 15s"},
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

	tests := []struct {
		name     string
		verb     string
		elapsed  time.Duration
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
			verb: "deployed", elapsed: 8 * time.Second, version: 42, strategy: "Linear50PercentEvery30Seconds", addendum: "baking started",
			want: "deployed (8s) — v42, Linear50PercentEvery30Seconds, baking started",
		},
		{
			name: "complete with no addendum",
			verb: "complete", elapsed: 8 * time.Second, version: 7, strategy: "Canary10Percent20Minutes",
			want: "complete (8s) — v7, Canary10Percent20Minutes",
		},
		{
			name: "no version inserts strategy after em-dash",
			verb: "deployed", elapsed: 8 * time.Second, strategy: "AllAtOnce",
			want: "deployed (8s) — AllAtOnce",
		},
		{
			name: "started with no strategy / no addendum / no version",
			verb: "started",
			want: "started",
		},
		{
			name: "zero elapsed omits the (...) suffix even for non-started verb",
			verb: "complete", version: 1, strategy: "AllAtOnce",
			want: "complete — v1, AllAtOnce",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := FormatDeploymentSummary(tt.verb, tt.elapsed, tt.version, tt.strategy, tt.addendum); got != tt.want {
				t.Errorf("FormatDeploymentSummary() = %q, want %q", got, tt.want)
			}
		})
	}
}
