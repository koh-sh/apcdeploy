package cmd

import (
	"context"

	"github.com/koh-sh/apcdeploy/internal/cli"
	"github.com/koh-sh/apcdeploy/internal/deploy"
	"github.com/spf13/cobra"
)

var (
	deployConfigFile string
	deployWait       bool
	deployTimeout    int
)

// DeployCommand returns the deploy command
func DeployCommand() *cobra.Command {
	return newDeployCmd()
}

func newDeployCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deploy",
		Short: "Deploy configuration to AWS AppConfig",
		Long: `Deploy configuration data to AWS AppConfig.

This command will:
1. Load the local configuration file (apcdeploy.yml)
2. Validate the configuration data
3. Create a new hosted configuration version
4. Start a deployment to the specified environment
5. Optionally wait for the deployment to complete`,
		RunE:         runDeploy,
		SilenceUsage: true, // Don't show usage on runtime errors
	}

	cmd.Flags().StringVarP(&deployConfigFile, "config", "c", "apcdeploy.yml", "Path to configuration file")
	cmd.Flags().BoolVar(&deployWait, "wait", false, "Wait for deployment to complete")
	cmd.Flags().IntVar(&deployTimeout, "timeout", 600, "Timeout in seconds for deployment")

	return cmd
}

func runDeploy(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Create options
	opts := &deploy.Options{
		ConfigFile: deployConfigFile,
		Wait:       deployWait,
		Timeout:    deployTimeout,
	}

	// Create reporter
	reporter := cli.NewReporter()

	// Run deployment
	executor := deploy.NewExecutor(reporter)
	return executor.Execute(ctx, opts)
}
