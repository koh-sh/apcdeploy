package display

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/appconfig/types"
	"github.com/koh-sh/apcdeploy/internal/aws"
	"github.com/koh-sh/apcdeploy/internal/config"
)

func captureOutput(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	return buf.String()
}

func TestShowDeploymentStatus(t *testing.T) {
	tests := []struct {
		name       string
		deployment *aws.DeploymentDetails
		cfg        *config.Config
		resources  *aws.ResolvedResources
		wantText   []string
	}{
		{
			name: "completed deployment",
			deployment: &aws.DeploymentDetails{
				DeploymentNumber:     1,
				State:                types.DeploymentStateComplete,
				ConfigurationVersion: "v1.0.0",
				Description:          "Test deployment",
				StartedAt:            ptrTime(time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)),
				CompletedAt:          ptrTime(time.Date(2024, 1, 1, 10, 5, 0, 0, time.UTC)),
				PercentageComplete:   100,
			},
			cfg: &config.Config{
				Application: "test-app",
				Environment: "test-env",
			},
			resources: &aws.ResolvedResources{
				Profile: &aws.ProfileInfo{Name: "test-profile"},
			},
			wantText: []string{
				"Deployment Status",
				"Application:   test-app",
				"Profile:       test-profile",
				"Environment:   test-env",
				"Deployment #:  1",
				"COMPLETE",
				"Version:       v1.0.0",
				"Description:   Test deployment",
				"Started:",
				"Completed:",
				"Duration:",
			},
		},
		{
			name: "deploying deployment",
			deployment: &aws.DeploymentDetails{
				DeploymentNumber:       2,
				State:                  types.DeploymentStateDeploying,
				ConfigurationVersion:   "v2.0.0",
				StartedAt:              ptrTime(time.Now().Add(-5 * time.Minute)),
				PercentageComplete:     30,
				GrowthFactor:           10,
				FinalBakeTimeInMinutes: 5,
			},
			cfg: &config.Config{
				Application: "test-app",
				Environment: "prod-env",
			},
			resources: &aws.ResolvedResources{
				Profile: &aws.ProfileInfo{Name: "prod-profile"},
			},
			wantText: []string{
				"Deployment Status",
				"Application:   test-app",
				"Profile:       prod-profile",
				"Environment:   prod-env",
				"Deployment #:  2",
				"DEPLOYING",
				"Version:       v2.0.0",
				"Progress",
				"Percentage:    30.0%",
				"Elapsed:",
				"Estimated:",
				"Growth Factor: 10.0%",
				"Bake Time:     5 minutes",
				"Current Phase:",
			},
		},
		{
			name: "baking deployment",
			deployment: &aws.DeploymentDetails{
				DeploymentNumber:     3,
				State:                types.DeploymentStateBaking,
				ConfigurationVersion: "v3.0.0",
				StartedAt:            ptrTime(time.Now().Add(-10 * time.Minute)),
				PercentageComplete:   100,
			},
			cfg: &config.Config{
				Application: "test-app",
				Environment: "staging",
			},
			resources: &aws.ResolvedResources{
				Profile: &aws.ProfileInfo{Name: "staging-profile"},
			},
			wantText: []string{
				"Deployment Status",
				"BAKING",
				"Progress",
				"Current Phase: Baking",
			},
		},
		{
			name: "rolled back deployment",
			deployment: &aws.DeploymentDetails{
				DeploymentNumber:     4,
				State:                types.DeploymentStateRolledBack,
				ConfigurationVersion: "v4.0.0",
				StartedAt:            ptrTime(time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)),
				CompletedAt:          ptrTime(time.Date(2024, 1, 1, 10, 2, 0, 0, time.UTC)),
			},
			cfg: &config.Config{
				Application: "test-app",
				Environment: "prod",
			},
			resources: &aws.ResolvedResources{
				Profile: &aws.ProfileInfo{Name: "prod-profile"},
			},
			wantText: []string{
				"Deployment Status",
				"ROLLED_BACK",
				"Deployment was rolled back",
				"Check CloudWatch Alarms",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := captureOutput(func() {
				ShowDeploymentStatus(tt.deployment, tt.cfg, tt.resources)
			})

			for _, want := range tt.wantText {
				if !strings.Contains(output, want) {
					t.Errorf("ShowDeploymentStatus() output missing %q\nGot:\n%s", want, output)
				}
			}
		})
	}
}

