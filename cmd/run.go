package cmd

import (
	"context"

	"github.com/koh-sh/apcdeploy/internal/cli"
	"github.com/koh-sh/apcdeploy/internal/run"
	"github.com/spf13/cobra"
)

const (
	// DefaultDeploymentTimeout is the default timeout for deployments in seconds
	DefaultDeploymentTimeout = 600
)

var (
	runWaitDeploy bool
	runWaitBake   bool
	runTimeout    int
	runForce      bool
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

	return cmd
}

func runRun(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Create options
	opts := &run.Options{
		ConfigFile: configFile,
		WaitDeploy: runWaitDeploy,
		WaitBake:   runWaitBake,
		Timeout:    runTimeout,
		Force:      runForce,
		Silent:     isSilent(),
	}

	// Create reporter
	reporter := cli.GetReporter(isSilent())

	// Run deployment
	executor := run.NewExecutor(reporter)
	return executor.Execute(ctx, opts)
}
