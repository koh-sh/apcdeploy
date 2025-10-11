package init

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/aws/aws-sdk-go-v2/service/appconfig"
	"github.com/charmbracelet/huh"
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
	selected, err := s.prompter.Select("Select AWS region:", regions)
	if err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return "", errors.New("operation cancelled")
		}
		return "", err
	}

	s.reporter.Success(fmt.Sprintf("Selected region: %s", selected))
	return selected, nil
}

// SelectApplication prompts user to select an application or returns provided app
func (s *InteractiveSelector) SelectApplication(ctx context.Context, client *awsInternal.Client, providedApp string) (string, error) {
	// Skip prompt if app is provided
	if providedApp != "" {
		return providedApp, nil
	}

	s.reporter.Progress("Fetching applications...")

	// List applications
	output, err := client.AppConfig.ListApplications(ctx, &appconfig.ListApplicationsInput{})
	if err != nil {
		return "", fmt.Errorf("failed to list applications: %w", err)
	}

	// Extract and sort names
	apps := make([]string, 0, len(output.Items))
	for _, item := range output.Items {
		if item.Name != nil {
			apps = append(apps, *item.Name)
		}
	}

	if len(apps) == 0 {
		return "", errors.New("no applications found. Please create an application in AWS AppConfig first")
	}

	sort.Strings(apps)

	// Prompt user to select
	selected, err := s.prompter.Select("Select application:", apps)
	if err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return "", errors.New("operation cancelled")
		}
		return "", err
	}

	s.reporter.Success(fmt.Sprintf("Selected application: %s", selected))
	return selected, nil
}

// SelectConfigurationProfile prompts user to select a profile or returns provided profile
func (s *InteractiveSelector) SelectConfigurationProfile(ctx context.Context, client *awsInternal.Client, appID string, providedProfile string) (string, error) {
	// Skip prompt if profile is provided
	if providedProfile != "" {
		return providedProfile, nil
	}

	s.reporter.Progress("Fetching configuration profiles...")

	// List profiles
	output, err := client.AppConfig.ListConfigurationProfiles(ctx, &appconfig.ListConfigurationProfilesInput{
		ApplicationId: &appID,
	})
	if err != nil {
		return "", fmt.Errorf("failed to list configuration profiles: %w", err)
	}

	// Extract and sort names
	profiles := make([]string, 0, len(output.Items))
	for _, item := range output.Items {
		if item.Name != nil {
			profiles = append(profiles, *item.Name)
		}
	}

	if len(profiles) == 0 {
		return "", errors.New("no configuration profiles found. Please create a configuration profile first")
	}

	sort.Strings(profiles)

	// Prompt user to select
	selected, err := s.prompter.Select("Select configuration profile:", profiles)
	if err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return "", errors.New("operation cancelled")
		}
		return "", err
	}

	s.reporter.Success(fmt.Sprintf("Selected configuration profile: %s", selected))
	return selected, nil
}

// SelectEnvironment prompts user to select an environment or returns provided env
func (s *InteractiveSelector) SelectEnvironment(ctx context.Context, client *awsInternal.Client, appID string, providedEnv string) (string, error) {
	// Skip prompt if env is provided
	if providedEnv != "" {
		return providedEnv, nil
	}

	s.reporter.Progress("Fetching environments...")

	// List environments
	output, err := client.AppConfig.ListEnvironments(ctx, &appconfig.ListEnvironmentsInput{
		ApplicationId: &appID,
	})
	if err != nil {
		return "", fmt.Errorf("failed to list environments: %w", err)
	}

	// Extract and sort names
	envs := make([]string, 0, len(output.Items))
	for _, item := range output.Items {
		if item.Name != nil {
			envs = append(envs, *item.Name)
		}
	}

	if len(envs) == 0 {
		return "", errors.New("no environments found. Please create an environment first")
	}

	sort.Strings(envs)

	// Prompt user to select
	selected, err := s.prompter.Select("Select environment:", envs)
	if err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return "", errors.New("operation cancelled")
		}
		return "", err
	}

	s.reporter.Success(fmt.Sprintf("Selected environment: %s", selected))
	return selected, nil
}
