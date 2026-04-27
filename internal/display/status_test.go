package display

import (
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/appconfig/types"
	"github.com/koh-sh/apcdeploy/internal/aws"
	"github.com/koh-sh/apcdeploy/internal/config"
	mockreporter "github.com/koh-sh/apcdeploy/internal/reporter/testing"
)

func TestDeploymentStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		deployment   *aws.DeploymentDetails
		cfg          *config.Config
		resources    *aws.ResolvedResources
		wantStdout   string
		wantTableHas []string // substrings expected in the main table rows
		wantHeaders  []string // headers expected via Reporter.Header
		wantWarn     bool
		wantWarnText string
		denyTableHas []string // substrings that must NOT appear in the main table
	}{
		{
			name: "completed deployment",
			deployment: &aws.DeploymentDetails{
				DeploymentNumber:     1,
				State:                types.DeploymentStateComplete,
				ConfigurationVersion: "v1.0.0",
				Description:          "Test deployment",
				StartedAt:            new(time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)),
				CompletedAt:          new(time.Date(2024, 1, 1, 10, 5, 0, 0, time.UTC)),
				PercentageComplete:   100,
			},
			cfg: &config.Config{Application: "test-app", Environment: "test-env"},
			resources: &aws.ResolvedResources{
				Profile: &aws.ProfileInfo{Name: "test-profile"},
			},
			wantStdout:   "COMPLETE\n",
			wantHeaders:  []string{"Deployment Status"},
			wantTableHas: []string{"test-app", "test-profile", "test-env", "v1.0.0", "Test deployment", "COMPLETE"},
		},
		{
			name: "deploying deployment shows progress section",
			deployment: &aws.DeploymentDetails{
				DeploymentNumber:       2,
				State:                  types.DeploymentStateDeploying,
				ConfigurationVersion:   "v2.0.0",
				StartedAt:              new(time.Now().Add(-5 * time.Minute)),
				PercentageComplete:     30,
				GrowthFactor:           10,
				FinalBakeTimeInMinutes: 5,
			},
			cfg:          &config.Config{Application: "test-app", Environment: "prod-env"},
			resources:    &aws.ResolvedResources{Profile: &aws.ProfileInfo{Name: "prod-profile"}},
			wantStdout:   "DEPLOYING\n",
			wantHeaders:  []string{"Deployment Status", "Progress"},
			wantTableHas: []string{"DEPLOYING", "v2.0.0", "30.0%", "10.0%", "5 minutes"},
		},
		{
			name: "baking deployment shows progress section",
			deployment: &aws.DeploymentDetails{
				DeploymentNumber:       3,
				State:                  types.DeploymentStateBaking,
				ConfigurationVersion:   "v3.0.0",
				StartedAt:              new(time.Now().Add(-10 * time.Minute)),
				PercentageComplete:     100,
				FinalBakeTimeInMinutes: 5,
			},
			cfg:          &config.Config{Application: "test-app", Environment: "staging"},
			resources:    &aws.ResolvedResources{Profile: &aws.ProfileInfo{Name: "stage-profile"}},
			wantStdout:   "BAKING\n",
			wantHeaders:  []string{"Deployment Status", "Progress"},
			wantTableHas: []string{"BAKING", "v3.0.0", "100.0%", "Baking (monitoring for issues)"},
		},
		{
			name: "rolled back deployment surfaces warn + reason",
			deployment: &aws.DeploymentDetails{
				DeploymentNumber:       4,
				State:                  types.DeploymentStateRolledBack,
				ConfigurationVersion:   "v4.0.0",
				Description:            "Deploying new configuration",
				DeploymentStrategyName: "AppConfig.Linear50PercentEvery30Seconds",
				StartedAt:              new(time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)),
				CompletedAt:            new(time.Date(2024, 1, 1, 10, 2, 0, 0, time.UTC)),
				EventLog: []types.DeploymentEvent{
					{
						EventType:   types.DeploymentEventTypeRollbackStarted,
						Description: new("Rollback initiated by CloudWatch Alarm"),
					},
				},
			},
			cfg:          &config.Config{Application: "test-app", Environment: "prod"},
			resources:    &aws.ResolvedResources{Profile: &aws.ProfileInfo{Name: "prod-profile"}},
			wantStdout:   "ROLLED_BACK\n",
			wantHeaders:  []string{"Deployment Status"},
			wantTableHas: []string{"ROLLED_BACK", "AppConfig.Linear50PercentEvery30Seconds"},
			denyTableHas: []string{"Deploying new configuration"}, // Description suppressed for ROLLED_BACK
			wantWarn:     true,
			wantWarnText: "Deployment was rolled back",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := &mockreporter.MockReporter{}
			DeploymentStatus(r, tt.deployment, tt.cfg, tt.resources)

			if got := string(r.Stdout); got != tt.wantStdout {
				t.Errorf("stdout payload = %q, want %q", got, tt.wantStdout)
			}

			for _, h := range tt.wantHeaders {
				if !r.HasMessage("header: " + h) {
					t.Errorf("expected Header(%q) to be emitted; messages=%v", h, r.Messages)
				}
			}

			tableContent := flattenTables(r.Tables)
			for _, want := range tt.wantTableHas {
				if !strings.Contains(tableContent, want) {
					t.Errorf("expected table cells to contain %q; got %q", want, tableContent)
				}
			}
			for _, deny := range tt.denyTableHas {
				if strings.Contains(tableContent, deny) {
					t.Errorf("expected table cells NOT to contain %q; got %q", deny, tableContent)
				}
			}

			if tt.wantWarn {
				if !r.HasMessage("warn: " + tt.wantWarnText) {
					t.Errorf("expected Warn(%q); messages=%v", tt.wantWarnText, r.Messages)
				}
			}
		})
	}
}

