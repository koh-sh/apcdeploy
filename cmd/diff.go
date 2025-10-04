package cmd

import (
	"context"

	"github.com/koh-sh/apcdeploy/internal/cli"
	"github.com/koh-sh/apcdeploy/internal/diff"
	"github.com/spf13/cobra"
)

var (
	diffConfigFile string
	diffRegion     string
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
		RunE:         runDiff,
		SilenceUsage: true, // Don't show usage on runtime errors
	}

	cmd.Flags().StringVarP(&diffConfigFile, "config", "c", "apcdeploy.yml", "Path to configuration file")
	cmd.Flags().StringVar(&diffRegion, "region", "", "AWS region (overrides config file)")

	return cmd
}

func runDiff(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Create options
	opts := &diff.Options{
		ConfigFile: diffConfigFile,
		Region:     diffRegion,
	}

	// Create reporter
	reporter := cli.NewReporter()

	// Run diff
	executor := diff.NewExecutor(reporter)
	return executor.Execute(ctx, opts)
}
