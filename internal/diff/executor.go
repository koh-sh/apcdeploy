package diff

import (
	"context"
	"errors"
	"fmt"

	"github.com/koh-sh/apcdeploy/internal/aws"
	"github.com/koh-sh/apcdeploy/internal/config"
	"github.com/koh-sh/apcdeploy/internal/reporter"
)

// ErrDiffFound is returned when differences are found and ExitNonzero is true
var ErrDiffFound = errors.New("differences found")

// Executor handles the diff operation orchestration
type Executor struct {
	reporter      reporter.ProgressReporter
	clientFactory func(context.Context, string) (*aws.Client, error)
}

// NewExecutor creates a new diff executor
func NewExecutor(rep reporter.ProgressReporter) *Executor {
	return &Executor{
		reporter:      rep,
		clientFactory: aws.NewClient,
	}
}

// NewExecutorWithFactory creates a new diff executor with a custom client factory
// This is useful for testing with mock clients
func NewExecutorWithFactory(rep reporter.ProgressReporter, factory func(context.Context, string) (*aws.Client, error)) *Executor {
	return &Executor{
		reporter:      rep,
		clientFactory: factory,
	}
}

// Execute performs the complete diff workflow
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

	// Step 4: Load local configuration data
	localData, err := config.LoadDataFile(cfg.DataFile)
	if err != nil {
		return fmt.Errorf("failed to load local configuration file: %w", err)
	}

	// Step 5: Get latest deployment
	e.reporter.Progress("Fetching latest deployment...")
	deployment, err := aws.GetLatestDeployment(ctx, awsClient, resources.ApplicationID, resources.EnvironmentID, resources.Profile.ID)
	if err != nil {
		return fmt.Errorf("failed to get latest deployment: %w", err)
	}

	// Step 6: Handle case when no deployment exists
	if deployment == nil {
		e.reporter.Warning("No deployment found - this will be the initial deployment")
		fmt.Println("\nLocal configuration:")
		fmt.Println(string(localData))
		fmt.Println()
		fmt.Println("Run 'apcdeploy deploy' to create the first deployment.")
		return nil
	}

	// Step 7: Handle case when deployment is in progress
	if deployment.State == "DEPLOYING" || deployment.State == "BAKING" {
		fmt.Println()
		e.reporter.Warning(fmt.Sprintf("Deployment #%d is currently %s", deployment.DeploymentNumber, deployment.State))
		fmt.Println("The diff will be calculated against the currently deploying version.")
		fmt.Println()
	}

	// Step 8: Get remote configuration
	e.reporter.Progress("Fetching deployed configuration...")
	remoteData, err := aws.GetHostedConfigurationVersion(ctx, awsClient, resources.ApplicationID, resources.Profile.ID, deployment.ConfigurationVersion)
	if err != nil {
		return fmt.Errorf("failed to get deployed configuration: %w", err)
	}

	// Step 9: Calculate diff
	diffResult, err := calculate(string(remoteData), string(localData), cfg.DataFile, resources.Profile.Type)
	if err != nil {
		return fmt.Errorf("failed to calculate diff: %w", err)
	}

	// Step 10: Display diff
	if opts.Silent {
		DisplaySilent(diffResult)
	} else {
		display(diffResult, cfg, resources, deployment)
	}

	// Step 11: Return error if differences found and ExitNonzero is set
	if opts.ExitNonzero && diffResult.HasChanges {
		return ErrDiffFound
	}

	return nil
}
