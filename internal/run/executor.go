package run

import (
	"context"
	"fmt"
	"time"

	"github.com/koh-sh/apcdeploy/internal/aws"
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

// Execute performs the complete deployment workflow
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

	// Build the checklist plan additively so the conditional "detect changes"
	// phase doesn't require remapping indices afterward. idxChanges stays -1
	// when --force skips the AWS round-trip; the only path that touches
	// idxChanges is gated by !opts.Force, so the sentinel never reaches the
	// checklist.
	var labels []string
	add := func(name string) int {
		labels = append(labels, name)
		return len(labels) - 1
	}

	idxResolve := add("Resolving AWS resources")
	idxOngoing := add("Checking for ongoing deployments")
	idxChanges := -1
	if !opts.Force {
		idxChanges = add("Detecting changes")
	}
	idxVersion := add("Creating configuration version")
	idxDeploy := add("Starting deployment")

	chk := e.reporter.Checklist(labels)
	defer chk.Close()

	chk.Start(idxResolve)
	resolved, err := deployer.ResolveResources(ctx)
	if err != nil {
		chk.Fail(idxResolve, "")
		return fmt.Errorf("failed to resolve resources: %w", err)
	}
	chk.Done(idxResolve, fmt.Sprintf("Resolved resources: App=%s, Profile=%s, Env=%s, Strategy=%s",
		resolved.ApplicationID, resolved.Profile.ID, resolved.EnvironmentID, resolved.DeploymentStrategyID))

	chk.Start(idxOngoing)
	hasOngoingDeployment, _, err := deployer.CheckOngoingDeployment(ctx, resolved)
	if err != nil {
		chk.Fail(idxOngoing, "")
		return fmt.Errorf("failed to check ongoing deployments: %w", err)
	}
	if hasOngoingDeployment {
		chk.Fail(idxOngoing, "")
		return fmt.Errorf("deployment already in progress")
	}
	chk.Done(idxOngoing, "No ongoing deployments")

	contentType, err := deployer.DetermineContentType(resolved.Profile.Type, cfg.DataFile)
	if err != nil {
		return fmt.Errorf("failed to determine content type: %w", err)
	}

	// Local validation runs without a checklist row: it's instant and
	// failures surface as the returned error, which root.go renders.
	if err := deployer.ValidateLocalData(dataContent, contentType); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	if !opts.Force {
		chk.Start(idxChanges)
		hasChanges, err := deployer.HasConfigurationChanges(ctx, resolved, dataContent, cfg.DataFile, contentType)
		if err != nil {
			chk.Fail(idxChanges, "")
			return fmt.Errorf("failed to check for changes: %w", err)
		}
		if !hasChanges {
			chk.Skip(idxChanges, "No changes detected — skipping deployment")
			return nil
		}
		chk.Done(idxChanges, "Detected changes")
	}

	chk.Start(idxVersion)
	versionNumber, err := deployer.CreateVersion(ctx, resolved, dataContent, contentType, opts.Description)
	if err != nil {
		chk.Fail(idxVersion, "")
		if aws.IsValidationError(err) {
			return fmt.Errorf("%s", aws.FormatValidationError(err))
		}
		return fmt.Errorf("failed to create configuration version: %w", err)
	}
	chk.Done(idxVersion, fmt.Sprintf("Created configuration version %d", versionNumber))

	chk.Start(idxDeploy)
	deploymentNumber, err := deployer.StartDeployment(ctx, resolved, versionNumber, opts.Description)
	if err != nil {
		chk.Fail(idxDeploy, "")
		return fmt.Errorf("failed to start deployment: %w", err)
	}
	chk.Done(idxDeploy, fmt.Sprintf("Started deployment #%d", deploymentNumber))

	// Supersede the deferred Close above: the wait phase's progress bar
	// manages its own cursor and would tangle with the checklist's animation
	// goroutine if both ran concurrently. Close is idempotent, so the deferred
	// call later becomes a no-op on the success path. The defer remains useful
	// for the error returns above, where it cleans up before cmd/root.go
	// renders the error line.
	chk.Close()

	switch {
	case opts.WaitDeploy:
		// Wait for deploy phase only (until BAKING starts)
		pb := e.reporter.Progress("Deploying...")
		if err := deployer.WaitForDeploymentPhase(ctx, resolved, deploymentNumber, false, opts.Timeout, MakeDeployTick(pb, "Baking...")); err != nil {
			pb.Stop()
			return fmt.Errorf("deployment failed: %w", err)
		}
		pb.Done("Deployment phase completed (now baking)")

	case opts.WaitBake:
		// Two-phase wait. The deploy phase uses a progress bar (AppConfig
		// reports a real rollout %) and the bake phase uses a spinner with a
		// "(~N min left)" countdown (no quantified progress is being made —
		// it's just a monitoring window).
		//
		// waitCtx caps total wait at opts.Timeout. The per-phase timeout
		// passed below is the remaining budget against that deadline so the
		// inner Wait* timeout reflects "how long this phase may still take",
		// not the original full budget. This keeps the timed-out error
		// message ("bake phase timed out after X") meaningful.
		deadline := time.Now().Add(time.Duration(opts.Timeout) * time.Second)
		waitCtx, cancel := context.WithDeadline(ctx, deadline)
		defer cancel()

		pb := e.reporter.Progress("Deploying...")
		if err := deployer.WaitForDeploymentPhase(waitCtx, resolved, deploymentNumber, false, remainingSeconds(deadline), MakeDeployTick(pb, "Deploying...")); err != nil {
			pb.Stop()
			return fmt.Errorf("deployment failed: %w", err)
		}
		pb.Done("Deployment phase completed (now baking)")

		bakeSpin := e.reporter.Spin("Baking...")
		if err := deployer.WaitForBakingComplete(waitCtx, resolved, deploymentNumber, remainingSeconds(deadline), MakeBakeTick(bakeSpin)); err != nil {
			bakeSpin.Stop()
			return fmt.Errorf("deployment failed: %w", err)
		}
		bakeSpin.Done("Deployment completed successfully")

	default:
		// No wait requested
		e.reporter.Warn(fmt.Sprintf("Deployment #%d is in progress. Use 'apcdeploy status' to check the status.", deploymentNumber))
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
