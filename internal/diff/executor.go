package diff

import (
	"context"
	"errors"
	"fmt"

	"github.com/koh-sh/apcdeploy/internal/aws"
	"github.com/koh-sh/apcdeploy/internal/batch"
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

// Execute performs the diff workflow for a single config file. Stdout
// emission (the unified diff body) happens inline so the single-target
// output matches the existing single-config behaviour: no
// `=== <id> ===` header (output.md §7.2 stdout header rules).
//
// The multi-config path uses RunOnTarget instead and is responsible for
// buffering and reordering the diff bodies; see cmd/diff.go.
func (e *Executor) Execute(ctx context.Context, opts *Options) error {
	cfg, err := config.LoadConfig(opts.ConfigFile)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

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

	payload, hasChanges, isPriorMissing, err := e.runOnTargetWithClient(ctx, target, tr, awsClient)
	if err != nil {
		return err
	}
	// Preserve the existing single-target stdout split: when there is no
	// prior deployment, the local data is raw user content (use Data) so
	// it can be piped into apcdeploy run / git apply unchanged. Real
	// diff bodies go through Diff so the colorizer can apply +/-
	// styling. The multi-target path always uses Diff (one stream).
	if len(payload) > 0 {
		if isPriorMissing {
			e.reporter.Data(payload)
		} else {
			e.reporter.Diff(payload)
		}
	}
	if opts.ExitNonzero && hasChanges {
		return ErrDiffFound
	}
	return nil
}

// RunOnTarget runs diff for one pre-loaded target. It returns the
// stdout payload (nil when there are no changes), a hasChanges flag
// (used by --exit-nonzero), and the error.
//
// The orchestrator path in cmd/diff.go collects payloads per target
// and flushes them to Reporter.Diff in argument order with a
// `=== <id> ===` header (output.md §7.2). The single-target Execute
// emits payload directly without a header.
func (e *Executor) RunOnTarget(ctx context.Context, t *batch.Target, tr reporter.TargetReporter) ([]byte, bool, error) {
	awsClient, err := e.clientFactory(ctx, t.Config.Region)
	if err != nil {
		tr.Fail(err)
		return nil, false, fmt.Errorf("failed to initialize AWS client: %w", err)
	}
	payload, hasChanges, _, err := e.runOnTargetWithClient(ctx, t, tr, awsClient)
	return payload, hasChanges, err
}

// runOnTargetWithClient is the per-target body shared by Execute and
// RunOnTarget. The in-progress deployment warning bypasses the Reporter
// (CONTRACT EXCEPTION in display.go); the diff payload is returned
// rather than emitted so the caller can order multi-target output.
//
// The third return is "isPriorMissing" — true when the row reported
// "no prior deployment" so Execute can route the payload through Data
// instead of Diff (raw local content is not a unified diff).
func (e *Executor) runOnTargetWithClient(ctx context.Context, t *batch.Target, tr reporter.TargetReporter, awsClient *aws.Client) ([]byte, bool, bool, error) {
	cfg := t.Config

	tr.SetPhase("comparing", "")

	resolver := aws.NewResolver(awsClient)
	resources, err := resolver.ResolveAll(ctx, cfg.Application, cfg.ConfigurationProfile, cfg.Environment, cfg.DeploymentStrategy)
	if err != nil {
		tr.Fail(err)
		return nil, false, false, fmt.Errorf("failed to resolve resources: %w", err)
	}

	deployment, err := aws.GetLatestDeployment(ctx, awsClient, resources.ApplicationID, resources.EnvironmentID, resources.Profile.ID)
	if err != nil {
		tr.Fail(err)
		return nil, false, false, fmt.Errorf("failed to get latest deployment: %w", err)
	}

	localData, err := config.LoadDataFile(cfg.DataFile)
	if err != nil {
		tr.Fail(err)
		return nil, false, false, fmt.Errorf("failed to load local configuration file: %w", err)
	}

	if deployment == nil {
		tr.Done("no prior deployment")
		// Local data is the would-be initial deployment payload. Hand
		// it back as the stdout payload — Execute will route it
		// through Data so it stays raw / pipeable.
		return ensureTrailingNewlineBytes(localData), true, true, nil
	}

	remoteData, err := aws.GetHostedConfigurationVersion(ctx, awsClient, resources.ApplicationID, resources.Profile.ID, deployment.ConfigurationVersion)
	if err != nil {
		tr.Fail(err)
		return nil, false, false, fmt.Errorf("failed to get deployed configuration: %w", err)
	}

	diffResult, err := calculate(string(remoteData), string(localData), cfg.DataFile, resources.Profile.Type)
	if err != nil {
		tr.Fail(err)
		return nil, false, false, fmt.Errorf("failed to calculate diff: %w", err)
	}

	if !diffResult.HasChanges {
		tr.Done("no changes")
		displayDeploymentWarning(deployment)
		return nil, false, false, nil
	}

	added, removed := countChanges(diffResult.UnifiedDiff)
	tr.Done(formatDiffSummary(added, removed))
	displayDeploymentWarning(deployment)
	return []byte(ensureTrailingNewline(diffResult.UnifiedDiff)), true, false, nil
}

// ensureTrailingNewlineBytes is the []byte version of ensureTrailingNewline,
// used for the "no prior deployment" payload (which is raw user data, not a
// unified diff string).
func ensureTrailingNewlineBytes(b []byte) []byte {
	if len(b) == 0 || b[len(b)-1] == '\n' {
		return b
	}
	out := make([]byte, len(b)+1)
	copy(out, b)
	out[len(b)] = '\n'
	return out
}
