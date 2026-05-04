package aws

import (
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/appconfig/types"
)

func TestBakeTimeStartedAt(t *testing.T) {
	t.Parallel()

	now := time.Now()
	earlier := now.Add(-30 * time.Second)

	tests := []struct {
		name    string
		details *DeploymentDetails
		want    *time.Time
	}{
		{
			name:    "nil details returns nil",
			details: nil,
			want:    nil,
		},
		{
			name:    "empty event log returns nil",
			details: &DeploymentDetails{},
			want:    nil,
		},
		{
			name: "event log without BAKE_TIME_STARTED returns nil",
			details: &DeploymentDetails{
				EventLog: []types.DeploymentEvent{
					{EventType: types.DeploymentEventTypeDeploymentStarted, OccurredAt: &earlier},
					{EventType: types.DeploymentEventTypePercentageUpdated, OccurredAt: &now},
				},
			},
			want: nil,
		},
		{
			name: "BAKE_TIME_STARTED event timestamp is returned",
			details: &DeploymentDetails{
				EventLog: []types.DeploymentEvent{
					{EventType: types.DeploymentEventTypeDeploymentStarted, OccurredAt: &earlier},
					{EventType: types.DeploymentEventTypeBakeTimeStarted, OccurredAt: &now},
				},
			},
			want: &now,
		},
		{
			name: "BAKE_TIME_STARTED event without OccurredAt returns nil",
			details: &DeploymentDetails{
				EventLog: []types.DeploymentEvent{
					{EventType: types.DeploymentEventTypeBakeTimeStarted, OccurredAt: nil},
				},
			},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := BakeTimeStartedAt(tt.details)
			switch {
			case tt.want == nil && got != nil:
				t.Errorf("BakeTimeStartedAt() = %v, want nil", got)
			case tt.want != nil && got == nil:
				t.Errorf("BakeTimeStartedAt() = nil, want %v", tt.want)
			case tt.want != nil && got != nil && !got.Equal(*tt.want):
				t.Errorf("BakeTimeStartedAt() = %v, want %v", got, tt.want)
			}
		})
	}
}
