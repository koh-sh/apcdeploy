package edit

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	awsInternal "github.com/koh-sh/apcdeploy/internal/aws"
	"github.com/koh-sh/apcdeploy/internal/config"
	initPkg "github.com/koh-sh/apcdeploy/internal/init"
	"github.com/koh-sh/apcdeploy/internal/prompt"
	"github.com/koh-sh/apcdeploy/internal/reporter"
	"github.com/koh-sh/apcdeploy/internal/run"
)

// workflow orchestrates the edit command against AWS AppConfig.
type workflow struct {
	awsClient *awsInternal.Client
	reporter  reporter.Reporter
	prompter  prompt.Prompter
	selector  *initPkg.InteractiveSelector
}

// newWorkflow creates a workflow, resolving the AWS region interactively if needed.
func newWorkflow(ctx context.Context, opts *Options, prompter prompt.Prompter, rep reporter.Reporter) (*workflow, error) {
	// A TTY is required in two cases:
	//   - any targeting flag is missing (interactive selection is needed)
	//   - $EDITOR is unset and we'll fall back to vi, which itself needs a TTY
	// When all flags are provided AND the user has set $EDITOR explicitly,
	// we trust them — automation can run a non-interactive editor (e.g. a
	// test fixture) without a controlling terminal.
	needsInteractive := opts.Region == "" || opts.Application == "" || opts.Profile == "" || opts.Environment == ""
	editorIsDefault := strings.TrimSpace(os.Getenv("EDITOR")) == ""
	if needsInteractive || editorIsDefault {
		if err := prompter.CheckTTY(); err != nil {
			return nil, fmt.Errorf("%w: provide --region/--app/--profile/--env and set $EDITOR to run non-interactively", err)
		}
	}

	region, err := initPkg.SelectOrUseRegion(ctx, opts.Region, prompter, rep)
	if err != nil {
		return nil, err
	}

	awsClient, err := awsInternal.NewClient(ctx, region)
	if err != nil {
		return nil, err
	}

	return newWorkflowWithClient(awsClient, prompter, rep), nil
}

// newWorkflowWithClient constructs a workflow with a pre-built AWS client (for tests).
func newWorkflowWithClient(awsClient *awsInternal.Client, prompter prompt.Prompter, rep reporter.Reporter) *workflow {
	return &workflow{
		awsClient: awsClient,
		reporter:  rep,
		prompter:  prompter,
		selector:  initPkg.NewInteractiveSelector(prompter, rep),
	}
}

// resolvedTargets holds the AWS IDs and profile metadata for the edit target,
// plus the human-readable names needed to build the canonical identifier.
type resolvedTargets struct {
	AppName string
	AppID   string
	EnvName string
	EnvID   string
	Profile *awsInternal.ProfileInfo
}

// Identifier returns the canonical region/app/profile/env string used in the
// Targets row. Profile.Name comes from the resolver lookup, not the
// user-supplied flag, so it always matches the AWS-side display name.
func (t *resolvedTargets) Identifier(region string) string {
	return region + "/" + t.AppName + "/" + t.Profile.Name + "/" + t.EnvName
}

// Run executes the edit workflow.
//
// Output shape (docs/design/output.md §7.6):
//   - resolve / fetch / ongoing-check are silent unless they fail
//   - $EDITOR launches without a "launching $EDITOR" spinner (§7.6)
//   - after the editor closes, a single Targets row carries the deployment
//     lifecycle: creating-version → deploying → ✓ deployed/complete (...)
//     or ⊘ skipped (no changes) when the edit was a no-op.
func (w *workflow) Run(ctx context.Context, opts *Options) error {
	targets, err := w.resolveTargets(ctx, opts)
	if err != nil {
		return err
	}

	deployed, strategyID, strategyName, err := w.prepareDeployment(ctx, targets, opts)
	if err != nil {
		return err
	}

	return w.editAndDeploy(ctx, targets, deployed, strategyID, strategyName, opts)
}

