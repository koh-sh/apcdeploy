package cmd

import (
	"context"

	"github.com/koh-sh/apcdeploy/internal/cli"
	"github.com/koh-sh/apcdeploy/internal/get"
	"github.com/spf13/cobra"
)

var getConfigFile string

// GetCommand returns the get command
func GetCommand() *cobra.Command {
	return newGetCmd()
}

func newGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get configuration from AWS AppConfig",
		Long: `Get configuration from AWS AppConfig.

This command will:
1. Load the local configuration file (apcdeploy.yml)
2. Resolve AWS resources (application, environment, configuration profile)
3. Fetch the latest configuration from AppConfig
4. Output the configuration to stdout`,
		RunE:         runGet,
		SilenceUsage: true, // Don't show usage on runtime errors
	}

	cmd.Flags().StringVarP(&getConfigFile, "config", "c", "apcdeploy.yml", "Path to configuration file")

	return cmd
}

func runGet(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Create options
	opts := &get.Options{
		ConfigFile: getConfigFile,
	}

	// Create reporter
	reporter := cli.NewReporter()

	// Get configuration
	executor := get.NewExecutor(reporter)
	return executor.Execute(ctx, opts)
}
