package cmd

import (
	"context"

	"github.com/koh-sh/apcdeploy/internal/cli"
	"github.com/koh-sh/apcdeploy/internal/status"
	"github.com/spf13/cobra"
)

var statusDeploymentID string

// StatusCommand returns the status command
func StatusCommand() *cobra.Command {
	return newStatusCmd()
}

func newStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show deployment status",
		Long: `Show the status of deployments in AWS AppConfig.

This command displays information about the latest deployment or a specific deployment
identified by deployment number.`,
		RunE:         runStatus,
		SilenceUsage: true, // Don't show usage on runtime errors
	}

	cmd.Flags().StringVar(&statusDeploymentID, "deployment", "", "Deployment number to check (defaults to latest)")

	return cmd
}

func runStatus(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Create options
	opts := &status.Options{
		ConfigFile:   configFile,
		DeploymentID: statusDeploymentID,
		Silent:       isSilent(),
	}

	// Create reporter
	reporter := cli.GetReporter(isSilent())

	// Run status check
	executor := status.NewExecutor(reporter)
	return executor.Execute(ctx, opts)
}