// resolveTargets selects the application/profile/environment (interactively if
// the corresponding option is empty) and resolves them to AWS IDs.
//
// No spinner here: List* APIs are fast and the interactive selection itself
// already gives the user feedback that resolution is happening.
func (w *workflow) resolveTargets(ctx context.Context, opts *Options) (*resolvedTargets, error) {
	selectedApp, err := w.selector.SelectApplication(ctx, w.awsClient, opts.Application)
	if err != nil {
		return nil, err
	}

	resolver := awsInternal.NewResolver(w.awsClient)
	appID, err := resolver.ResolveApplication(ctx, selectedApp)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve application: %w", err)
	}

	selectedProfile, err := w.selector.SelectConfigurationProfile(ctx, w.awsClient, appID, opts.Profile)
	if err != nil {
		return nil, err
	}

	selectedEnv, err := w.selector.SelectEnvironment(ctx, w.awsClient, appID, opts.Environment)
	if err != nil {
		return nil, err
	}

	profile, err := resolver.ResolveConfigurationProfile(ctx, appID, selectedProfile)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve configuration profile: %w", err)
	}
	envID, err := resolver.ResolveEnvironment(ctx, appID, selectedEnv)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve environment: %w", err)
	}

	return &resolvedTargets{
		AppName: selectedApp,
		AppID:   appID,
		EnvName: selectedEnv,
		EnvID:   envID,
		Profile: profile,
	}, nil
}

// prepareDeployment fetches the latest deployment, determines the strategy to
// use (resolved to both ID and human-readable name), and aborts if another
// deployment is already in progress.
//
// Errors here pre-empt the editor launch — the user wastes no keystrokes on
// content that can't be deployed anyway.
func (w *workflow) prepareDeployment(ctx context.Context, t *resolvedTargets, opts *Options) (*awsInternal.DeployedConfigInfo, string, string, error) {
	ongoing, _, err := w.awsClient.CheckOngoingDeployment(ctx, t.AppID, t.EnvID)
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to check ongoing deployments: %w", err)
	}
	if ongoing {
		return nil, "", "", fmt.Errorf("deployment already in progress")
	}

	deployed, err := awsInternal.GetLatestDeployedConfiguration(ctx, w.awsClient, t.AppID, t.EnvID, t.Profile.ID)
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to get latest deployed configuration: %w", err)
	}
	if deployed == nil {
		return nil, "", "", fmt.Errorf("%w: run 'apcdeploy run' to create the first deployment", awsInternal.ErrNoDeployment)
	}

	resolver := awsInternal.NewResolver(w.awsClient)
	strategyID, strategyName, err := resolveStrategy(ctx, resolver, opts.DeploymentStrategy, deployed.DeploymentStrategyID)
	if err != nil {
		return nil, "", "", err
	}

	return deployed, strategyID, strategyName, nil
}

// editAndDeploy launches the editor, validates the result, creates a new
// configuration version when content changed, and starts the deployment.
func (w *workflow) editAndDeploy(ctx context.Context, t *resolvedTargets, deployed *awsInternal.DeployedConfigInfo, strategyID, strategyName string, opts *Options) error {
	ext := config.ExtensionForContentType(deployed.ContentType)

	// No "launching $EDITOR" spinner per output.md §7.6 — short-lived
	// spinners on instant operations create flicker, and the editor itself
	// is the user-facing signal that a hand-off is happening.
	_, edited, err := editBuffer(deployed.Content, ext)
	if err != nil {
		return fmt.Errorf("failed to edit configuration: %w", err)
	}

	if err := config.ValidateData(edited, deployed.ContentType); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	id := t.Identifier(w.awsClient.Region)
	tg := w.reporter.Targets([]string{id})
	defer tg.Close()

	changed, err := config.HasContentChanged(deployed.Content, edited, ext, t.Profile.Type)
	if err != nil {
		tg.Fail(id, err)
		return fmt.Errorf("failed to compare configuration: %w", err)
	}
	if !changed {
		tg.Skip(id, "skipped (no changes)")
		return nil
	}

	tg.SetPhase(id, "creating-version", "")
	versionNumber, err := w.awsClient.CreateHostedConfigurationVersion(ctx, t.AppID, t.Profile.ID, edited, deployed.ContentType, opts.Description)
	if err != nil {
		tg.Fail(id, err)
		if awsInternal.IsValidationError(err) {
			return fmt.Errorf("%s", awsInternal.FormatValidationError(err))
		}
		return fmt.Errorf("failed to create configuration version: %w", err)
	}

	deployStart := time.Now()
	tg.SetPhase(id, "deploying", "")
	deploymentNumber, err := w.awsClient.StartDeployment(ctx, t.AppID, t.EnvID, t.Profile.ID, strategyID, versionNumber, opts.Description)
	if err != nil {
		tg.Fail(id, err)
		return fmt.Errorf("failed to start deployment: %w", err)
	}

	return w.waitIfRequested(ctx, tg, id, t, deploymentNumber, versionNumber, strategyName, deployStart, opts)
}

