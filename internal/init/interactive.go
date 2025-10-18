package init

import (
	"context"
	"errors"
	"fmt"
	"sort"

	awsInternal "github.com/koh-sh/apcdeploy/internal/aws"
	"github.com/koh-sh/apcdeploy/internal/prompt"
	"github.com/koh-sh/apcdeploy/internal/reporter"
)

// InteractiveSelector handles interactive resource selection
type InteractiveSelector struct {
	prompter prompt.Prompter
	reporter reporter.ProgressReporter
}

// NewInteractiveSelector creates a new InteractiveSelector
func NewInteractiveSelector(p prompt.Prompter, r reporter.ProgressReporter) *InteractiveSelector {
	return &InteractiveSelector{
		prompter: p,
		reporter: r,
	}
}

// handlePromptError wraps prompt errors with user-friendly messages
func handlePromptError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, prompt.ErrUserCancelled) {
		return err
	}
	return err
}

// promptAndReport handles the common pattern of prompting user and reporting success
func (s *InteractiveSelector) promptAndReport(promptMsg string, options []string, successTemplate string) (string, error) {
	selected, err := s.prompter.Select(promptMsg, options)
	if err != nil {
		return "", handlePromptError(err)
	}

	s.reporter.Success(fmt.Sprintf(successTemplate, selected))
	return selected, nil
}

// SelectRegion prompts user to select a region or returns provided region
func (s *InteractiveSelector) SelectRegion(ctx context.Context, accountClient awsInternal.AccountAPI, providedRegion string) (string, error) {
	// Skip prompt if region is provided
	if providedRegion != "" {
		return providedRegion, nil
	}

	s.reporter.Progress("Fetching available regions...")

	// List enabled regions
	regions, err := awsInternal.ListEnabledRegions(ctx, accountClient)
	if err != nil {
		return "", fmt.Errorf("failed to list regions: %w", err)
	}

	// Check for empty list
	if len(regions) == 0 {
		return "", errors.New("no enabled regions found in your AWS account")
	}

	// Prompt user to select
	return s.promptAndReport(
		"Select AWS region:",
		regions,
		"Selected region: %s",
	)
}

// SelectApplication prompts user to select an application or returns provided app
func (s *InteractiveSelector) SelectApplication(ctx context.Context, client *awsInternal.Client, providedApp string) (string, error) {
	// Skip prompt if app is provided
	if providedApp != "" {
		return providedApp, nil
	}

	s.reporter.Progress("Fetching applications...")

	// List applications
	applications, err := client.ListAllApplications(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to list applications: %w", err)
	}

	// Extract and sort names
	apps := make([]string, 0, len(applications))
	for _, item := range applications {
		if item.Name != nil {
			apps = append(apps, *item.Name)
		}
	}

	if len(apps) == 0 {
		return "", errors.New("no applications found. Please create an application in AppConfig first")
	}

	sort.Strings(apps)

	// Prompt user to select
	return s.promptAndReport(
		"Select application:",
		apps,
		"Selected application: %s",
	)
}

// SelectConfigurationProfile prompts user to select a profile or returns provided profile
func (s *InteractiveSelector) SelectConfigurationProfile(ctx context.Context, client *awsInternal.Client, appID string, providedProfile string) (string, error) {
	// Skip prompt if profile is provided
	if providedProfile != "" {
		return providedProfile, nil
	}

	s.reporter.Progress("Fetching configuration profiles...")

	// List profiles
	configProfiles, err := client.ListAllConfigurationProfiles(ctx, appID)
	if err != nil {
		return "", fmt.Errorf("failed to list configuration profiles: %w", err)
	}

	// Extract and sort names
	profiles := make([]string, 0, len(configProfiles))
	for _, item := range configProfiles {
		if item.Name != nil {
			profiles = append(profiles, *item.Name)
		}
	}

	if len(profiles) == 0 {
		return "", errors.New("no configuration profiles found. Please create a configuration profile in AppConfig first")
	}

	sort.Strings(profiles)

	// Prompt user to select
	return s.promptAndReport(
		"Select configuration profile:",
		profiles,
		"Selected configuration profile: %s",
	)
}

// SelectEnvironment prompts user to select an environment or returns provided env
func (s *InteractiveSelector) SelectEnvironment(ctx context.Context, client *awsInternal.Client, appID string, providedEnv string) (string, error) {
	// Skip prompt if env is provided
	if providedEnv != "" {
		return providedEnv, nil
	}

	s.reporter.Progress("Fetching environments...")

	// List environments
	environments, err := client.ListAllEnvironments(ctx, appID)
	if err != nil {
		return "", fmt.Errorf("failed to list environments: %w", err)
	}

	// Extract and sort names
	envs := make([]string, 0, len(environments))
	for _, item := range environments {
		if item.Name != nil {
			envs = append(envs, *item.Name)
		}
	}

	if len(envs) == 0 {
		return "", errors.New("no environments found. Please create an environment in AppConfig first")
	}

	sort.Strings(envs)

	// Prompt user to select
	return s.promptAndReport(
		"Select environment:",
		envs,
		"Selected environment: %s",
	)
}
