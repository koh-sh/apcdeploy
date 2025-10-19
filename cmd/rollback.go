package cmd

import (
	"context"

	"github.com/koh-sh/apcdeploy/internal/cli"
	"github.com/koh-sh/apcdeploy/internal/prompt"
	"github.com/koh-sh/apcdeploy/internal/rollback"
	"github.com/spf13/cobra"
)

var rollbackSkipConfirmation bool

// RollbackCommand returns the rollback command
func RollbackCommand() *cobra.Command {
	return newRollbackCmd()
}

func newRollbackCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rollback",
		Short: "Stop an ongoing deployment",
		Long: `Stop an ongoing deployment in AWS AppConfig.

This command stops an in-progress deployment by calling the AWS AppConfig StopDeployment API.
It automatically finds the current ongoing deployment and stops it.`,
		RunE:         runRollback,
		SilenceUsage: true, // Don't show usage on runtime errors
	}

	cmd.Flags().BoolVarP(&rollbackSkipConfirmation, "yes", "y", false, "Skip confirmation prompt")

	return cmd
}

func runRollback(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Create options
	opts := &rollback.Options{
		ConfigFile:       configFile,
		Silent:           isSilent(),
		SkipConfirmation: rollbackSkipConfirmation,
	}

	// Create reporter and prompter
	reporter := cli.GetReporter(isSilent())
	prompter := &prompt.HuhPrompter{}

	// Run rollback
	executor := rollback.NewExecutor(reporter, prompter)
	return executor.Execute(ctx, opts)
}
