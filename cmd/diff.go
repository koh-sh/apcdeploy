package cmd

import (
	"context"
	"fmt"

	"github.com/koh-sh/apcdeploy/internal/aws"
	"github.com/koh-sh/apcdeploy/internal/config"
	"github.com/koh-sh/apcdeploy/internal/diff"
	"github.com/koh-sh/apcdeploy/internal/display"
	"github.com/spf13/cobra"
)

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
		RunE: runDiff,
	}

	// Note: --config and --region flags are defined globally in root.go

	return cmd
}

func runDiff(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Load configuration
	configFile, err := cmd.Flags().GetString("config")
	if err != nil {
		return fmt.Errorf("failed to get config flag: %w", err)
	}

	cfg, err := config.LoadConfig(configFile)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Initialize AWS client
	region, err := cmd.Flags().GetString("region")
	if err != nil {
		return fmt.Errorf("failed to get region flag: %w", err)
	}
	if region != "" {
		cfg.Region = region
	}

	awsClient, err := aws.NewClient(ctx, cfg.Region)
	if err != nil {
		return fmt.Errorf("failed to initialize AWS client: %w", err)
	}

	// Resolve resources
	fmt.Println(display.Progress("Resolving resources..."))
	resolver := aws.NewResolver(awsClient)
	resources, err := resolver.ResolveAll(ctx, cfg.Application, cfg.ConfigurationProfile, cfg.Environment, cfg.DeploymentStrategy)
	if err != nil {
		return fmt.Errorf("failed to resolve resources: %w", err)
	}

	// Load local configuration data
	localData, err := config.LoadDataFile(cfg.DataFile)
	if err != nil {
		return fmt.Errorf("failed to load local configuration file: %w", err)
	}

	// Get latest deployment
	fmt.Println(display.Progress("Fetching latest deployment..."))
	deployment, err := aws.GetLatestDeployment(ctx, awsClient, resources.ApplicationID, resources.EnvironmentID, resources.Profile.ID)
	if err != nil {
		return fmt.Errorf("failed to get latest deployment: %w", err)
	}

	// Handle case when no deployment exists
	if deployment == nil {
		fmt.Println(display.Warning("No deployment found - this will be the initial deployment"))
		fmt.Println("\nLocal configuration:")
		fmt.Println(string(localData))
		fmt.Println()
		fmt.Println("Run 'apcdeploy deploy' to create the first deployment.")
		return nil
	}

	// Handle case when deployment is in progress
	if deployment.State == "DEPLOYING" || deployment.State == "BAKING" {
		fmt.Println(display.Warning(fmt.Sprintf("Deployment #%d is currently %s", deployment.DeploymentNumber, deployment.State)))
		fmt.Println("The diff will be calculated against the currently deploying version.")
		fmt.Println()
	}

	// Get remote configuration
	fmt.Println(display.Progress("Fetching deployed configuration..."))
	remoteData, err := aws.GetHostedConfigurationVersion(ctx, awsClient, resources.ApplicationID, resources.Profile.ID, deployment.ConfigurationVersion)
	if err != nil {
		return fmt.Errorf("failed to get deployed configuration: %w", err)
	}

	// Calculate diff
	diffResult, err := diff.Calculate(string(remoteData), string(localData), cfg.DataFile)
	if err != nil {
		return fmt.Errorf("failed to calculate diff: %w", err)
	}

	// Display diff
	diff.Display(diffResult, cfg, resources, deployment)

	return nil
}
