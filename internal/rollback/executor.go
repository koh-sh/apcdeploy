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
	reporter      reporter.Reporter
	prompter      prompt.Prompter
	clientFactory func(context.Context, string) (*aws.Client, error)
}

// NewExecutor creates a new rollback executor
func NewExecutor(rep reporter.Reporter, prom prompt.Prompter) *Executor {
	return &Executor{
		reporter:      rep,
		prompter:      prom,
		clientFactory: aws.NewClient,
	}
}

// NewExecutorWithFactory creates a new rollback executor with a custom client factory
// This is useful for testing with mock clients
func NewExecutorWithFactory(rep reporter.Reporter, prom prompt.Prompter, factory func(context.Context, string) (*aws.Client, error)) *Executor {
	return &Executor{
		reporter:      rep,
		prompter:      prom,
		clientFactory: factory,
	}
}

// Execute performs the rollback workflow
func (e *Executor) Execute(ctx context.Context, opts *Options) error {
	cfg, err := config.LoadConfig(opts.ConfigFile)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	awsClient, err := e.clientFactory(ctx, cfg.Region)
	if err != nil {
		return fmt.Errorf("failed to initialize AWS client: %w", err)
	}

	// The discovery phase ("find a deployment to roll back") spans resolve +
	// CheckOngoing + GetDeploymentDetails. They share a single user-facing
	// step; the actual stop happens after the confirmation prompt as a
	// separate spinner so the user sees a clear "we acted" line.
	sp := e.reporter.Spin("Finding ongoing deployment...")
	resolver := aws.NewResolver(awsClient)
	resources, err := resolver.ResolveAll(ctx, cfg.Application, cfg.ConfigurationProfile, cfg.Environment, cfg.DeploymentStrategy)
	if err != nil {
		sp.Stop()
		return fmt.Errorf("failed to resolve resources: %w", err)
	}

	hasOngoing, deployment, err := awsClient.CheckOngoingDeployment(ctx, resources.ApplicationID, resources.EnvironmentID)
	if err != nil {
		sp.Stop()
		return fmt.Errorf("failed to check ongoing deployment: %w", err)
	}
	if !hasOngoing || deployment == nil {
		sp.Stop()
		return ErrNoOngoingDeployment
	}

	deploymentNumber := deployment.DeploymentNumber
	details, err := aws.GetDeploymentDetails(ctx, awsClient, resources.ApplicationID, resources.EnvironmentID, deploymentNumber)
	if err != nil {
		sp.Stop()
		return fmt.Errorf("failed to get deployment details: %w", err)
	}
	sp.Done(fmt.Sprintf("Found ongoing deployment #%d (%s)", deploymentNumber, details.State))

	if !opts.SkipConfirmation {
		if err := e.prompter.CheckTTY(); err != nil {
			return fmt.Errorf("use --yes to skip confirmation: %w", err)
		}

		display.DeploymentStatus(e.reporter, details, cfg, resources)

		message := fmt.Sprintf("Stop deployment #%d? This will rollback the deployment. (Y/Yes)", deploymentNumber)
		response, err := e.prompter.Input(message, "")
		if err != nil {
			return fmt.Errorf("failed to get user confirmation: %w", err)
		}

		normalized := strings.ToLower(strings.TrimSpace(response))
		if normalized != "y" && normalized != "yes" {
			return ErrUserDeclined
		}
	}

	sp = e.reporter.Spin(fmt.Sprintf("Stopping deployment #%d...", deploymentNumber))
	if err := awsClient.StopDeployment(ctx, resources.ApplicationID, resources.EnvironmentID, deploymentNumber); err != nil {
		sp.Stop()
		return fmt.Errorf("failed to stop deployment: %w", err)
	}
	sp.Done(fmt.Sprintf("Stopped deployment #%d", deploymentNumber))

	return nil
}
