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

// Execute performs the pull workflow for a single config file. The
// per-target body is shared with the multi-config orchestrator path
// (RunOnTarget) so both routes produce identical Targets output.
//
// Output shape:
//   - updated:        ✓ updated <data-file-path>
//   - no changes:     ✓ no changes
//   - no deployment:  ✗ failed: no deployment found  (returns aws.ErrNoDeployment)
//   - resolve/fetch/write errors: ✗ failed: <message> (returns wrapped error)
func (e *Executor) Execute(ctx context.Context, opts *Options) error {
	cfg, err := config.LoadConfig(opts.ConfigFile)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Resolve the AWS client first so the row identifier reflects the
	// SDK-default region when cfg.Region was omitted (multi-config users
	// are expected to set region in yml).
	awsClient, err := e.clientFactory(ctx, cfg.Region)
	if err != nil {
		return fmt.Errorf("failed to initialize AWS client: %w", err)
	}

	target := &batch.Target{
		Path:       opts.ConfigFile,
		Config:     cfg,
		Identifier: config.Identifier(awsClient.Region, cfg),
	}

	tg := e.reporter.Targets([]string{target.Identifier})
	defer tg.Close()
	tr := batch.NewTargetReporter(tg, target.Identifier)

	return e.runOnTargetWithClient(ctx, target, tr, awsClient)
}

// RunOnTarget runs the pull workflow for one pre-loaded target. Used by
// the multi-config orchestrator (cmd/pull.go wires it as the
// batch.ExecuteFunc); callers MUST drive the returned TargetReporter to
// a terminal state via tr.Done / tr.Skip / tr.Fail (this function does).
func (e *Executor) RunOnTarget(ctx context.Context, t *batch.Target, tr reporter.TargetReporter) error {
	awsClient, err := e.clientFactory(ctx, t.Config.Region)
	if err != nil {
		tr.Fail(err)
		return fmt.Errorf("failed to initialize AWS client: %w", err)
	}
	return e.runOnTargetWithClient(ctx, t, tr, awsClient)
}

// runOnTargetWithClient is the per-target body shared by Execute (single
// config) and RunOnTarget (orchestrator). t.Config.DataFile is already
// resolved to an absolute path by config.LoadConfig, so no relative-path
// fix-up is needed here.
func (e *Executor) runOnTargetWithClient(ctx context.Context, t *batch.Target, tr reporter.TargetReporter, awsClient *aws.Client) error {
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
			tr.Done("no changes")
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
