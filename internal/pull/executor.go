package pull

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/koh-sh/apcdeploy/internal/aws"
	"github.com/koh-sh/apcdeploy/internal/batch"
	"github.com/koh-sh/apcdeploy/internal/config"
	"github.com/koh-sh/apcdeploy/internal/reporter"
)

// Executor handles the pull operation orchestration
type Executor struct {
	reporter      reporter.Reporter
	clientFactory func(context.Context, string) (*aws.Client, error)
}

// NewExecutor creates a new pull executor
func NewExecutor(rep reporter.Reporter) *Executor {
	return &Executor{
		reporter:      rep,
		clientFactory: aws.NewClient,
	}
}

// NewExecutorWithFactory creates a new pull executor with a custom client factory
// This is useful for testing with mock clients
func NewExecutorWithFactory(rep reporter.Reporter, factory func(context.Context, string) (*aws.Client, error)) *Executor {
	return &Executor{
		reporter:      rep,
		clientFactory: factory,
	}
}

// RunOnTarget runs the pull workflow for one pre-loaded target. It is
// invoked from cmd/pull.go via batch.Orchestrator (single-target runs
// flow through the orchestrator with a single goroutine). Callers MUST
// drive the supplied TargetReporter to a terminal state via Done /
// Skip / Fail (this function does).
//
// Output shape:
//   - updated:        ✓ updated <data-file-path>
//   - no changes:     ⊘ no changes  (Skip — counted as no-op by the orchestrator)
//   - no deployment:  ✗ failed: no deployment found  (returns aws.ErrNoDeployment)
//   - resolve/fetch/write errors: ✗ failed: <message> (returns wrapped error)
//
// t.Config.DataFile is already resolved to an absolute path by
// config.LoadConfig, so no relative-path fix-up is needed.
func (e *Executor) RunOnTarget(ctx context.Context, t *batch.Target, tr reporter.TargetReporter) error {
	awsClient, err := e.clientFactory(ctx, t.Config.Region)
	if err != nil {
		tr.Fail(err)
		return fmt.Errorf("failed to initialize AWS client: %w", err)
	}

	cfg := t.Config

	tr.SetPhase("fetching", "")

	resolver := aws.NewResolver(awsClient)
	resources, err := resolver.ResolveAll(ctx, cfg.Application, cfg.ConfigurationProfile, cfg.Environment, "")
	if err != nil {
		tr.Fail(err)
		return fmt.Errorf("failed to resolve resources: %w", err)
	}

	deployedConfig, err := aws.GetLatestDeployedConfiguration(ctx, awsClient, resources.ApplicationID, resources.EnvironmentID, resources.Profile.ID)
	if err != nil {
		tr.Fail(err)
		return fmt.Errorf("failed to get latest deployed configuration: %w", err)
	}
	if deployedConfig == nil {
		tr.Fail(aws.ErrNoDeployment)
		return fmt.Errorf("%w: run 'apcdeploy run' to create the first deployment", aws.ErrNoDeployment)
	}

	dataFilePath := cfg.DataFile

	// Compare against the existing local file (if any) so a no-op pull skips
	// the write — pull is idempotent and should not touch mtimes when nothing
	// changed. A read error is treated as "file missing" and falls through to
	// the write path.
	if localData, readErr := config.LoadDataFile(dataFilePath); readErr == nil {
		ext := filepath.Ext(dataFilePath)
		hasChanges, err := config.HasContentChanged(localData, deployedConfig.Content, ext, resources.Profile.Type)
		if err != nil {
			tr.Fail(err)
			return fmt.Errorf("failed to check for changes: %w", err)
		}
		if !hasChanges {
			tr.Skip("no changes")
			return nil
		}
	}

	if err := config.WriteDataFile(deployedConfig.Content, deployedConfig.ContentType, dataFilePath, resources.Profile.Type, true); err != nil {
		tr.Fail(err)
		return fmt.Errorf("failed to write data file: %w", err)
	}
	tr.Done("updated " + dataFilePath)
	return nil
}
