package run

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/koh-sh/apcdeploy/internal/aws"
	"github.com/koh-sh/apcdeploy/internal/batch"
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

// Execute performs the deployment workflow for a single config file. The
// per-target body is shared with the multi-config orchestrator path
// (RunOnTarget) so both routes produce identical Targets output.
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
func (e *Executor) Execute(ctx context.Context, opts *Options) error {
	if err := validateOpts(opts); err != nil {
		return err
	}

	cfg, dataContent, err := loadConfiguration(opts.ConfigFile)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	deployer, err := e.deployerFactory(ctx, cfg)
	if err != nil {
		return fmt.Errorf("failed to create deployer: %w", err)
	}

	target := &batch.Target{
		Path:       opts.ConfigFile,
		Config:     cfg,
		Identifier: config.Identifier(deployer.awsClient.Region, cfg),
	}

	tg := e.reporter.Targets([]string{target.Identifier})
	defer tg.Close()
	tr := batch.NewTargetReporter(tg, target.Identifier)

	return e.runOnTargetWithDeployer(ctx, target, dataContent, tr, deployer, opts)
}

// RunOnTarget runs the deployment for one pre-loaded target. Used by the
// multi-config orchestrator (cmd/run.go wires it via RunOnTargetWithOpts
// closure since ExecuteFunc has no opts parameter).
func (e *Executor) RunOnTarget(ctx context.Context, t *batch.Target, tr reporter.TargetReporter, opts *Options) error {
	if err := validateOpts(opts); err != nil {
		return err
	}

	dataContent, err := os.ReadFile(t.Config.DataFile)
	if err != nil {
		tr.Fail(err)
		return fmt.Errorf("failed to read data file %s: %w", t.Config.DataFile, err)
	}

	deployer, err := e.deployerFactory(ctx, t.Config)
	if err != nil {
		tr.Fail(err)
		return fmt.Errorf("failed to create deployer: %w", err)
	}

	return e.runOnTargetWithDeployer(ctx, t, dataContent, tr, deployer, opts)
}

// runOnTargetWithDeployer is the per-target body shared by Execute and
// RunOnTarget. It assumes the AWS deployer is already constructed and
// the data file content has been read.
func (e *Executor) runOnTargetWithDeployer(ctx context.Context, t *batch.Target, dataContent []byte, tr reporter.TargetReporter, deployer *Deployer, opts *Options) error {
	cfg := t.Config

	tr.SetPhase("preparing", "")

	resolved, err := deployer.ResolveResources(ctx)
	if err != nil {
		tr.Fail(err)
		return fmt.Errorf("failed to resolve resources: %w", err)
	}

	hasOngoing, _, err := deployer.CheckOngoingDeployment(ctx, resolved)
	if err != nil {
		tr.Fail(err)
		return fmt.Errorf("failed to check ongoing deployments: %w", err)
	}
	if hasOngoing {
		ongoingErr := fmt.Errorf("deployment already in progress")
		tr.Fail(ongoingErr)
		return ongoingErr
	}

	contentType, err := deployer.DetermineContentType(resolved.Profile.Type, cfg.DataFile)
	if err != nil {
		tr.Fail(err)
		return fmt.Errorf("failed to determine content type: %w", err)
	}

	if err := deployer.ValidateLocalData(dataContent, contentType); err != nil {
		tr.Fail(err)
		return fmt.Errorf("validation failed: %w", err)
	}

	if !opts.Force {
		tr.SetPhase("comparing", "")
		hasChanges, err := deployer.HasConfigurationChanges(ctx, resolved, dataContent, cfg.DataFile, contentType)
		if err != nil {
			tr.Fail(err)
			return fmt.Errorf("failed to check for changes: %w", err)
		}
		if !hasChanges {
			tr.Skip("skipped (no changes)")
			return nil
		}
	}

	tr.SetPhase("creating-version", "")
	versionNumber, err := deployer.CreateVersion(ctx, resolved, dataContent, contentType, opts.Description)
	if err != nil {
		tr.Fail(err)
		if aws.IsValidationError(err) {
			return fmt.Errorf("%s", aws.FormatValidationError(err))
		}
		return fmt.Errorf("failed to create configuration version: %w", err)
	}

	deployStart := time.Now()
	tr.SetPhase("deploying", "")
	deploymentNumber, err := deployer.StartDeployment(ctx, resolved, versionNumber, opts.Description)
	if err != nil {
		tr.Fail(err)
		return fmt.Errorf("failed to start deployment: %w", err)
	}

	strategyName := cfg.DeploymentStrategy
	switch {
	case opts.WaitDeploy:
		if err := deployer.WaitForDeploymentPhase(ctx, resolved, deploymentNumber, false, opts.Timeout, MakeTargetsDeployTick(tr)); err != nil {
			tr.Fail(err)
			return fmt.Errorf("deployment failed: %w", err)
		}
		tr.Done(cli.FormatDeploymentSummary("deployed", deployStart, versionNumber, strategyName, "baking started"))

	case opts.WaitBake:
		// waitCtx caps total wait at opts.Timeout. The per-phase timeout
		// passed below is the remaining budget against that deadline so the
		// inner Wait* timeout reflects "how long this phase may still take".
		deadline := time.Now().Add(time.Duration(opts.Timeout) * time.Second)
		waitCtx, cancel := context.WithDeadline(ctx, deadline)
		defer cancel()

		if err := deployer.WaitForDeploymentPhase(waitCtx, resolved, deploymentNumber, false, remainingSeconds(deadline), MakeTargetsDeployTick(tr)); err != nil {
			tr.Fail(err)
			return fmt.Errorf("deployment failed: %w", err)
		}
		tr.SetPhase("baking", "")
		if err := deployer.WaitForBakingComplete(waitCtx, resolved, deploymentNumber, remainingSeconds(deadline), MakeTargetsBakeTick(tr)); err != nil {
			tr.Fail(err)
			return fmt.Errorf("deployment failed: %w", err)
		}
		tr.Done(cli.FormatDeploymentSummary("complete", deployStart, versionNumber, strategyName, ""))

	default:
		tr.Done(cli.FormatDeploymentSummary("started", deployStart, versionNumber, strategyName, fmt.Sprintf("deployment #%d", deploymentNumber)))
	}

	return nil
}

// validateOpts checks Options invariants that don't depend on AWS state.
// Extracted so single-target Execute and the multi-config orchestrator
// path apply the same checks before doing any AWS work.
func validateOpts(opts *Options) error {
	if opts.Timeout < 0 {
		return fmt.Errorf("timeout must be a non-negative value")
	}
	if opts.WaitDeploy && opts.WaitBake {
		return fmt.Errorf("--wait-deploy and --wait-bake cannot be used together")
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
