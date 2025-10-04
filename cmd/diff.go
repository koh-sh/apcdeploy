package cmd

import (
	"context"

	"github.com/koh-sh/apcdeploy/internal/diff"
	"github.com/spf13/cobra"
)

// DiffCommand returns the diff command
func DiffCommand() *cobra.Command {
	return newDiffCmd()
}

func newDiffCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "diff",
		Short: "Show differences between local configuration and deployed configuration",
		Long: `Show differences between local configuration and the currently deployed configuration in AWS AppConfig.

This command compares your local configuration file with the latest deployed version
and displays the differences in unified diff format.`,
		RunE: runDiff,
	}

	// Note: --config and --region flags are defined globally in root.go

	return cmd
}

func runDiff(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Get flags
	configFile, err := cmd.Flags().GetString("config")
	if err != nil {
		return err
	}

	region, err := cmd.Flags().GetString("region")
	if err != nil {
		return err
	}

	// Create options
	opts := &diff.Options{
		ConfigFile: configFile,
		Region:     region,
	}

	// Create reporter
	reporter := &cliReporter{}

	// Run diff
	executor := diff.NewExecutor(reporter)
	return executor.Execute(ctx, opts)
}
