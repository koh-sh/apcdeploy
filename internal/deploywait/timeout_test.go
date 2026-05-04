package deploywait

import (
	"testing"
	"time"
)

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
		{"current time clamps to 1s", now, time.Second},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := RemainingDuration(tt.deadline)
			if got < tt.wantMin {
				t.Errorf("RemainingDuration(%v) = %v, want >= %v", tt.deadline, got, tt.wantMin)
			}
		})
	}
}
