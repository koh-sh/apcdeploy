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

// Execute performs the rollback workflow.
//
// Output shape (docs/design/output.md §7.7):
//   - has-ongoing path: optional confirmation block, then a single Targets row
//     transitioning preparing → stopping → ✓ stopped (deployment #N).
//   - no-ongoing path: a single Targets row finalized as ⊘ no ongoing deployment.
//   - error path: Targets row finalized as ✗ failed: <message>; the error is
//     also returned so cmd/root.go sets a non-zero exit code.
func (e *Executor) Execute(ctx context.Context, opts *Options) error {
	cfg, err := config.LoadConfig(opts.ConfigFile)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	awsClient, err := e.clientFactory(ctx, cfg.Region)
	if err != nil {
		return fmt.Errorf("failed to initialize AWS client: %w", err)
	}

	id := config.Identifier(awsClient.Region, cfg)

	resolver := aws.NewResolver(awsClient)
	resources, err := resolver.ResolveAll(ctx, cfg.Application, cfg.ConfigurationProfile, cfg.Environment, cfg.DeploymentStrategy)
	if err != nil {
		return fmt.Errorf("failed to resolve resources: %w", err)
	}

	hasOngoing, deployment, err := awsClient.CheckOngoingDeployment(ctx, resources.ApplicationID, resources.EnvironmentID)
	if err != nil {
		return fmt.Errorf("failed to check ongoing deployment: %w", err)
	}
	if !hasOngoing || deployment == nil {
		tg := e.reporter.Targets([]string{id})
		tg.Skip(id, "no ongoing deployment")
		tg.Close()
		return ErrNoOngoingDeployment
	}

	deploymentNumber := deployment.DeploymentNumber
	details, err := aws.GetDeploymentDetails(ctx, awsClient, resources.ApplicationID, resources.EnvironmentID, deploymentNumber)
	if err != nil {
		return fmt.Errorf("failed to get deployment details: %w", err)
	}

	if !opts.SkipConfirmation {
		if err := e.prompter.CheckTTY(); err != nil {
			return fmt.Errorf("use --yes to skip confirmation: %w", err)
		}

		// Render the deployment context (state, version, strategy, etc.) so
		// the user can decide whether to proceed. Doing this before opening
		// Targets avoids the in-place renderer fighting with the prompt.
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

	tg := e.reporter.Targets([]string{id})
	defer tg.Close()
	tg.SetPhase(id, "stopping", "")
	if err := awsClient.StopDeployment(ctx, resources.ApplicationID, resources.EnvironmentID, deploymentNumber); err != nil {
		tg.Fail(id, err)
		return fmt.Errorf("failed to stop deployment: %w", err)
	}
	tg.Done(id, fmt.Sprintf("stopped (deployment #%d)", deploymentNumber))
	return nil
}
