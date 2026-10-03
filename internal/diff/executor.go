package diff

import (
	"context"
	"fmt"

	"github.com/koh-sh/apcdeploy/internal/aws"
	"github.com/koh-sh/apcdeploy/internal/batch"
	"github.com/koh-sh/apcdeploy/internal/config"
	"github.com/koh-sh/apcdeploy/internal/reporter"
)

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

// TargetResult is the per-target outcome of RunOnTarget.
type TargetResult struct {
	// Payload is the stdout diff body (nil when there are no changes).
	Payload []byte
	// HasChanges reports whether local and deployed content differ
	// (used by --exit-nonzero).
	HasChanges bool
	// Warning is the in-progress deployment notice ("" when the latest
	// deployment is not DEPLOYING / BAKING). RunOnTarget never writes it
	// itself; the caller emits it via WriteDeploymentWarnings after the
	// Targets block is closed.
	Warning string
}

// RunOnTarget runs diff for one pre-loaded target and returns its
// TargetResult alongside the error.
//
// The orchestrator path in cmd/diff.go collects results per target
// and, after the orchestrator returns, flushes payloads to
// Reporter.Diff in argument order with a `=== <id> ===` header and
// then writes the collected warnings.
//
// When there is no prior deployment, the payload is the local data
// formatted as an "all lines added" unified diff (every line prefixed
// with `+`) — semantically the would-be initial deployment is "all
// new content".
func (e *Executor) RunOnTarget(ctx context.Context, t *batch.Target, tr reporter.TargetReporter) (TargetResult, error) {
	awsClient, err := e.clientFactory(ctx, t.Config.Region)
	if err != nil {
		tr.Fail(err)
		return TargetResult{}, fmt.Errorf("failed to initialize AWS client: %w", err)
	}

	cfg := t.Config

	tr.SetPhase("comparing", "")

	resolver := aws.NewResolver(awsClient)
	resources, err := resolver.ResolveAll(ctx, cfg.Application, cfg.ConfigurationProfile, cfg.Environment, cfg.DeploymentStrategy)
	if err != nil {
		tr.Fail(err)
		return TargetResult{}, fmt.Errorf("failed to resolve resources: %w", err)
	}

	deployment, err := aws.GetLatestDeployment(ctx, awsClient, resources.ApplicationID, resources.EnvironmentID, resources.Profile.ID)
	if err != nil {
		tr.Fail(err)
		return TargetResult{}, fmt.Errorf("failed to get latest deployment: %w", err)
	}

	localData, err := config.LoadDataFile(cfg.DataFile)
	if err != nil {
		tr.Fail(err)
		return TargetResult{}, fmt.Errorf("failed to load local configuration file: %w", err)
	}

	// remoteData is empty when there is no prior deployment. calculate
	// handles that case as "all added" without a separate code path.
	var remoteData []byte
	if deployment != nil {
		remoteData, err = aws.GetHostedConfigurationVersion(ctx, awsClient, resources.ApplicationID, resources.Profile.ID, deployment.ConfigurationVersion)
		if err != nil {
			tr.Fail(err)
			return TargetResult{}, fmt.Errorf("failed to get deployed configuration: %w", err)
		}
	}

	diffResult, err := calculate(string(remoteData), string(localData), cfg.DataFile, resources.Profile.Type)
	if err != nil {
		tr.Fail(err)
		return TargetResult{}, fmt.Errorf("failed to calculate diff: %w", err)
	}

	if deployment == nil {
		tr.Done("no prior deployment")
		return TargetResult{Payload: []byte(ensureTrailingNewline(diffResult.UnifiedDiff)), HasChanges: true}, nil
	}
	warning := deploymentWarning(deployment)
	if !diffResult.HasChanges {
		tr.Done("no changes")
		return TargetResult{Warning: warning}, nil
	}
	added, removed := countChanges(diffResult.UnifiedDiff)
	tr.Done(formatDiffSummary(added, removed))
	return TargetResult{
		Payload:    []byte(ensureTrailingNewline(diffResult.UnifiedDiff)),
		HasChanges: true,
		Warning:    warning,
	}, nil
}
