package run

import (
	"context"
	"fmt"

	"github.com/koh-sh/apcdeploy/internal/aws"
	"github.com/koh-sh/apcdeploy/internal/config"
	"github.com/koh-sh/apcdeploy/internal/reporter"
)

// Executor handles the deployment orchestration
type Executor struct {
	reporter        reporter.ProgressReporter
	deployerFactory func(context.Context, *config.Config) (*Deployer, error)
}

// NewExecutor creates a new deployment executor
func NewExecutor(rep reporter.ProgressReporter) *Executor {
	return &Executor{
		reporter:        rep,
		deployerFactory: New,
	}
}

// NewExecutorWithFactory creates a new deployment executor with a custom deployer factory
// This is useful for testing with mock deployers
func NewExecutorWithFactory(rep reporter.ProgressReporter, factory func(context.Context, *config.Config) (*Deployer, error)) *Executor {
	return &Executor{
		reporter:        rep,
		deployerFactory: factory,
	}
}

// Execute performs the complete deployment workflow
func (e *Executor) Execute(ctx context.Context, opts *Options) error {
	// Validate timeout
	if opts.Timeout < 0 {
		return fmt.Errorf("timeout must be a positive value")
	}

	// Step 1: Load configuration
	e.reporter.Progress("Loading configuration...")
	cfg, dataContent, err := loadConfiguration(opts.ConfigFile)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}
	e.reporter.Success("Configuration loaded")

	// Step 2: Create deployer
	deployer, err := e.deployerFactory(ctx, cfg)
	if err != nil {
		return fmt.Errorf("failed to create deployer: %w", err)
	}

	// Step 3: Resolve resources
	e.reporter.Progress("Resolving AWS resources...")
	resolved, err := deployer.ResolveResources(ctx)
	if err != nil {
		return fmt.Errorf("failed to resolve resources: %w", err)
	}
	e.reporter.Success(fmt.Sprintf("Resolved resources: App=%s, Profile=%s, Env=%s, Strategy=%s",
		resolved.ApplicationID,
		resolved.Profile.ID,
		resolved.EnvironmentID,
		resolved.DeploymentStrategyID,
	))

	// Step 4: Check for ongoing deployments
	e.reporter.Progress("Checking for ongoing deployments...")
	hasOngoingDeployment, _, err := deployer.CheckOngoingDeployment(ctx, resolved)
	if err != nil {
		return fmt.Errorf("failed to check ongoing deployments: %w", err)
	}
	if hasOngoingDeployment {
		return fmt.Errorf("deployment already in progress")
	}
	e.reporter.Success("No ongoing deployments")

	// Step 5: Determine content type
	contentType, err := deployer.DetermineContentType(resolved.Profile.Type, cfg.DataFile)
	if err != nil {
		return fmt.Errorf("failed to determine content type: %w", err)
	}

	// Step 6: Validate local data
	e.reporter.Progress("Validating configuration data...")
	if err := deployer.ValidateLocalData(dataContent, contentType); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}
	e.reporter.Success("Configuration data validated")

	// Step 7: Create hosted configuration version
	e.reporter.Progress("Creating configuration version...")
	versionNumber, err := deployer.CreateVersion(ctx, resolved, dataContent, contentType)
	if err != nil {
		// Check if this is a validation error and provide user-friendly message
		if aws.IsValidationError(err) {
			return fmt.Errorf("%s", aws.FormatValidationError(err))
		}
		return fmt.Errorf("failed to create configuration version: %w", err)
	}
	e.reporter.Success(fmt.Sprintf("Created configuration version %d", versionNumber))

	// Step 8: Start deployment
	e.reporter.Progress("Starting deployment...")
	deploymentNumber, err := deployer.StartDeployment(ctx, resolved, versionNumber)
	if err != nil {
		return fmt.Errorf("failed to start deployment: %w", err)
	}
	e.reporter.Success(fmt.Sprintf("Deployment #%d started", deploymentNumber))

	// Step 9: Wait for deployment if requested
	if opts.Wait {
		e.reporter.Progress("Waiting for deployment to complete...")
		if err := deployer.WaitForDeployment(ctx, resolved, deploymentNumber, opts.Timeout); err != nil {
			return fmt.Errorf("deployment failed: %w", err)
		}
		e.reporter.Success("Deployment completed successfully")
	} else {
		e.reporter.Warning(fmt.Sprintf("Deployment #%d is in progress. Use 'apcdeploy status' to check the status.", deploymentNumber))
	}

	return nil
}
