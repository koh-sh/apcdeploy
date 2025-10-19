package cmd

import (
	"context"

	"github.com/koh-sh/apcdeploy/internal/cli"
	"github.com/koh-sh/apcdeploy/internal/pull"
	"github.com/spf13/cobra"
)

// PullCommand returns the pull command
func PullCommand() *cobra.Command {
	return newPullCmd()
}

func newPullCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pull",
		Short: "Pull latest deployed configuration and update local data file",
		Long: `Pull the latest deployed configuration from AWS AppConfig and update the local data file.

This command retrieves the currently deployed configuration and overwrites your local data file.
Useful when configuration changes are made directly in the AWS Console and you want to sync
your local files with the deployed state.

Note: This command does NOT use the AppConfig Data API, so it does not incur per-call charges.`,
		RunE:         runPull,
		SilenceUsage: true, // Don't show usage on runtime errors
	}

	return cmd
}

func runPull(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Create options
	opts := &pull.Options{
		ConfigFile: configFile,
		Silent:     isSilent(),
	}

	// Create reporter
	reporter := cli.GetReporter(isSilent())

	// Pull configuration
	executor := pull.NewExecutor(reporter)
	return executor.Execute(ctx, opts)
}
