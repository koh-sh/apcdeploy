package rollback

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/koh-sh/apcdeploy/internal/aws"
	"github.com/koh-sh/apcdeploy/internal/config"
	"github.com/koh-sh/apcdeploy/internal/display"
	"github.com/koh-sh/apcdeploy/internal/prompt"
	"github.com/koh-sh/apcdeploy/internal/reporter"
)

// ErrUserDeclined is returned when the user declines to proceed with the operation
var ErrUserDeclined = errors.New("operation declined by user")

// ErrNoOngoingDeployment is returned when no ongoing deployment is found
var ErrNoOngoingDeployment = errors.New("no ongoing deployment found")

// Executor handles the rollback operation orchestration
type Executor struct {
	reporter      reporter.ProgressReporter
	prompter      prompt.Prompter
	clientFactory func(context.Context, string) (*aws.Client, error)
}

// NewExecutor creates a new rollback executor
func NewExecutor(rep reporter.ProgressReporter, prom prompt.Prompter) *Executor {
	return &Executor{
		reporter:      rep,
		prompter:      prom,
		clientFactory: aws.NewClient,
	}
}

// NewExecutorWithFactory creates a new rollback executor with a custom client factory
// This is useful for testing with mock clients
func NewExecutorWithFactory(rep reporter.ProgressReporter, prom prompt.Prompter, factory func(context.Context, string) (*aws.Client, error)) *Executor {
	return &Executor{
		reporter:      rep,
		prompter:      prom,
		clientFactory: factory,
	}
}

// Execute performs the rollback workflow
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

	// Step 4: Find ongoing deployment
	e.reporter.Progress("Checking for ongoing deployment...")
	hasOngoing, deployment, err := awsClient.CheckOngoingDeployment(ctx, resources.ApplicationID, resources.EnvironmentID)
	if err != nil {
		return fmt.Errorf("failed to check ongoing deployment: %w", err)
	}

	if !hasOngoing || deployment == nil {
		return ErrNoOngoingDeployment
	}

	deploymentNumber := deployment.DeploymentNumber
	e.reporter.Success(fmt.Sprintf("Found ongoing deployment #%d", deploymentNumber))

	// Step 5: Get deployment details for confirmation
	details, err := aws.GetDeploymentDetails(ctx, awsClient, resources.ApplicationID, resources.EnvironmentID, deploymentNumber)
	if err != nil {
		return fmt.Errorf("failed to get deployment details: %w", err)
	}

	// Step 6: Prompt for confirmation unless skipped
	if !opts.SkipConfirmation {
		// Check TTY availability before interactive prompt
		if err := e.prompter.CheckTTY(); err != nil {
			return fmt.Errorf("use --yes to skip confirmation: %w", err)
		}

		// Show deployment information unless in silent mode
		if !opts.Silent {
			display.ShowDeploymentStatus(details, cfg, resources)
		}

		message := fmt.Sprintf("Stop deployment #%d? This will rollback the deployment. (Y/Yes)", deploymentNumber)
		response, err := e.prompter.Input(message, "")
		if err != nil {
			return fmt.Errorf("failed to get user confirmation: %w", err)
		}

		// Accept Y, y, Yes, yes
		normalized := strings.ToLower(strings.TrimSpace(response))
		if normalized != "y" && normalized != "yes" {
			return ErrUserDeclined
		}
	}

	// Step 7: Stop deployment
	e.reporter.Progress(fmt.Sprintf("Stopping deployment #%d...", deploymentNumber))
	if err := awsClient.StopDeployment(ctx, resources.ApplicationID, resources.EnvironmentID, deploymentNumber); err != nil {
		return fmt.Errorf("failed to stop deployment: %w", err)
	}

	e.reporter.Success(fmt.Sprintf("Deployment #%d stopped successfully", deploymentNumber))

	return nil
}
