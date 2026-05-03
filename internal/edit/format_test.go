package edit

import (
	"testing"
	"time"

	awsInternal "github.com/koh-sh/apcdeploy/internal/aws"
)

func TestFormatElapsed(t *testing.T) {
	t.Parallel()

	tests := []struct {
		d    time.Duration
		want string
	}{
		{0, "0s"},
		{45 * time.Second, "45s"},
		{time.Minute, "1m"},
		{8*time.Minute + 15*time.Second, "8m 15s"},
	}
	for _, tt := range tests {
		if got := formatElapsed(tt.d); got != tt.want {
			t.Errorf("formatElapsed(%v) = %q, want %q", tt.d, got, tt.want)
		}
	}
}

func TestFormatEditSummary(t *testing.T) {
	t.Parallel()

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
			name: "started omits elapsed",
			verb: "started", version: 43, strategy: "AppConfig.AllAtOnce", addendum: "deployment #9",
			want: "started — v43, AppConfig.AllAtOnce, deployment #9",
		},
		{
			name: "deployed with elapsed",
			verb: "deployed", start: fixed, version: 43, strategy: "AllAtOnce", addendum: "baking started",
			want: "deployed (8s) — v43, AllAtOnce, baking started",
		},
		{
			name: "complete with elapsed and no addendum",
			verb: "complete", start: fixed, version: 5, strategy: "Linear50PercentEvery30Seconds",
			want: "complete (8s) — v5, Linear50PercentEvery30Seconds",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := formatEditSummary(tt.verb, tt.start, tt.version, tt.strategy, tt.addendum); got != tt.want {
				t.Errorf("formatEditSummary() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestResolvedTargetsIdentifier(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		region string
		t      *resolvedTargets
		want   string
	}{
		{
			name:   "uses profile name from resolver",
			region: "us-east-1",
			t: &resolvedTargets{
				AppName: "app",
				EnvName: "prod",
				Profile: &awsInternal.ProfileInfo{Name: "feature-flags"},
			},
			want: "us-east-1/app/feature-flags/prod",
		},
		{
			name:   "different region",
			region: "eu-west-1",
			t: &resolvedTargets{
				AppName: "my-app",
				EnvName: "dev",
				Profile: &awsInternal.ProfileInfo{Name: "config"},
			},
			want: "eu-west-1/my-app/config/dev",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.t.Identifier(tt.region); got != tt.want {
				t.Errorf("Identifier(%q) = %q, want %q", tt.region, got, tt.want)
			}
		})
	}
}

func TestRemainingDuration(t *testing.T) {
	t.Parallel()

	now := time.Now()
	tests := []struct {
		name     string
		deadline time.Time
		wantMin  time.Duration
	}{
		{"future deadline", now.Add(30 * time.Second), 20 * time.Second},
		{"past deadline clamps to 1s", now.Add(-1 * time.Second), time.Second},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := remainingDuration(tt.deadline)
			if got < tt.wantMin {
				t.Errorf("remainingDuration(%v) = %v, want >= %v", tt.deadline, got, tt.wantMin)
			}
		})
	}
}
