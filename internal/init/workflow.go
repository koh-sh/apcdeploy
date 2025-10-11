package init

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/account"
	awsInternal "github.com/koh-sh/apcdeploy/internal/aws"
	"github.com/koh-sh/apcdeploy/internal/prompt"
	"github.com/koh-sh/apcdeploy/internal/reporter"
)

// InitWorkflow handles the complete initialization workflow including interactive selection
type InitWorkflow struct {
	awsClient   *awsInternal.Client
	reporter    reporter.ProgressReporter
	prompter    prompt.Prompter
	selector    *InteractiveSelector
	initializer *Initializer
}

// NewInitWorkflow creates a new InitWorkflow
func NewInitWorkflow(ctx context.Context, opts *Options, prompter prompt.Prompter, reporter reporter.ProgressReporter) (*InitWorkflow, error) {
	// Step 1: Region selection (needed before creating AWS client)
	selectedRegion, err := selectOrUseRegion(ctx, opts.Region, prompter, reporter)
	if err != nil {
		return nil, err
	}

	// Step 2: Create AWS AppConfig client with selected/provided region
	awsClient, err := awsInternal.NewClient(ctx, selectedRegion)
	if err != nil {
		return nil, err
	}

	return NewInitWorkflowWithClient(awsClient, prompter, reporter), nil
}

// selectOrUseRegion returns the provided region or prompts user to select one
func selectOrUseRegion(ctx context.Context, providedRegion string, prompter prompt.Prompter, reporter reporter.ProgressReporter) (string, error) {
	// Return provided region if available
	if providedRegion != "" {
		return providedRegion, nil
	}

	// Create Account client for region listing
	accountCfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to load AWS config: %w", err)
	}
	accountClient := account.NewFromConfig(accountCfg)

	// Create interactive selector and prompt for region
	selector := NewInteractiveSelector(prompter, reporter)
	return selector.SelectRegion(ctx, accountClient, providedRegion)
}

// NewInitWorkflowWithClient creates a new InitWorkflow with a provided AWS client
// This is useful for testing with mock clients
func NewInitWorkflowWithClient(awsClient *awsInternal.Client, prompter prompt.Prompter, reporter reporter.ProgressReporter) *InitWorkflow {
	return &InitWorkflow{
		awsClient:   awsClient,
		reporter:    reporter,
		prompter:    prompter,
		selector:    NewInteractiveSelector(prompter, reporter),
		initializer: New(awsClient, reporter),
	}
}

// Run executes the initialization workflow
func (w *InitWorkflow) Run(ctx context.Context, opts *Options) error {
	// Step 3: Application selection
	selectedApp := opts.Application
	if selectedApp == "" {
		var err error
		selectedApp, err = w.selector.SelectApplication(ctx, w.awsClient, opts.Application)
		if err != nil {
			return err
		}
	}

	// Step 4: Resolve application name to ID (needed for profile/env listing)
	resolver := awsInternal.NewResolver(w.awsClient)
	appID, err := resolver.ResolveApplication(ctx, selectedApp)
	if err != nil {
		return fmt.Errorf("failed to resolve application: %w", err)
	}

	// Step 5: Profile selection
	selectedProfile := opts.Profile
	if selectedProfile == "" {
		selectedProfile, err = w.selector.SelectConfigurationProfile(ctx, w.awsClient, appID, opts.Profile)
		if err != nil {
			return err
		}
	}

	// Step 6: Environment selection
	selectedEnv := opts.Environment
	if selectedEnv == "" {
		selectedEnv, err = w.selector.SelectEnvironment(ctx, w.awsClient, appID, opts.Environment)
		if err != nil {
			return err
		}
	}

	// Step 7: Create options with selected/provided values
	finalOpts := &Options{
		Application: selectedApp,
		Profile:     selectedProfile,
		Environment: selectedEnv,
		Region:      opts.Region,
		ConfigFile:  opts.ConfigFile,
		OutputData:  opts.OutputData,
		Force:       opts.Force,
		Silent:      opts.Silent,
	}

	// Step 8: Run existing initialization logic
	_, err = w.initializer.Run(ctx, finalOpts)
	return err
}
