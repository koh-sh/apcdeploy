package pull

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/koh-sh/apcdeploy/internal/aws"
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

// Execute performs the complete pull workflow.
//
// Output shape (docs/design/output.md §7.3):
//   - updated:        ✓ updated <data-file-path>
//   - no changes:     ✓ no changes
//   - no deployment:  ✗ failed: no deployment found  (returns aws.ErrNoDeployment)
//   - resolve/fetch/write errors: ✗ failed: <message> (returns wrapped error)
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
	tg := e.reporter.Targets([]string{id})
	defer tg.Close()
	tg.SetPhase(id, "fetching", "")

	resolver := aws.NewResolver(awsClient)
	resources, err := resolver.ResolveAll(ctx, cfg.Application, cfg.ConfigurationProfile, cfg.Environment, "")
	if err != nil {
		tg.Fail(id, err)
		return fmt.Errorf("failed to resolve resources: %w", err)
	}

	deployedConfig, err := aws.GetLatestDeployedConfiguration(ctx, awsClient, resources.ApplicationID, resources.EnvironmentID, resources.Profile.ID)
	if err != nil {
		tg.Fail(id, err)
		return fmt.Errorf("failed to get latest deployed configuration: %w", err)
	}
	if deployedConfig == nil {
		tg.Fail(id, aws.ErrNoDeployment)
		return fmt.Errorf("%w: run 'apcdeploy run' to create the first deployment", aws.ErrNoDeployment)
	}

	dataFilePath := cfg.DataFile
	if !filepath.IsAbs(dataFilePath) {
		dataFilePath = filepath.Join(filepath.Dir(opts.ConfigFile), cfg.DataFile)
	}

	// Compare against the existing local file (if any) so a no-op pull skips
	// the write — pull is idempotent and should not touch mtimes when nothing
	// changed. A read error is treated as "file missing" and falls through to
	// the write path.
	if localData, readErr := config.LoadDataFile(dataFilePath); readErr == nil {
		ext := filepath.Ext(dataFilePath)
		hasChanges, err := config.HasContentChanged(localData, deployedConfig.Content, ext, resources.Profile.Type)
		if err != nil {
			tg.Fail(id, err)
			return fmt.Errorf("failed to check for changes: %w", err)
		}
		if !hasChanges {
			tg.Done(id, "no changes")
			return nil
		}
	}

	if err := config.WriteDataFile(deployedConfig.Content, deployedConfig.ContentType, dataFilePath, resources.Profile.Type, true); err != nil {
		tg.Fail(id, err)
		return fmt.Errorf("failed to write data file: %w", err)
	}
	tg.Done(id, "updated "+dataFilePath)
	return nil
}
