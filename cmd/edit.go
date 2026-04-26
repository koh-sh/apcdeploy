package cmd

import (
	"context"

	"github.com/koh-sh/apcdeploy/internal/cli"
	"github.com/koh-sh/apcdeploy/internal/edit"
	"github.com/koh-sh/apcdeploy/internal/prompt"
	"github.com/spf13/cobra"
)

var (
	editRegion             string
	editApp                string
	editProfile            string
	editEnv                string
	editDeploymentStrategy string
	editWaitDeploy         bool
	editWaitBake           bool
	editTimeout            int
)

// EditCommand returns the edit command
func EditCommand() *cobra.Command {
	return newEditCmd()
}

func newEditCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "edit",
		Short: "Edit the deployed configuration directly in $EDITOR and deploy",
		Long: `Edit the currently deployed AppConfig configuration directly in your $EDITOR.

This command fetches the latest deployed configuration, opens it in $EDITOR
(falling back to vi), and deploys the result. It does NOT use apcdeploy.yml —
the target is selected via flags or interactive prompts, similar to 'init'.

If --deployment-strategy is omitted, the strategy of the most recent deployment
is reused. Validation behavior matches the 'run' command (size limits and
JSON/YAML syntax checks).`,
		RunE:         runEdit,
		SilenceUsage: true,
	}

	cmd.Flags().StringVar(&editRegion, "region", "", "AWS region")
	cmd.Flags().StringVar(&editApp, "app", "", "Application name")
	cmd.Flags().StringVar(&editProfile, "profile", "", "Configuration Profile name")
	cmd.Flags().StringVar(&editEnv, "env", "", "Environment name")
	cmd.Flags().StringVar(&editDeploymentStrategy, "deployment-strategy", "", "Deployment strategy name (defaults to the strategy of the latest deployment)")
	cmd.Flags().BoolVar(&editWaitDeploy, "wait-deploy", false, "Wait for deployment phase to complete (until baking starts)")
	cmd.Flags().BoolVar(&editWaitBake, "wait-bake", false, "Wait for complete deployment including baking phase")
	cmd.Flags().IntVar(&editTimeout, "timeout", DefaultDeploymentTimeout, "Timeout in seconds for deployment")

	return cmd
}

func runEdit(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	opts := &edit.Options{
		Region:             editRegion,
		Application:        editApp,
		Profile:            editProfile,
		Environment:        editEnv,
		DeploymentStrategy: editDeploymentStrategy,
		WaitDeploy:         editWaitDeploy,
		WaitBake:           editWaitBake,
		Timeout:            editTimeout,
		Silent:             isSilent(),
	}

	reporter := cli.GetReporter(isSilent())
	prompter := &prompt.HuhPrompter{}

	executor := edit.NewExecutor(reporter, prompter)
	return executor.Execute(ctx, opts)
}