func flattenTables(calls []mockreporter.TableCall) string {
	var sb strings.Builder
	for _, c := range calls {
		for _, row := range c.Rows {
			for _, cell := range row {
				sb.WriteString(cell)
				sb.WriteByte('|')
			}
		}
	}
	return sb.String()
}

func TestFormatTime(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		time     time.Time
		wantDate string
	}{
		{"specific time", time.Date(2024, 1, 15, 14, 30, 45, 0, time.UTC), "2024-01-15"},
		{"another time", time.Date(2024, 12, 31, 12, 0, 0, 0, time.UTC), "2024-12-31"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := formatTime(tt.time)
			if !strings.Contains(got, tt.wantDate) {
				t.Errorf("formatTime() = %v, want to contain %v", got, tt.wantDate)
			}
			if !strings.Contains(got, ":") {
				t.Errorf("formatTime() = %v, want to contain time separator ':'", got)
			}
		})
	}
}

func TestFormatDuration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		duration time.Duration
		want     string
	}{
		{"seconds only", 45 * time.Second, "45s"},
		{"minutes and seconds", 3*time.Minute + 30*time.Second, "3m 30s"},
		{"hours, minutes and seconds", 2*time.Hour + 15*time.Minute + 45*time.Second, "2h 15m 45s"},
		{"zero duration", 0, "0s"},
		{"one hour exactly", 1 * time.Hour, "1h 0m 0s"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := formatDuration(tt.duration); got != tt.want {
				t.Errorf("formatDuration() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFormatCurrentPhase(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		deployment *aws.DeploymentDetails
		want       string
	}{
		{"baking", &aws.DeploymentDetails{State: types.DeploymentStateBaking}, "Baking (monitoring for issues)"},
		{"starting", &aws.DeploymentDetails{State: types.DeploymentStateDeploying, PercentageComplete: 10}, "Starting deployment"},
		{"initial rollout", &aws.DeploymentDetails{State: types.DeploymentStateDeploying, PercentageComplete: 30}, "Initial rollout phase"},
		{"mid rollout", &aws.DeploymentDetails{State: types.DeploymentStateDeploying, PercentageComplete: 60}, "Mid rollout phase"},
		{"final rollout", &aws.DeploymentDetails{State: types.DeploymentStateDeploying, PercentageComplete: 80}, "Final rollout phase"},
		{"completing", &aws.DeploymentDetails{State: types.DeploymentStateDeploying, PercentageComplete: 100}, "Completing deployment"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := formatCurrentPhase(tt.deployment); got != tt.want {
				t.Errorf("formatCurrentPhase() = %v, want %v", got, tt.want)
			}
		})
	}
}
