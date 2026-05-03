package diff

import (
	"context"
	"errors"
	"fmt"

	"github.com/koh-sh/apcdeploy/internal/aws"
	"github.com/koh-sh/apcdeploy/internal/config"
	"github.com/koh-sh/apcdeploy/internal/reporter"
)

// ErrDiffFound is returned when differences are found and ExitNonzero is true
var ErrDiffFound = errors.New("differences found")

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

// Execute performs the complete diff workflow.
//
// Output shape (docs/design/output.md §7.2):
//   - changed:       ✓ diff (N lines changed) on the Targets row, unified diff
//                    on stdout (no `=== ===` header for N=1, per §7.2 stdout
//                    header rules).
//   - no changes:    ✓ no changes on the Targets row, no stdout payload.
//   - no deployment: ✓ no prior deployment on the Targets row, local data on
//                    stdout (acts as the right-hand side of the would-be diff).
//   - errors:        ✗ failed: <message> on the Targets row.
//
// The in-progress deployment warning still bypasses the Reporter via display
// (CONTRACT EXCEPTION) so scripts under --silent still see the risk note.
func (e *Executor) Execute(ctx context.Context, opts *Options) error {
	cfg, err := config.LoadConfig(opts.ConfigFile)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	awsClient, err := e.clientFactory(ctx, cfg.Region)
	if err != nil {
		return fmt.Errorf("failed to initialize AWS client: %w", err)
	}

	localData, err := config.LoadDataFile(cfg.DataFile)
	if err != nil {
		return fmt.Errorf("failed to load local configuration file: %w", err)
	}

	id := config.Identifier(awsClient.Region, cfg)
	tg := e.reporter.Targets([]string{id})
	defer tg.Close()
	tg.SetPhase(id, "comparing", "")

	resolver := aws.NewResolver(awsClient)
	resources, err := resolver.ResolveAll(ctx, cfg.Application, cfg.ConfigurationProfile, cfg.Environment, cfg.DeploymentStrategy)
	if err != nil {
		tg.Fail(id, err)
		return fmt.Errorf("failed to resolve resources: %w", err)
	}

	deployment, err := aws.GetLatestDeployment(ctx, awsClient, resources.ApplicationID, resources.EnvironmentID, resources.Profile.ID)
	if err != nil {
		tg.Fail(id, err)
		return fmt.Errorf("failed to get latest deployment: %w", err)
	}

	if deployment == nil {
		tg.Done(id, "no prior deployment")
		// The local data is the would-be initial deployment payload — emit it
		// to stdout so consumers can pipe it into apcdeploy run / git apply.
		e.reporter.Data(localData)
		if len(localData) > 0 && localData[len(localData)-1] != '\n' {
			e.reporter.Data([]byte("\n"))
		}
		return nil
	}

	remoteData, err := aws.GetHostedConfigurationVersion(ctx, awsClient, resources.ApplicationID, resources.Profile.ID, deployment.ConfigurationVersion)
	if err != nil {
		tg.Fail(id, err)
		return fmt.Errorf("failed to get deployed configuration: %w", err)
	}

	diffResult, err := calculate(string(remoteData), string(localData), cfg.DataFile, resources.Profile.Type)
	if err != nil {
		tg.Fail(id, err)
		return fmt.Errorf("failed to calculate diff: %w", err)
	}

	display(e.reporter, tg, id, diffResult, deployment)

	if opts.ExitNonzero && diffResult.HasChanges {
		return ErrDiffFound
	}

	return nil
}
