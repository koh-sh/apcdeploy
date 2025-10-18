package cmd

import (
	"context"

	"github.com/koh-sh/apcdeploy/internal/cli"
	"github.com/koh-sh/apcdeploy/internal/lsresources"
	"github.com/spf13/cobra"
)

var (
	// lsResourcesRegion is the AWS region for listing resources
	lsResourcesRegion string
	// lsResourcesJSON enables JSON output format
	lsResourcesJSON bool
	// lsResourcesShowStrategies enables displaying deployment strategies
	lsResourcesShowStrategies bool
)

// LsResourcesCommand returns the ls-resources command
func LsResourcesCommand() *cobra.Command {
	return newLsResourcesCmd()
}

func newLsResourcesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ls-resources",
		Short: "List AWS AppConfig resources (applications, profiles, environments)",
		Long: `List all AWS AppConfig resources in a hierarchical view, including applications,
configuration profiles, and environments in the specified region.

This is useful for discovering available resources before running 'init' command,
especially for AI agents that cannot use interactive prompts.`,
		RunE:         runLsResources,
		SilenceUsage: true, // Don't show usage on runtime errors
	}

	cmd.Flags().StringVar(&lsResourcesRegion, "region", "", "AWS region (uses AWS SDK default if not specified)")
	cmd.Flags().BoolVar(&lsResourcesJSON, "json", false, "Output in JSON format")
	cmd.Flags().BoolVar(&lsResourcesShowStrategies, "show-strategies", false, "Include deployment strategies in output")

	return cmd
}

func runLsResources(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Create options
	opts := &lsresources.Options{
		Region:         lsResourcesRegion,
		JSON:           lsResourcesJSON,
		ShowStrategies: lsResourcesShowStrategies,
		Silent:         isSilent(),
	}

	// Create reporter
	reporter := cli.GetReporter(isSilent())

	// Execute
	executor := lsresources.NewExecutor(reporter)
	return executor.Execute(ctx, opts)
}
