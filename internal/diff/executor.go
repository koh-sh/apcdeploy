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
	reporter      reporter.Reporter
	clientFactory func(context.Context, string) (*aws.Client, error)
}

// NewExecutor creates a new diff executor
func NewExecutor(rep reporter.Reporter) *Executor {
	return &Executor{
		reporter:      rep,
		clientFactory: aws.NewClient,
	}
}

// NewExecutorWithFactory creates a new diff executor with a custom client factory
// This is useful for testing with mock clients
func NewExecutorWithFactory(rep reporter.Reporter, factory func(context.Context, string) (*aws.Client, error)) *Executor {
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
	e.reporter.Step("Resolving resources...")
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
	e.reporter.Step("Fetching latest deployment...")
	deployment, err := aws.GetLatestDeployment(ctx, awsClient, resources.ApplicationID, resources.EnvironmentID, resources.Profile.ID)
	if err != nil {
		return fmt.Errorf("failed to get latest deployment: %w", err)
	}

	// Step 6: Handle case when no deployment exists. Emit the local data as
	// the stdout payload (acts as the "right side" of the would-be diff) and
	// surface the next-step hint via the Reporter so silent mode suppresses
	// the human-facing parts automatically.
	if deployment == nil {
		e.reporter.Warn("No deployment found - this will be the initial deployment")
		e.reporter.Header("Local configuration")
		e.reporter.Data(localData)
		if len(localData) > 0 && localData[len(localData)-1] != '\n' {
			e.reporter.Data([]byte("\n"))
		}
		e.reporter.Info("Run 'apcdeploy run' to create the first deployment.")
		return nil
	}

	// Step 7: Get remote configuration
	e.reporter.Step("Fetching deployed configuration...")
	remoteData, err := aws.GetHostedConfigurationVersion(ctx, awsClient, resources.ApplicationID, resources.Profile.ID, deployment.ConfigurationVersion)
	if err != nil {
		return fmt.Errorf("failed to get deployed configuration: %w", err)
	}

	// Step 8: Calculate diff
	diffResult, err := calculate(string(remoteData), string(localData), cfg.DataFile, resources.Profile.Type)
	if err != nil {
		return fmt.Errorf("failed to calculate diff: %w", err)
	}

	// Step 9: Display diff (silent mode is handled by the Reporter).
	display(e.reporter, diffResult, cfg, resources, deployment)

	// Step 10: Return error if differences found and ExitNonzero is set
	if opts.ExitNonzero && diffResult.HasChanges {
		return ErrDiffFound
	}

	return nil
}
