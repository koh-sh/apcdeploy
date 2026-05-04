package deploywait

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/appconfig"
	"github.com/aws/aws-sdk-go-v2/service/appconfig/types"
	awsInternal "github.com/koh-sh/apcdeploy/internal/aws"
	"github.com/koh-sh/apcdeploy/internal/aws/mock"
)

// timestamps used by the table tests; aliased for brevity.
var (
	tStart = time.Date(2024, 1, 2, 10, 0, 0, 0, time.UTC)
	tBake  = time.Date(2024, 1, 2, 10, 5, 0, 0, time.UTC)
	tDone  = time.Date(2024, 1, 2, 10, 12, 0, 0, time.UTC)
)

func TestAWSElapsedForDeploy(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		// out, err returned from the mocked GetDeployment call.
		out *appconfig.GetDeploymentOutput
		err error
		// fallback is set well in the past so the wall-clock fallback
		// branch produces a duration we can sign-check (>0).
		fallback time.Time
		// wantExact, when non-zero, asserts an exact AWS-derived elapsed.
		// When zero, the test only asserts that the result is positive
		// (the wall-clock fallback path).
		wantExact time.Duration
	}{
		{
			name: "AWS-derived elapsed from BAKE_TIME_STARTED minus StartedAt",
			out: &appconfig.GetDeploymentOutput{
				StartedAt: &tStart,
				EventLog: []types.DeploymentEvent{
					{EventType: types.DeploymentEventTypeBakeTimeStarted, OccurredAt: &tBake},
				},
			},
			fallback:  time.Now().Add(-time.Hour),
			wantExact: 5 * time.Minute,
		},
		{
			name:     "GetDeployment error → wall-clock fallback",
			err:      errors.New("api boom"),
			fallback: time.Now().Add(-30 * time.Second),
		},
		{
			name: "no BAKE_TIME_STARTED event yet → wall-clock fallback",
			out: &appconfig.GetDeploymentOutput{
				StartedAt: &tStart,
				EventLog:  nil,
			},
			fallback: time.Now().Add(-30 * time.Second),
		},
		{
			name: "StartedAt nil → wall-clock fallback",
			out: &appconfig.GetDeploymentOutput{
				StartedAt: nil,
				EventLog: []types.DeploymentEvent{
					{EventType: types.DeploymentEventTypeBakeTimeStarted, OccurredAt: &tBake},
				},
			},
			fallback: time.Now().Add(-30 * time.Second),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := &mock.MockAppConfigClient{
				GetDeploymentFunc: func(_ context.Context, _ *appconfig.GetDeploymentInput, _ ...func(*appconfig.Options)) (*appconfig.GetDeploymentOutput, error) {
					return tt.out, tt.err
				},
			}
			client := awsInternal.NewTestClient(m)
			got := AWSElapsedForDeploy(context.Background(), client, "app", "env", 1, tt.fallback)
			if tt.wantExact != 0 {
				if got != tt.wantExact {
					t.Errorf("AWSElapsedForDeploy = %v, want %v", got, tt.wantExact)
				}
			} else if got <= 0 {
				t.Errorf("AWSElapsedForDeploy = %v, want > 0 (wall-clock fallback)", got)
			}
		})
	}
}

func TestAWSElapsedForBake(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		out       *appconfig.GetDeploymentOutput
		err       error
		fallback  time.Time
		wantExact time.Duration
	}{
		{
			name: "AWS-derived total from CompletedAt minus StartedAt",
			out: &appconfig.GetDeploymentOutput{
				StartedAt:   &tStart,
				CompletedAt: &tDone,
			},
			fallback:  time.Now().Add(-time.Hour),
			wantExact: 12 * time.Minute,
		},
		{
			name:     "GetDeployment error → wall-clock fallback",
			err:      errors.New("api boom"),
			fallback: time.Now().Add(-30 * time.Second),
		},
		{
			name: "CompletedAt nil → wall-clock fallback",
			out: &appconfig.GetDeploymentOutput{
				StartedAt:   &tStart,
				CompletedAt: nil,
			},
			fallback: time.Now().Add(-30 * time.Second),
		},
		{
			name: "StartedAt nil → wall-clock fallback",
			out: &appconfig.GetDeploymentOutput{
				StartedAt:   nil,
				CompletedAt: &tDone,
			},
			fallback: time.Now().Add(-30 * time.Second),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := &mock.MockAppConfigClient{
				GetDeploymentFunc: func(_ context.Context, _ *appconfig.GetDeploymentInput, _ ...func(*appconfig.Options)) (*appconfig.GetDeploymentOutput, error) {
					return tt.out, tt.err
				},
			}
			client := awsInternal.NewTestClient(m)
			got := AWSElapsedForBake(context.Background(), client, "app", "env", 1, tt.fallback)
			if tt.wantExact != 0 {
				if got != tt.wantExact {
					t.Errorf("AWSElapsedForBake = %v, want %v", got, tt.wantExact)
				}
			} else if got <= 0 {
				t.Errorf("AWSElapsedForBake = %v, want > 0 (wall-clock fallback)", got)
			}
		})
	}
}
