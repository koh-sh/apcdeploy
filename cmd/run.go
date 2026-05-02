package cmd

import (
	"context"
	"fmt"

	"github.com/koh-sh/apcdeploy/internal/cli"
	"github.com/koh-sh/apcdeploy/internal/run"
	"github.com/spf13/cobra"
)

const (
	// DefaultDeploymentTimeout is the default timeout for deployments in seconds.
	// Set to 30 minutes to safely cover AppConfig.AllAtOnce (10 min bake) and
	// AppConfig.Canary10Percent20Minutes (20 min deploy + 10 min bake) under
	// --wait-bake. Strategies with longer total durations (e.g.
	// AppConfig.Linear20PercentEvery6Minutes) require an explicit --timeout.
	DefaultDeploymentTimeout = 1800
)

var (
	runWaitDeploy  bool
	runWaitBake    bool
	runTimeout     int
	runForce       bool
	runDescription string
)

// RunCommand returns the run command
func RunCommand() *cobra.Command {
	return newRunCmd()
}

func newRunCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run configuration deployment to AWS AppConfig",
		Long: `Run configuration deployment to AWS AppConfig.

This command will:
1. Load the local configuration file (apcdeploy.yml)
2. Validate the configuration data
3. Create a new hosted configuration version
4. Start a deployment to the specified environment
5. Optionally wait for the deployment phase (--wait-deploy) or full completion (--wait-bake)`,
		RunE:         runRun,
		SilenceUsage: true, // Don't show usage on runtime errors
	}

	cmd.Flags().BoolVar(&runWaitDeploy, "wait-deploy", false, "Wait for deployment phase to complete (until baking starts)")
	cmd.Flags().BoolVar(&runWaitBake, "wait-bake", false, "Wait for complete deployment including baking phase")
	cmd.Flags().IntVar(&runTimeout, "timeout", DefaultDeploymentTimeout, "Timeout in seconds for deployment")
	cmd.Flags().BoolVar(&runForce, "force", false, "Force deployment even when there are no changes")
	cmd.Flags().StringVar(&runDescription, "description", "", fmt.Sprintf(`Description attached to the configuration version and deployment (max %d chars; defaults to %q, pass "" to clear)`, maxDescriptionLength, defaultDescription))

	return cmd
}

func runRun(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	if err := validateDescription(runDescription); err != nil {
		return err
	}
	description := resolveDescription(cmd, runDescription)

	opts := &run.Options{
		ConfigFile:  configFile,
		WaitDeploy:  runWaitDeploy,
		WaitBake:    runWaitBake,
		Timeout:     runTimeout,
		Force:       runForce,
		Description: description,
	}

	reporter := cli.GetReporter(isSilent())

	executor := run.NewExecutor(reporter)
	return executor.Execute(ctx, opts)
}