// resolveStrategy returns the deployment strategy ID and a display name.
//
// When the user passed --deployment-strategy, both the resolved ID and the
// supplied name are returned. When the strategy is inherited from the last
// deployment, the inherited ID is returned for both fields — the human name
// would require an extra ListDeploymentStrategies call which:
//   - is unnecessary for execution (only the ID is used),
//   - and would risk a stray AWS round-trip when the user has not actually
//     changed strategies.
//
// The Done summary therefore shows the strategy ID for the inherited case;
// callers that want a human label should pass --deployment-strategy.
func resolveStrategy(ctx context.Context, resolver *awsInternal.Resolver, providedName, inheritedID string) (string, string, error) {
	if providedName != "" {
		id, err := resolver.ResolveDeploymentStrategy(ctx, providedName)
		if err != nil {
			return "", "", fmt.Errorf("failed to resolve deployment strategy: %w", err)
		}
		return id, providedName, nil
	}
	if inheritedID == "" {
		return "", "", fmt.Errorf("could not determine deployment strategy from latest deployment; specify --deployment-strategy")
	}
	return inheritedID, inheritedID, nil
}

// waitIfRequested optionally blocks for the deploy or bake phase to complete,
// driving the same Targets row through the deploying/baking sub-phases that
// run uses. The done summary follows the output.md §3.3.2 format and
// distinguishes the verb by wait mode (output.md §7.1.0).
func (w *workflow) waitIfRequested(ctx context.Context, tg reporter.Targets, id string, t *resolvedTargets, deploymentNumber, versionNumber int32, strategyName string, deployStart time.Time, opts *Options) error {
	timeout := time.Duration(opts.Timeout) * time.Second
	switch {
	case opts.WaitDeploy:
		if err := w.awsClient.WaitForDeploymentPhase(ctx, t.AppID, t.EnvID, deploymentNumber, false, timeout, run.MakeTargetsDeployTick(tg, id)); err != nil {
			tg.Fail(id, err)
			return fmt.Errorf("deployment failed: %w", err)
		}
		tg.Done(id, formatEditSummary("deployed", deployStart, versionNumber, strategyName, "baking started"))
	case opts.WaitBake:
		// waitCtx caps total wait at opts.Timeout. The per-phase timeout
		// passed below is the remaining budget against that deadline so the
		// inner Wait* timeout reflects "how long this phase may still take".
		deadline := time.Now().Add(timeout)
		waitCtx, cancel := context.WithDeadline(ctx, deadline)
		defer cancel()

		if err := w.awsClient.WaitForDeploymentPhase(waitCtx, t.AppID, t.EnvID, deploymentNumber, false, remainingDuration(deadline), run.MakeTargetsDeployTick(tg, id)); err != nil {
			tg.Fail(id, err)
			return fmt.Errorf("deployment failed: %w", err)
		}
		tg.SetPhase(id, "baking", "")
		if err := w.awsClient.WaitForBakingComplete(waitCtx, t.AppID, t.EnvID, deploymentNumber, remainingDuration(deadline), run.MakeTargetsBakeTick(tg, id)); err != nil {
			tg.Fail(id, err)
			return fmt.Errorf("deployment failed: %w", err)
		}
		tg.Done(id, formatEditSummary("complete", deployStart, versionNumber, strategyName, ""))
	default:
		tg.Done(id, formatEditSummary("started", deployStart, versionNumber, strategyName, fmt.Sprintf("deployment #%d", deploymentNumber)))
	}
	return nil
}

// formatEditSummary mirrors run's summary format. Edit reuses the same shape
// because the deploy lifecycle after the editor close is identical to a
// `run` invocation.
func formatEditSummary(verb string, start time.Time, version int32, strategy, addendum string) string {
	out := verb
	if !start.IsZero() && verb != "started" {
		out += " (" + formatElapsed(time.Since(start)) + ")"
	}
	if version > 0 {
		out += fmt.Sprintf(" — v%d", version)
	}
	if strategy != "" {
		if version > 0 {
			out += ", " + strategy
		} else {
			out += " — " + strategy
		}
	}
	if addendum != "" {
		out += ", " + addendum
	}
	return out
}

// formatElapsed renders a duration as compact "Ns" or "Nm Ns".
func formatElapsed(d time.Duration) string {
	d = d.Round(time.Second)
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	m := int(d.Minutes())
	s := int(d.Seconds()) - m*60
	if s == 0 {
		return fmt.Sprintf("%dm", m)
	}
	return fmt.Sprintf("%dm %ds", m, s)
}

// remainingDuration returns the time until deadline, clamped at 1s to
// avoid passing 0/negative values to wait functions that interpret 0 as
// "no timeout". The actual wait is bounded by the shared waitCtx deadline
// regardless, so the floor only matters when this helper is called after
// the budget is already exhausted.
func remainingDuration(deadline time.Time) time.Duration {
	remaining := time.Until(deadline)
	if remaining < time.Second {
		return time.Second
	}
	return remaining
}
