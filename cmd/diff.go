package cmd

import (
	"context"
	"errors"
	"os"

	"github.com/koh-sh/apcdeploy/internal/cli"
	"github.com/koh-sh/apcdeploy/internal/diff"
	"github.com/spf13/cobra"
)

var diffExitNonzero bool

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

	cmd.Flags().BoolVar(&diffExitNonzero, "exit-nonzero", false, "Exit with code 1 if differences exist")

	return cmd
}

func runDiff(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Create options
	opts := &diff.Options{
		ConfigFile:  configFile,
		ExitNonzero: diffExitNonzero,
		Silent:      isSilent(),
	}

	// Create reporter
	reporter := cli.GetReporter(isSilent())

	// Run diff
	executor := diff.NewExecutor(reporter)
	err := executor.Execute(ctx, opts)

	// Handle exit-nonzero case
	if errors.Is(err, diff.ErrDiffFound) {
		os.Exit(1)
	}

	return err
}
