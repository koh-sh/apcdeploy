package status

import (
	"context"
	"fmt"
	"strconv"

	"github.com/koh-sh/apcdeploy/internal/aws"
	"github.com/koh-sh/apcdeploy/internal/config"
	"github.com/koh-sh/apcdeploy/internal/display"
	"github.com/koh-sh/apcdeploy/internal/reporter"
)

// Executor handles the status operation orchestration
type Executor struct {
	reporter      reporter.ProgressReporter
	clientFactory func(context.Context, string) (*aws.Client, error)
}

// NewExecutor creates a new status executor
func NewExecutor(rep reporter.ProgressReporter) *Executor {
	return &Executor{
		reporter:      rep,
		clientFactory: aws.NewClient,
	}
}

// NewExecutorWithFactory creates a new status executor with a custom client factory
// This is useful for testing with mock clients
func NewExecutorWithFactory(rep reporter.ProgressReporter, factory func(context.Context, string) (*aws.Client, error)) *Executor {
	return &Executor{
		reporter:      rep,
		clientFactory: factory,
	}
}

// Execute performs the status check workflow
func (e *Executor) Execute(ctx context.Context, opts *Options) error {
	// Step 1: Load configuration
	cfg, err := config.LoadConfig(opts.ConfigFile)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Step 2: Initialize AWS client
	awsClient, err := e.clientFactory(ctx, cfg.Region)
	if err != nil {
		return fmt.Errorf("failed to initialize AWS client: %w", err)
	}

	// Step 3: Resolve resources
	e.reporter.Progress("Resolving resources...")
	resolver := aws.NewResolver(awsClient)
	resources, err := resolver.ResolveAll(ctx, cfg.Application, cfg.ConfigurationProfile, cfg.Environment, cfg.DeploymentStrategy)
	if err != nil {
		return fmt.Errorf("failed to resolve resources: %w", err)
	}

	// Step 4: Get deployment information
	var deploymentInfo *aws.DeploymentDetails
	if opts.DeploymentID != "" {
		// Get specific deployment
		e.reporter.Progress(fmt.Sprintf("Fetching deployment #%s...", opts.DeploymentID))
		deploymentInfo, err = e.getDeploymentByID(ctx, awsClient, resources, opts.DeploymentID)
		if err != nil {
			return fmt.Errorf("failed to get deployment: %w", err)
		}
	} else {
		// Get latest deployment
		e.reporter.Progress("Fetching latest deployment...")
		deploymentInfo, err = e.getLatestDeployment(ctx, awsClient, resources)
		if err != nil {
			return fmt.Errorf("failed to get latest deployment: %w", err)
		}
	}

	// Step 5: Display status
	if deploymentInfo == nil {
		e.reporter.Warning("No deployments found")
		fmt.Println("\nNo deployments have been created yet for this configuration.")
		fmt.Println("\nNext steps:")
		fmt.Println("  1. Review your configuration file")
		fmt.Println("  2. Run 'apcdeploy deploy' to create your first deployment")
		return nil
	}

	display.ShowDeploymentStatus(deploymentInfo, cfg, resources)

	return nil
}

// getDeploymentByID retrieves a specific deployment by its ID
func (e *Executor) getDeploymentByID(ctx context.Context, client *aws.Client, resources *aws.ResolvedResources, deploymentID string) (*aws.DeploymentDetails, error) {
	// Parse deployment ID
	deploymentNumber, err := strconv.ParseInt(deploymentID, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid deployment ID: %s", deploymentID)
	}

	// Get deployment details
	deployment, err := aws.GetDeploymentDetails(ctx, client, resources.ApplicationID, resources.EnvironmentID, int32(deploymentNumber))
	if err != nil {
		return nil, err
	}

	// Check if deployment is for the target configuration profile
	if deployment.ConfigurationProfileID != resources.Profile.ID {
		return nil, fmt.Errorf("deployment #%d is not for configuration profile %s", deploymentNumber, resources.Profile.Name)
	}

	// Resolve deployment strategy name
	resolver := aws.NewResolver(client)
	strategyName, err := resolver.ResolveDeploymentStrategyIDToName(ctx, deployment.DeploymentStrategyID)
	if err != nil {
		// If we can't resolve the name, just use the ID
		strategyName = deployment.DeploymentStrategyID
	}
	deployment.DeploymentStrategyName = strategyName

	return deployment, nil
}

// getLatestDeployment retrieves the latest deployment for the configuration profile
// This includes ROLLED_BACK deployments for status command
func (e *Executor) getLatestDeployment(ctx context.Context, client *aws.Client, resources *aws.ResolvedResources) (*aws.DeploymentDetails, error) {
	deployment, err := aws.GetLatestDeploymentIncludingRollback(ctx, client, resources.ApplicationID, resources.EnvironmentID, resources.Profile.ID)
	if err != nil {
		return nil, err
	}

	if deployment == nil {
		return nil, nil
	}

	// Get full deployment details
	details, err := aws.GetDeploymentDetails(ctx, client, resources.ApplicationID, resources.EnvironmentID, deployment.DeploymentNumber)
	if err != nil {
		return nil, err
	}

	// Resolve deployment strategy name
	resolver := aws.NewResolver(client)
	strategyName, err := resolver.ResolveDeploymentStrategyIDToName(ctx, details.DeploymentStrategyID)
	if err != nil {
		// If we can't resolve the name, just use the ID
		strategyName = details.DeploymentStrategyID
	}
	details.DeploymentStrategyName = strategyName

	return details, nil
}
