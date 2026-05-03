package run

import (
	"context"
	"fmt"
	"time"

	"github.com/koh-sh/apcdeploy/internal/aws"
	"github.com/koh-sh/apcdeploy/internal/cli"
	"github.com/koh-sh/apcdeploy/internal/config"
	"github.com/koh-sh/apcdeploy/internal/reporter"
)

// Executor handles the deployment orchestration
type Executor struct {
	reporter        reporter.Reporter
	deployerFactory func(context.Context, *config.Config) (*Deployer, error)
}

// NewExecutor creates a new deployment executor
func NewExecutor(rep reporter.Reporter) *Executor {
	return &Executor{
		reporter:        rep,
		deployerFactory: New,
	}
}

// NewExecutorWithFactory creates a new deployment executor with a custom deployer factory
// This is useful for testing with mock deployers
func NewExecutorWithFactory(rep reporter.Reporter, factory func(context.Context, *config.Config) (*Deployer, error)) *Executor {
	return &Executor{
		reporter:        rep,
		deployerFactory: factory,
	}
}

// Execute performs the complete deployment workflow.
//
// Output shape (docs/design/output.md §7.1):
//   - wait none:    ✓ started — v<N>, <Strategy>
//   - wait-deploy:  ✓ deployed (<elapsed>) — v<N>, <Strategy>, baking started
//   - wait-bake:    ✓ complete  (<elapsed>) — v<N>, <Strategy>
//   - no changes:   ⊘ skipped (no changes)
//   - errors:       ✗ failed: <message>
//
// Sub-phases (output.md §3.2):
//
//	preparing → comparing → creating-version → deploying → baking
//
// The deploying sub-phase drives Targets.SetProgress with AppConfig's
// PercentageComplete so the caller sees a real rollout bar; the baking
// sub-phase uses Targets.SetPhase("baking", detail) instead because there
// is no quantified progress to report (it's a monitoring wait).
func (e *Executor) Execute(ctx context.Context, opts *Options) error {
	if opts.Timeout < 0 {
		return fmt.Errorf("timeout must be a non-negative value")
	}
	if opts.WaitDeploy && opts.WaitBake {
		return fmt.Errorf("--wait-deploy and --wait-bake cannot be used together")
	}

	cfg, dataContent, err := loadConfiguration(opts.ConfigFile)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	deployer, err := e.deployerFactory(ctx, cfg)
	if err != nil {
		return fmt.Errorf("failed to create deployer: %w", err)
	}

	id := config.Identifier(deployer.awsClient.Region, cfg)
	tg := e.reporter.Targets([]string{id})
	defer tg.Close()
	tg.SetPhase(id, "preparing", "")

	resolved, err := deployer.ResolveResources(ctx)
	if err != nil {
		tg.Fail(id, err)
		return fmt.Errorf("failed to resolve resources: %w", err)
	}

	hasOngoing, _, err := deployer.CheckOngoingDeployment(ctx, resolved)
	if err != nil {
		tg.Fail(id, err)
		return fmt.Errorf("failed to check ongoing deployments: %w", err)
	}
	if hasOngoing {
		ongoingErr := fmt.Errorf("deployment already in progress")
		tg.Fail(id, ongoingErr)
		return ongoingErr
	}

	contentType, err := deployer.DetermineContentType(resolved.Profile.Type, cfg.DataFile)
	if err != nil {
		tg.Fail(id, err)
		return fmt.Errorf("failed to determine content type: %w", err)
	}

	if err := deployer.ValidateLocalData(dataContent, contentType); err != nil {
		tg.Fail(id, err)
		return fmt.Errorf("validation failed: %w", err)
	}

	if !opts.Force {
		tg.SetPhase(id, "comparing", "")
		hasChanges, err := deployer.HasConfigurationChanges(ctx, resolved, dataContent, cfg.DataFile, contentType)
		if err != nil {
			tg.Fail(id, err)
			return fmt.Errorf("failed to check for changes: %w", err)
		}
		if !hasChanges {
			tg.Skip(id, "skipped (no changes)")
			return nil
		}
	}

	tg.SetPhase(id, "creating-version", "")
	versionNumber, err := deployer.CreateVersion(ctx, resolved, dataContent, contentType, opts.Description)
	if err != nil {
		tg.Fail(id, err)
		if aws.IsValidationError(err) {
			return fmt.Errorf("%s", aws.FormatValidationError(err))
		}
		return fmt.Errorf("failed to create configuration version: %w", err)
	}

	deployStart := time.Now()
	tg.SetPhase(id, "deploying", "")
	deploymentNumber, err := deployer.StartDeployment(ctx, resolved, versionNumber, opts.Description)
	if err != nil {
		tg.Fail(id, err)
		return fmt.Errorf("failed to start deployment: %w", err)
	}

	strategyName := cfg.DeploymentStrategy
	switch {
	case opts.WaitDeploy:
		if err := deployer.WaitForDeploymentPhase(ctx, resolved, deploymentNumber, false, opts.Timeout, MakeTargetsDeployTick(tg, id)); err != nil {
			tg.Fail(id, err)
			return fmt.Errorf("deployment failed: %w", err)
		}
		tg.Done(id, cli.FormatDeploymentSummary("deployed", deployStart, versionNumber, strategyName, "baking started"))

	case opts.WaitBake:
		// waitCtx caps total wait at opts.Timeout. The per-phase timeout passed
		// below is the remaining budget against that deadline so the inner
		// Wait* timeout reflects "how long this phase may still take".
		deadline := time.Now().Add(time.Duration(opts.Timeout) * time.Second)
		waitCtx, cancel := context.WithDeadline(ctx, deadline)
		defer cancel()

		if err := deployer.WaitForDeploymentPhase(waitCtx, resolved, deploymentNumber, false, remainingSeconds(deadline), MakeTargetsDeployTick(tg, id)); err != nil {
			tg.Fail(id, err)
			return fmt.Errorf("deployment failed: %w", err)
		}
		tg.SetPhase(id, "baking", "")
		if err := deployer.WaitForBakingComplete(waitCtx, resolved, deploymentNumber, remainingSeconds(deadline), MakeTargetsBakeTick(tg, id)); err != nil {
			tg.Fail(id, err)
			return fmt.Errorf("deployment failed: %w", err)
		}
		tg.Done(id, cli.FormatDeploymentSummary("complete", deployStart, versionNumber, strategyName, ""))

	default:
		tg.Done(id, cli.FormatDeploymentSummary("started", deployStart, versionNumber, strategyName, fmt.Sprintf("deployment #%d", deploymentNumber)))
	}

	return nil
}

// remainingSeconds returns the seconds remaining until deadline, clamped at
// 1 to avoid passing 0/negative values to wait functions that interpret 0
// as "no timeout". The actual wait is bounded by the shared waitCtx
// deadline regardless, so the floor only matters when this helper is
// called after the budget is already exhausted.
func remainingSeconds(deadline time.Time) int {
	remaining := int(time.Until(deadline).Seconds())
	if remaining < 1 {
		return 1
	}
	return remaining
}
