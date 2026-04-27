package display

import (
	"fmt"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/appconfig/types"
	"github.com/koh-sh/apcdeploy/internal/aws"
	"github.com/koh-sh/apcdeploy/internal/cli"
	"github.com/koh-sh/apcdeploy/internal/config"
	"github.com/koh-sh/apcdeploy/internal/reporter"
)

// DeploymentStatus renders the status of a deployment through the Reporter.
//
// The deployment state is always written to stdout via Reporter.Data so
// scripts can consume it under --silent. The header / table / progress / box
// sections are written via Reporter primitives, which the silent variant
// suppresses automatically — callers MUST NOT branch on opts.Silent.
func DeploymentStatus(r reporter.Reporter, deployment *aws.DeploymentDetails, cfg *config.Config, resources *aws.ResolvedResources) {
	// Machine-readable payload: deployment state on stdout.
	r.Data([]byte(string(deployment.State) + "\n"))

	// Human-facing summary on stderr (suppressed in silent mode).
	r.Header("Deployment Status")

	rows := [][]string{
		{"Application", cfg.Application},
		{"Profile", resources.Profile.Name},
		{"Environment", cfg.Environment},
		{"Deployment #", strconv.Itoa(int(deployment.DeploymentNumber))},
		{"Status", cli.StateBadge(string(deployment.State))},
		{"Version", deployment.ConfigurationVersion},
	}
	if deployment.State != types.DeploymentStateRolledBack && deployment.Description != "" {
		rows = append(rows, []string{"Description", deployment.Description})
	}
	if deployment.DeploymentStrategyName != "" {
		rows = append(rows, []string{"Strategy", deployment.DeploymentStrategyName})
	}
	if deployment.StartedAt != nil {
		rows = append(rows, []string{"Started", formatTime(*deployment.StartedAt)})
	}
	if deployment.CompletedAt != nil {
		rows = append(rows, []string{"Completed", formatTime(*deployment.CompletedAt)})
		if deployment.StartedAt != nil {
			duration := deployment.CompletedAt.Sub(*deployment.StartedAt)
			rows = append(rows, []string{"Duration", formatDuration(duration)})
		}
	}
	r.Table([]string{"Field", "Value"}, rows)

	if deployment.State == types.DeploymentStateDeploying || deployment.State == types.DeploymentStateBaking {
		r.Header("Progress")
		progressRows := [][]string{
			{"Percentage", fmt.Sprintf("%.1f%%", deployment.PercentageComplete)},
		}
		if deployment.StartedAt != nil {
			elapsed := time.Since(*deployment.StartedAt)
			progressRows = append(progressRows, []string{"Elapsed", formatDuration(elapsed)})
			if deployment.PercentageComplete > 0 {
				estimatedTotal := time.Duration(float64(elapsed) / float64(deployment.PercentageComplete) * 100)
				if remaining := estimatedTotal - elapsed; remaining > 0 {
					progressRows = append(progressRows, []string{"Estimated", formatDuration(remaining) + " remaining"})
				}
			}
		}
		if deployment.GrowthFactor > 0 {
			progressRows = append(progressRows, []string{"Growth Factor", fmt.Sprintf("%.1f%%", deployment.GrowthFactor)})
		}
		if deployment.FinalBakeTimeInMinutes > 0 {
			progressRows = append(progressRows, []string{"Bake Time", fmt.Sprintf("%d minutes", deployment.FinalBakeTimeInMinutes)})
		}
		progressRows = append(progressRows, []string{"Current Phase", formatCurrentPhase(deployment)})
		r.Table([]string{"Field", "Value"}, progressRows)
	}

	if deployment.State == types.DeploymentStateRolledBack {
		r.Warn("Deployment was rolled back")
		if reason := aws.ExtractRollbackReason(deployment.EventLog); reason != "" {
			r.Info("Reason: " + reason)
		}
	}
}

// formatTime formats a time.Time for display.
func formatTime(t time.Time) string {
	return t.Local().Format("2006-01-02 15:04:05 MST")
}

// formatDuration formats a duration for display.
func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second

	if h > 0 {
		return fmt.Sprintf("%dh %dm %ds", h, m, s)
	}
	if m > 0 {
		return fmt.Sprintf("%dm %ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}

// formatCurrentPhase determines the current phase of deployment.
func formatCurrentPhase(deployment *aws.DeploymentDetails) string {
	if deployment.State == types.DeploymentStateBaking {
		return "Baking (monitoring for issues)"
	}

	percentage := deployment.PercentageComplete
	switch {
	case percentage >= 100:
		return "Completing deployment"
	case percentage >= 75:
		return "Final rollout phase"
	case percentage >= 50:
		return "Mid rollout phase"
	case percentage >= 25:
		return "Initial rollout phase"
	default:
		return "Starting deployment"
	}
}
