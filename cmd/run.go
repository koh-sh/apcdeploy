package cmd

import (
	"context"

	"github.com/koh-sh/apcdeploy/internal/cli"
	"github.com/koh-sh/apcdeploy/internal/run"
	"github.com/spf13/cobra"
)

var (
	runConfigFile string
	runWait       bool
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
5. Optionally wait for the deployment to complete`,
		RunE:         runRun,
		SilenceUsage: true, // Don't show usage on runtime errors
	}

	cmd.Flags().StringVarP(&runConfigFile, "config", "c", "apcdeploy.yml", "Path to configuration file")
	cmd.Flags().BoolVar(&runWait, "wait", false, "Wait for deployment to complete")
	cmd.Flags().IntVar(&runTimeout, "timeout", 600, "Timeout in seconds for deployment")
	cmd.Flags().BoolVar(&runForce, "force", false, "Force deployment even when there are no changes")

	return cmd
}

func runRun(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Create options
	opts := &run.Options{
		ConfigFile: runConfigFile,
		Wait:       runWait,
		Timeout:    runTimeout,
		Force:      runForce,
		Silent:     IsSilent(),
	}

	// Create reporter
	reporter := cli.GetReporter(IsSilent())

	// Run deployment
	executor := run.NewExecutor(reporter)
	return executor.Execute(ctx, opts)
}