func TestFormatDeploymentState(t *testing.T) {
	tests := []struct {
		name  string
		state types.DeploymentState
		want  string
	}{
		{
			name:  "complete state",
			state: types.DeploymentStateComplete,
			want:  "COMPLETE",
		},
		{
			name:  "deploying state",
			state: types.DeploymentStateDeploying,
			want:  "DEPLOYING",
		},
		{
			name:  "baking state",
			state: types.DeploymentStateBaking,
			want:  "BAKING",
		},
		{
			name:  "rolled back state",
			state: types.DeploymentStateRolledBack,
			want:  "ROLLED_BACK",
		},
		{
			name:  "unknown state",
			state: types.DeploymentState("UNKNOWN"),
			want:  "UNKNOWN",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatDeploymentState(tt.state)
			if !strings.Contains(got, tt.want) {
				t.Errorf("formatDeploymentState() = %v, want to contain %v", got, tt.want)
			}
		})
	}
}

func TestFormatTime(t *testing.T) {
	tests := []struct {
		name     string
		time     time.Time
		wantDate string
	}{
		{
			name:     "specific time",
			time:     time.Date(2024, 1, 15, 14, 30, 45, 0, time.UTC),
			wantDate: "2024-01-15",
		},
		{
			name:     "another time",
			time:     time.Date(2024, 12, 31, 12, 0, 0, 0, time.UTC),
			wantDate: "2024-12-31",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatTime(tt.time)
			// Just check that the output contains the date (time zone may vary)
			if !strings.Contains(got, tt.wantDate) {
				t.Errorf("formatTime() = %v, want to contain date %v", got, tt.wantDate)
			}
			// Verify format includes time component
			if !strings.Contains(got, ":") {
				t.Errorf("formatTime() = %v, want to contain time separator ':'", got)
			}
		})
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		want     string
	}{
		{
			name:     "seconds only",
			duration: 45 * time.Second,
			want:     "45s",
		},
		{
			name:     "minutes and seconds",
			duration: 3*time.Minute + 30*time.Second,
			want:     "3m 30s",
		},
		{
			name:     "hours, minutes and seconds",
			duration: 2*time.Hour + 15*time.Minute + 45*time.Second,
			want:     "2h 15m 45s",
		},
		{
			name:     "zero duration",
			duration: 0,
			want:     "0s",
		},
		{
			name:     "one hour exactly",
			duration: 1 * time.Hour,
			want:     "1h 0m 0s",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatDuration(tt.duration)
			if got != tt.want {
				t.Errorf("formatDuration() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFormatCurrentPhase(t *testing.T) {
	tests := []struct {
		name       string
		deployment *aws.DeploymentDetails
		want       string
	}{
		{
			name: "baking state",
			deployment: &aws.DeploymentDetails{
				State: types.DeploymentStateBaking,
			},
			want: "Baking (monitoring for issues)",
		},
		{
			name: "starting deployment",
			deployment: &aws.DeploymentDetails{
				State:              types.DeploymentStateDeploying,
				PercentageComplete: 10,
			},
			want: "Starting deployment",
		},
		{
			name: "initial rollout phase",
			deployment: &aws.DeploymentDetails{
				State:              types.DeploymentStateDeploying,
				PercentageComplete: 30,
			},
			want: "Initial rollout phase",
		},
		{
			name: "mid rollout phase",
			deployment: &aws.DeploymentDetails{
				State:              types.DeploymentStateDeploying,
				PercentageComplete: 60,
			},
			want: "Mid rollout phase",
		},
		{
			name: "final rollout phase",
			deployment: &aws.DeploymentDetails{
				State:              types.DeploymentStateDeploying,
				PercentageComplete: 80,
			},
			want: "Final rollout phase",
		},
		{
			name: "completing deployment",
			deployment: &aws.DeploymentDetails{
				State:              types.DeploymentStateDeploying,
				PercentageComplete: 100,
			},
			want: "Completing deployment",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatCurrentPhase(tt.deployment)
			if got != tt.want {
				t.Errorf("formatCurrentPhase() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBold(t *testing.T) {
	tests := []struct {
		name string
		text string
		want string
	}{
		{
			name: "simple text",
			text: "Hello",
			want: "Hello",
		},
		{
			name: "text with spaces",
			text: "Hello World",
			want: "Hello World",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Bold(tt.text)
			if !strings.Contains(got, tt.want) {
				t.Errorf("Bold() = %v, want to contain %v", got, tt.want)
			}
		})
	}
}

func TestSeparator(t *testing.T) {
	result := Separator()
	if len(result) == 0 {
		t.Error("Separator() returned empty string")
	}
}

func ptrTime(t time.Time) *time.Time {
	return &t
}
