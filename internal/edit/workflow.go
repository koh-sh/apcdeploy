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

// resolvedTargets holds the AWS IDs and profile metadata for the edit target.
type resolvedTargets struct {
	AppID   string
	EnvID   string
	Profile *awsInternal.ProfileInfo
}

// Run executes the edit workflow. TTY availability is enforced upstream in
// newWorkflow, which runs before any selector here would block on a prompt.
func (w *workflow) Run(ctx context.Context, opts *Options) error {
	targets, err := w.resolveTargets(ctx, opts)
	if err != nil {
		return err
	}

	deployed, strategyID, err := w.prepareDeployment(ctx, targets, opts)
	if err != nil {
		return err
	}

	return w.editAndDeploy(ctx, targets, deployed, strategyID, opts)
}

// resolveTargets selects the application/profile/environment (interactively if
// the corresponding option is empty) and resolves them to AWS IDs.
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

	w.reporter.Step("Resolving AWS resources...")
	profile, err := resolver.ResolveConfigurationProfile(ctx, appID, selectedProfile)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve configuration profile: %w", err)
	}
	envID, err := resolver.ResolveEnvironment(ctx, appID, selectedEnv)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve environment: %w", err)
	}
	w.reporter.Success(fmt.Sprintf("Resolved resources: App=%s, Profile=%s, Env=%s", appID, profile.ID, envID))

	return &resolvedTargets{AppID: appID, EnvID: envID, Profile: profile}, nil
}

// prepareDeployment fetches the latest deployment, determines the strategy to
// use, and aborts if another deployment is already in progress.
//
// The ongoing-deployment check runs before announcing "Found deployment ..."
// so users hitting that abort path don't see a misleading success message
// immediately followed by an error.
func (w *workflow) prepareDeployment(ctx context.Context, t *resolvedTargets, opts *Options) (*awsInternal.DeployedConfigInfo, string, error) {
	w.reporter.Step("Checking for ongoing deployments...")
	ongoing, _, err := w.awsClient.CheckOngoingDeployment(ctx, t.AppID, t.EnvID)
	if err != nil {
		return nil, "", fmt.Errorf("failed to check ongoing deployments: %w", err)
	}
	if ongoing {
		return nil, "", fmt.Errorf("deployment already in progress")
	}
	w.reporter.Success("No ongoing deployments")

	w.reporter.Step("Fetching latest deployed configuration...")
	deployed, err := awsInternal.GetLatestDeployedConfiguration(ctx, w.awsClient, t.AppID, t.EnvID, t.Profile.ID)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get latest deployed configuration: %w", err)
	}
	if deployed == nil {
		return nil, "", fmt.Errorf("%w: run 'apcdeploy run' to create the first deployment", awsInternal.ErrNoDeployment)
	}
	w.reporter.Success(fmt.Sprintf("Found deployment #%d (version %d)", deployed.DeploymentNumber, deployed.VersionNumber))

	resolver := awsInternal.NewResolver(w.awsClient)
	strategyID, err := resolveStrategyID(ctx, resolver, opts.DeploymentStrategy, deployed.DeploymentStrategyID)
	if err != nil {
		return nil, "", err
	}

	return deployed, strategyID, nil
}

// editAndDeploy launches the editor, validates the result, creates a new
// configuration version when content changed, and starts the deployment.
func (w *workflow) editAndDeploy(ctx context.Context, t *resolvedTargets, deployed *awsInternal.DeployedConfigInfo, strategyID string, opts *Options) error {
	ext := config.ExtensionForContentType(deployed.ContentType)
	w.reporter.Step(fmt.Sprintf("Opening editor (%s)...", editorCommand()))
	_, edited, err := editBuffer(deployed.Content, ext)
	if err != nil {
		return fmt.Errorf("failed to edit configuration: %w", err)
	}

	w.reporter.Step("Validating configuration data...")
	if err := config.ValidateData(edited, deployed.ContentType); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}
	w.reporter.Success("Configuration data validated")

	changed, err := config.HasContentChanged(deployed.Content, edited, ext, t.Profile.Type)
	if err != nil {
		return fmt.Errorf("failed to compare configuration: %w", err)
	}
	if !changed {
		w.reporter.Success("No changes detected - skipping deployment")
		return nil
	}

	w.reporter.Step("Creating configuration version...")
	versionNumber, err := w.awsClient.CreateHostedConfigurationVersion(ctx, t.AppID, t.Profile.ID, edited, deployed.ContentType, "")
	if err != nil {
		if awsInternal.IsValidationError(err) {
			return fmt.Errorf("%s", awsInternal.FormatValidationError(err))
		}
		return fmt.Errorf("failed to create configuration version: %w", err)
	}
	w.reporter.Success(fmt.Sprintf("Created configuration version %d", versionNumber))

	w.reporter.Step("Starting deployment...")
	deploymentNumber, err := w.awsClient.StartDeployment(ctx, t.AppID, t.EnvID, t.Profile.ID, strategyID, versionNumber, "")
	if err != nil {
		return fmt.Errorf("failed to start deployment: %w", err)
	}
	w.reporter.Success(fmt.Sprintf("Deployment #%d started", deploymentNumber))

	return w.waitIfRequested(ctx, t.AppID, t.EnvID, deploymentNumber, opts)
}

// resolveStrategyID returns the deployment strategy ID to use.
// If the user provided one via flag, resolve it; otherwise inherit the strategy
// from the most recent deployment.
func resolveStrategyID(ctx context.Context, resolver *awsInternal.Resolver, providedName, inheritedID string) (string, error) {
	if providedName != "" {
		id, err := resolver.ResolveDeploymentStrategy(ctx, providedName)
		if err != nil {
			return "", fmt.Errorf("failed to resolve deployment strategy: %w", err)
		}
		return id, nil
	}
	if inheritedID == "" {
		return "", fmt.Errorf("could not determine deployment strategy from latest deployment; specify --deployment-strategy")
	}
	return inheritedID, nil
}

func (w *workflow) waitIfRequested(ctx context.Context, appID, envID string, deploymentNumber int32, opts *Options) error {
	timeout := time.Duration(opts.Timeout) * time.Second
	switch {
	case opts.WaitDeploy:
		pb := w.reporter.Progress("Deploying...")
		if err := w.awsClient.WaitForDeploymentPhase(ctx, appID, envID, deploymentNumber, false, timeout, run.MakeDeploymentTick(pb)); err != nil {
			pb.Stop()
			return fmt.Errorf("deployment failed: %w", err)
		}
		pb.Done("Deployment phase completed (now baking)")
	case opts.WaitBake:
		pb := w.reporter.Progress("Deploying...")
		if err := w.awsClient.WaitForDeploymentPhase(ctx, appID, envID, deploymentNumber, true, timeout, run.MakeDeploymentTick(pb)); err != nil {
			pb.Stop()
			return fmt.Errorf("deployment failed: %w", err)
		}
		pb.Done("Deployment completed successfully")
	default:
		w.reporter.Warn(fmt.Sprintf("Deployment #%d is in progress. Use 'apcdeploy status' to check the status.", deploymentNumber))
	}
	return nil
}
