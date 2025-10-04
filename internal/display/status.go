package display

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/appconfig/types"
	"github.com/koh-sh/apcdeploy/internal/aws"
	"github.com/koh-sh/apcdeploy/internal/config"
)

// ShowDeploymentStatus displays detailed deployment status information
func ShowDeploymentStatus(deployment *aws.DeploymentDetails, cfg *config.Config, resources *aws.ResolvedResources) {
	// Header
	fmt.Println("\n" + Bold("Deployment Status"))
	fmt.Println(Separator())

	// Configuration information
	fmt.Printf("  Application:   %s\n", cfg.Application)
	fmt.Printf("  Profile:       %s\n", resources.Profile.Name)
	fmt.Printf("  Environment:   %s\n", cfg.Environment)
	fmt.Println()

	// Deployment information
	fmt.Printf("  Deployment #:  %d\n", deployment.DeploymentNumber)
	fmt.Printf("  Status:        %s\n", formatDeploymentState(deployment.State))
	fmt.Printf("  Version:       %s\n", deployment.ConfigurationVersion)

	// Show description only for non-rolled-back deployments
	if deployment.State != types.DeploymentStateRolledBack && deployment.Description != "" {
		fmt.Printf("  Description:   %s\n", deployment.Description)
	}

	// Show deployment strategy
	if deployment.DeploymentStrategyName != "" {
		fmt.Printf("  Strategy:      %s\n", deployment.DeploymentStrategyName)
	}

	// Timing information
	if deployment.StartedAt != nil {
		fmt.Printf("  Started:       %s\n", formatTime(*deployment.StartedAt))
	}

	if deployment.CompletedAt != nil {
		fmt.Printf("  Completed:     %s\n", formatTime(*deployment.CompletedAt))
		if deployment.StartedAt != nil {
			duration := deployment.CompletedAt.Sub(*deployment.StartedAt)
			fmt.Printf("  Duration:      %s\n", formatDuration(duration))
		}
	}

	// Progress information (for in-progress deployments)
	if deployment.State == types.DeploymentStateDeploying || deployment.State == types.DeploymentStateBaking {
		fmt.Println()
		fmt.Println(Bold("  Progress"))
		fmt.Printf("  Percentage:    %.1f%%\n", deployment.PercentageComplete)

		if deployment.StartedAt != nil {
			elapsed := time.Since(*deployment.StartedAt)
			fmt.Printf("  Elapsed:       %s\n", formatDuration(elapsed))

			// Estimate remaining time
			if deployment.PercentageComplete > 0 {
				estimatedTotal := time.Duration(float64(elapsed) / float64(deployment.PercentageComplete) * 100)
				remaining := estimatedTotal - elapsed
				if remaining > 0 {
					fmt.Printf("  Estimated:     %s remaining\n", formatDuration(remaining))
				}
			}
		}

		// Deployment strategy information
		if deployment.GrowthFactor > 0 {
			fmt.Printf("  Growth Factor: %.1f%%\n", deployment.GrowthFactor)
		}
		if deployment.FinalBakeTimeInMinutes > 0 {
			fmt.Printf("  Bake Time:     %d minutes\n", deployment.FinalBakeTimeInMinutes)
		}

		// Current phase
		fmt.Printf("\n  Current Phase: %s\n", formatCurrentPhase(deployment))
	}

	// Rollback information
	if deployment.State == types.DeploymentStateRolledBack {
		fmt.Println()
		fmt.Println(Error("  Deployment was rolled back"))

		// Try to find rollback reason from event log
		rollbackReason := getRollbackReason(deployment.EventLog)
		if rollbackReason != "" {
			fmt.Printf("  Reason:        %s\n", rollbackReason)
		}
	}

	fmt.Println()
}

// formatDeploymentState formats the deployment state with appropriate styling
func formatDeploymentState(state types.DeploymentState) string {
	switch state {
	case types.DeploymentStateComplete:
		return Success("COMPLETE")
	case types.DeploymentStateDeploying:
		return Warning("DEPLOYING")
	case types.DeploymentStateBaking:
		return Warning("BAKING")
	case types.DeploymentStateRolledBack:
		return Error("ROLLED_BACK")
	default:
		return string(state)
	}
}

// formatTime formats a time.Time for display
func formatTime(t time.Time) string {
	return t.Local().Format("2006-01-02 15:04:05 MST")
}

// formatDuration formats a duration for display
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

// formatCurrentPhase determines the current phase of deployment
func formatCurrentPhase(deployment *aws.DeploymentDetails) string {
	if deployment.State == types.DeploymentStateBaking {
		return "Baking (monitoring for issues)"
	}

	percentage := deployment.PercentageComplete
	if percentage >= 100 {
		return "Completing deployment"
	}
	if percentage >= 75 {
		return "Final rollout phase"
	}
	if percentage >= 50 {
		return "Mid rollout phase"
	}
	if percentage >= 25 {
		return "Initial rollout phase"
	}
	return "Starting deployment"
}

// getRollbackReason extracts the rollback reason from deployment event log
func getRollbackReason(eventLog []types.DeploymentEvent) string {
	// Look for rollback events in reverse order (most recent first)
	for i := len(eventLog) - 1; i >= 0; i-- {
		event := eventLog[i]
		// Check for rollback-related events
		if event.EventType == types.DeploymentEventTypeRollbackStarted ||
			event.EventType == types.DeploymentEventTypeRollbackCompleted {
			if event.Description != nil && *event.Description != "" {
				return *event.Description
			}
		}
	}
	return ""
}
