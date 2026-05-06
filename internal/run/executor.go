package run

import (
	"context"
	"fmt"
	"time"

	"github.com/koh-sh/apcdeploy/internal/aws"
	"github.com/koh-sh/apcdeploy/internal/batch"
	"github.com/koh-sh/apcdeploy/internal/cli"
	"github.com/koh-sh/apcdeploy/internal/config"
	"github.com/koh-sh/apcdeploy/internal/deploywait"
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

// RunOnTarget performs the deployment workflow for one pre-loaded target.
// It is invoked from cmd/run.go via batch.Orchestrator (single-target runs
// flow through the orchestrator with a single goroutine).
//
// Options invariants that don't depend on per-target state (Timeout >= 0,
// not both WaitDeploy and WaitBake) are validated once in cmd/run.go
// before the orchestrator starts; this function assumes valid opts.
//
// Output shape:
//   - wait none:    ✓ started — v<N>, <Strategy>
//   - wait-deploy:  ✓ deployed (<elapsed>) — v<N>, <Strategy>, baking started
//   - wait-bake:    ✓ complete  (<elapsed>) — v<N>, <Strategy>
//   - no changes:   ⊘ skipped (no changes)
//   - errors:       ✗ failed: <message>
//
// Sub-phases:
//
//	preparing → comparing → creating-version → deploying → baking
func (e *Executor) RunOnTarget(ctx context.Context, t *batch.Target, tr reporter.TargetReporter, opts *Options) error {
	cfg := t.Config

	dataContent, err := config.LoadDataFile(cfg.DataFile)
	if err != nil {
		tr.Fail(err)
		return fmt.Errorf("failed to read data file %s: %w", cfg.DataFile, err)
	}

	deployer, err := e.deployerFactory(ctx, cfg)
	if err != nil {
		tr.Fail(err)
		return fmt.Errorf("failed to create deployer: %w", err)
	}

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
		timeout := time.Duration(opts.Timeout) * time.Second
		if err := deployer.awsClient.WaitForDeploymentPhase(ctx, resolved.ApplicationID, resolved.EnvironmentID, deploymentNumber, false, timeout, deploywait.MakeTargetsDeployTick(tr)); err != nil {
			tr.Fail(err)
			return fmt.Errorf("deployment failed: %w", err)
		}
		tr.Done(cli.FormatDeploymentSummary("deployed", deploywait.AWSElapsedForDeploy(ctx, deployer.awsClient, resolved.ApplicationID, resolved.EnvironmentID, deploymentNumber, deployStart), versionNumber, strategyName, "baking started"))

	case opts.WaitBake:
		// waitCtx caps total wait at opts.Timeout. The per-phase timeout
		// passed below is the remaining budget against that deadline so the
		// inner Wait* timeout reflects "how long this phase may still take".
		deadline := time.Now().Add(time.Duration(opts.Timeout) * time.Second)
		waitCtx, cancel := context.WithDeadline(ctx, deadline)
		defer cancel()

		if err := deployer.awsClient.WaitForDeploymentPhase(waitCtx, resolved.ApplicationID, resolved.EnvironmentID, deploymentNumber, false, deploywait.RemainingDuration(deadline), deploywait.MakeTargetsDeployTick(tr)); err != nil {
			tr.Fail(err)
			return fmt.Errorf("deployment failed: %w", err)
		}
		tr.SetPhase("baking", "")
		if err := deployer.awsClient.WaitForBakingComplete(waitCtx, resolved.ApplicationID, resolved.EnvironmentID, deploymentNumber, deploywait.RemainingDuration(deadline), deploywait.MakeTargetsBakeTick(tr)); err != nil {
			tr.Fail(err)
			return fmt.Errorf("deployment failed: %w", err)
		}
		tr.Done(cli.FormatDeploymentSummary("complete", deploywait.AWSElapsedForBake(ctx, deployer.awsClient, resolved.ApplicationID, resolved.EnvironmentID, deploymentNumber, deployStart), versionNumber, strategyName, ""))

	default:
		tr.Done(cli.FormatDeploymentSummary("started", 0, versionNumber, strategyName, fmt.Sprintf("deployment #%d", deploymentNumber)))
	}

	return nil
}
