package deploywait

import (
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/appconfig/types"
	"github.com/koh-sh/apcdeploy/internal/batch"
	reportertest "github.com/koh-sh/apcdeploy/internal/reporter/testing"
)

// TestMakeTargetsDeployTick verifies that progress / phase finalisation
// branches route correctly: DEPLOYING reports the live percent + ETA via
// SetProgress, while BAKING / COMPLETE pin to 1.0 (the row's caller swaps
// to the baking sub-phase or finalises Done immediately afterwards).
func TestMakeTargetsDeployTick(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		state         types.DeploymentState
		percent       float64
		totalDuration time.Duration
		wantPercent   float64
		// wantETAZero asserts ETA == 0. wantETAPositive asserts ETA > 0.
		// Tests fire the tick immediately after closure construction so
		// elapsed is effectively zero — for non-baking states the ETA
		// approximates totalDuration. We only assert sign rather than an
		// exact duration to avoid flakes from waitStart drift.
		wantETAZero     bool
		wantETAPositive bool
	}{
		{
			name: "deploying mid", state: types.DeploymentStateDeploying,
			percent: 42.5, totalDuration: 10 * time.Minute,
			wantPercent: 0.425, wantETAPositive: true,
		},
		{
			name: "deploying low", state: types.DeploymentStateDeploying,
			percent: 25, totalDuration: 8 * time.Minute,
			wantPercent: 0.25, wantETAPositive: true,
		},
		{
			name:  "AllAtOnce-style zero totalDuration → ETA 0",
			state: types.DeploymentStateDeploying, percent: 50, totalDuration: 0,
			wantPercent: 0.5, wantETAZero: true,
		},
		{
			name: "baking pins to 1.0 with ETA 0", state: types.DeploymentStateBaking,
			percent: 30, totalDuration: 10 * time.Minute,
			wantPercent: 1.0, wantETAZero: true,
		},
		{
			name: "complete pins to 1.0 with ETA 0", state: types.DeploymentStateComplete,
			percent: 100, totalDuration: 10 * time.Minute,
			wantPercent: 1.0, wantETAZero: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := &reportertest.MockReporter{}
			tg := m.Targets([]string{"id"})
			tick := MakeTargetsDeployTick(batch.NewTargetReporter(tg, "id"))
			tick(tt.state, tt.percent, tt.totalDuration)
			tg.Close()

			progressTr := []reportertest.TargetsTransition{}
			for _, call := range m.TargetsCalls {
				for _, tr := range call.Transitions {
					if tr.Kind == "progress" {
						progressTr = append(progressTr, tr)
					}
				}
			}
			if len(progressTr) != 1 {
				t.Fatalf("expected 1 progress transition; got %+v", progressTr)
			}
			if delta := progressTr[0].Percent - tt.wantPercent; delta < -0.001 || delta > 0.001 {
				t.Errorf("percent = %v, want %v", progressTr[0].Percent, tt.wantPercent)
			}
			switch {
			case tt.wantETAZero && progressTr[0].ETA != 0:
				t.Errorf("ETA = %v, want 0", progressTr[0].ETA)
			case tt.wantETAPositive && progressTr[0].ETA <= 0:
				t.Errorf("ETA = %v, want > 0", progressTr[0].ETA)
			}
		})
	}
}

// TestMakeTargetsBakeTick checks that bake ticks set the row's baking
// sub-phase with a "(~N min left)" countdown derived from elapsed/total.
func TestMakeTargetsBakeTick(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		elapsed    time.Duration
		total      time.Duration
		wantDetail string
	}{
		{"zero total falls back to <1 min", 0, 0, " (<1 min left)"},
		{"early in bake shows full window", 0, 10 * time.Minute, " (~10 min left)"},
		{"mid bake shows remaining", 5 * time.Minute, 10 * time.Minute, " (~5 min left)"},
		{"elapsed exceeds total clamps to <1 min", 11 * time.Minute, 10 * time.Minute, " (<1 min left)"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := &reportertest.MockReporter{}
			tg := m.Targets([]string{"id"})
			tick := MakeTargetsBakeTick(batch.NewTargetReporter(tg, "id"))
			tick(tt.elapsed, tt.total)
			tg.Close()

			var detail string
			for _, call := range m.TargetsCalls {
				for _, tr := range call.Transitions {
					if tr.Kind == "phase" && tr.Phase == "baking" {
						detail = tr.Detail
					}
				}
			}
			if detail == "" {
				t.Fatalf("expected baking phase transition; got %+v", m.TargetsCalls)
			}
			if detail != tt.wantDetail {
				t.Errorf("baking detail = %q, want %q", detail, tt.wantDetail)
			}
		})
	}
}

// TestRemainingFromElapsedSuffix exercises the time-boundary edge cases
// (sub-minute, exact minute, overshoot, zero total) directly.
func TestRemainingFromElapsedSuffix(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		elapsed time.Duration
		total   time.Duration
		want    string
	}{
		{"zero total falls back to <1 min", 0, 0, " (<1 min left)"},
		{"negative-looking total falls back to <1 min", 0, -5 * time.Minute, " (<1 min left)"},
		{"start of window shows full duration", 0, 10 * time.Minute, " (~10 min left)"},
		{"mid window rounds up partial minute", 3 * time.Minute, 10 * time.Minute, " (~7 min left)"},
		{"exactly one minute remaining renders as 1 min", 9 * time.Minute, 10 * time.Minute, " (~1 min left)"},
		{"non-integer remaining rounds up", 0, 2*time.Minute + 30*time.Second, " (~3 min left)"},
		{"thirty seconds remaining clamps to <1 min", 9*time.Minute + 30*time.Second, 10 * time.Minute, " (<1 min left)"},
		{"sub-second remaining clamps to <1 min", 9*time.Minute + 59*time.Second + 500*time.Millisecond, 10 * time.Minute, " (<1 min left)"},
		{"elapsed equals total clamps to <1 min", 10 * time.Minute, 10 * time.Minute, " (<1 min left)"},
		{"elapsed exceeds total clamps to <1 min", 12 * time.Minute, 10 * time.Minute, " (<1 min left)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := remainingFromElapsedSuffix(tt.elapsed, tt.total); got != tt.want {
				t.Errorf("remainingFromElapsedSuffix(%v, %v) = %q, want %q", tt.elapsed, tt.total, got, tt.want)
			}
		})
	}
}
