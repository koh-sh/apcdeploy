package cmd

import (
	"context"
	"fmt"

	"github.com/koh-sh/apcdeploy/internal/aws"
	"github.com/koh-sh/apcdeploy/internal/deploy"
	"github.com/koh-sh/apcdeploy/internal/display"
	"github.com/spf13/cobra"
)

var (
	deployConfigFile string
	deployNoWait     bool
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
		RunE: runDeploy,
	}

	cmd.Flags().StringVarP(&deployConfigFile, "config", "c", "apcdeploy.yml", "Path to configuration file")
	cmd.Flags().BoolVar(&deployNoWait, "no-wait", false, "Do not wait for deployment to complete")
	cmd.Flags().IntVar(&deployTimeout, "timeout", 300, "Timeout in seconds for deployment (default: 300)")

	return cmd
}

func runDeploy(cmd *cobra.Command, args []string) error {
	// Validate timeout
	if deployTimeout < 0 {
		return fmt.Errorf("timeout must be a positive value")
	}

	ctx := context.Background()

	// Step 1: Load configuration
	display.Progress("Loading configuration...")
	cfg, dataContent, err := deploy.LoadConfiguration(deployConfigFile)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}
	display.Success("Configuration loaded")

	// Step 2: Create deployer
	deployer, err := deploy.New(ctx, cfg)
	if err != nil {
		return fmt.Errorf("failed to create deployer: %w", err)
	}

	// Step 3: Resolve resources
	display.Progress("Resolving AWS resources...")
	resolved, err := deployer.ResolveResources(ctx)
	if err != nil {
		return fmt.Errorf("failed to resolve resources: %w", err)
	}
	display.Success(fmt.Sprintf("Resolved resources: App=%s, Profile=%s, Env=%s, Strategy=%s",
		resolved.ApplicationID,
		resolved.Profile.ID,
		resolved.EnvironmentID,
		resolved.DeploymentStrategyID,
	))

	// Step 4: Check for ongoing deployments
	display.Progress("Checking for ongoing deployments...")
	hasOngoingDeployment, _, err := deployer.CheckOngoingDeployment(ctx, resolved)
	if err != nil {
		return fmt.Errorf("failed to check ongoing deployments: %w", err)
	}
	if hasOngoingDeployment {
		return fmt.Errorf("deployment already in progress")
	}
	display.Success("No ongoing deployments")

	// Step 5: Determine content type
	contentType, err := deployer.DetermineContentType(resolved.Profile.Type, cfg.DataFile)
	if err != nil {
		return fmt.Errorf("failed to determine content type: %w", err)
	}

	// Step 6: Validate local data
	display.Progress("Validating configuration data...")
	if err := deployer.ValidateLocalData(dataContent, contentType); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}
	display.Success("Configuration data validated")

	// Step 7: Create hosted configuration version
	display.Progress("Creating configuration version...")
	versionNumber, err := deployer.CreateVersion(ctx, resolved, dataContent, contentType)
	if err != nil {
		// Check if this is a validation error and provide user-friendly message
		if aws.IsValidationError(err) {
			return fmt.Errorf("%s", aws.FormatValidationError(err))
		}
		return fmt.Errorf("failed to create configuration version: %w", err)
	}
	display.Success(fmt.Sprintf("Created configuration version %d", versionNumber))

	// Step 8: Start deployment
	display.Progress("Starting deployment...")
	deploymentNumber, err := deployer.StartDeployment(ctx, resolved, versionNumber)
	if err != nil {
		return fmt.Errorf("failed to start deployment: %w", err)
	}
	display.Success(fmt.Sprintf("Deployment #%d started", deploymentNumber))

	// Step 9: Wait for deployment if requested
	if !deployNoWait {
		display.Progress("Waiting for deployment to complete...")
		if err := deployer.WaitForDeployment(ctx, resolved, deploymentNumber, deployTimeout); err != nil {
			return fmt.Errorf("deployment failed: %w", err)
		}
		display.Success("Deployment completed successfully")
	} else {
		fmt.Printf("\nDeployment #%d is in progress. Use 'apcdeploy status' to check the status.\n", deploymentNumber)
	}

	return nil
}
