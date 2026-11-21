package cmd

import (
	"errors"

	"github.com/koh-sh/apcdeploy/internal/cli"
	"github.com/koh-sh/apcdeploy/internal/prompt"
	"github.com/koh-sh/apcdeploy/internal/rollback"
	"github.com/spf13/cobra"
)

var (
	rollbackSkipConfirmation bool
	rollbackDeployment       int32
)

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
It automatically finds the current ongoing deployment in the environment and stops it,
but only if that deployment belongs to the configuration profile in the config file.
A deployment of another profile is refused (even with --yes); re-run with a config
targeting that profile instead.

Pass --deployment N to stop only deployment #N: if a different deployment is in
flight, nothing is stopped. Use it with --yes so the deployment checked with
'apcdeploy status' is the one stopped.`,
		RunE:         runRollback,
		SilenceUsage: true, // Don't show usage on runtime errors
	}

	cmd.Flags().BoolVarP(&rollbackSkipConfirmation, "yes", "y", false, "Skip confirmation prompt")
	cmd.Flags().Int32Var(&rollbackDeployment, "deployment", 0, "Only stop this deployment number; refuse if another deployment is in flight")

	return cmd
}

func runRollback(cmd *cobra.Command, args []string) error {
	ctx := commandContext(cmd)

	cfgFile, err := requireSingleConfig("rollback")
	if err != nil {
		return err
	}
	if cmd.Flags().Changed("deployment") && rollbackDeployment <= 0 {
		return errors.New("--deployment must be a positive deployment number")
	}

	// Create options
	opts := &rollback.Options{
		ConfigFile:         cfgFile,
		SkipConfirmation:   rollbackSkipConfirmation,
		ExpectedDeployment: rollbackDeployment,
	}

	// Create reporter and prompter
	reporter := cli.GetReporter(isSilent())
	prompter := &prompt.HuhPrompter{}

	// Run rollback
	executor := rollback.NewExecutor(reporter, prompter)
	return executor.Execute(ctx, opts)
}
