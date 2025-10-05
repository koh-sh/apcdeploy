package aws

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/appconfig"
	"github.com/aws/aws-sdk-go-v2/service/appconfig/types"
	"github.com/koh-sh/apcdeploy/internal/config"
)

// Resolver handles AWS resource name to ID resolution
type Resolver struct {
	client AppConfigAPI
}

// NewResolver creates a new resolver with the given client
func NewResolver(client *Client) *Resolver {
	return &Resolver{
		client: client.AppConfig,
	}
}

// ResolveApplication resolves an application name to its ID
func (r *Resolver) ResolveApplication(ctx context.Context, appName string) (string, error) {
	output, err := r.client.ListApplications(ctx, &appconfig.ListApplicationsInput{})
	if err != nil {
		return "", fmt.Errorf("failed to list applications: %w", err)
	}

	return resolveByName(
		output.Items,
		appName,
		"application",
		func(app types.Application) *string { return app.Name },
		func(app types.Application) *string { return app.Id },
	)
}

// ProfileInfo contains Configuration Profile details
type ProfileInfo struct {
	ID   string
	Name string
	Type string
}

// ResolveConfigurationProfile resolves a configuration profile name to its ID and details
func (r *Resolver) ResolveConfigurationProfile(ctx context.Context, appID, profileName string) (*ProfileInfo, error) {
	output, err := r.client.ListConfigurationProfiles(ctx, &appconfig.ListConfigurationProfilesInput{
		ApplicationId: &appID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list configuration profiles: %w", err)
	}

	var matches []string
	for _, profile := range output.Items {
		if profile.Name != nil && *profile.Name == profileName {
			if profile.Id != nil {
				matches = append(matches, *profile.Id)
			}
		}
	}

	if len(matches) == 0 {
		return nil, fmt.Errorf("configuration profile not found: %s", profileName)
	}

	if len(matches) > 1 {
		return nil, fmt.Errorf("multiple configuration profiles found with name: %s", profileName)
	}

	// Get detailed profile information
	profileID := matches[0]
	profileOutput, err := r.client.GetConfigurationProfile(ctx, &appconfig.GetConfigurationProfileInput{
		ApplicationId:          &appID,
		ConfigurationProfileId: &profileID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get configuration profile: %w", err)
	}

	profileInfo := &ProfileInfo{
		ID:   profileID,
		Name: profileName,
	}

	if profileOutput.Type != nil {
		profileInfo.Type = *profileOutput.Type
	}

	return profileInfo, nil
}

// ResolveEnvironment resolves an environment name to its ID
func (r *Resolver) ResolveEnvironment(ctx context.Context, appID, envName string) (string, error) {
	output, err := r.client.ListEnvironments(ctx, &appconfig.ListEnvironmentsInput{
		ApplicationId: &appID,
	})
	if err != nil {
		return "", fmt.Errorf("failed to list environments: %w", err)
	}

	return resolveByName(
		output.Items,
		envName,
		"environment",
		func(env types.Environment) *string { return env.Name },
		func(env types.Environment) *string { return env.Id },
	)
}

// ResolveDeploymentStrategy resolves a deployment strategy name to its ID
func (r *Resolver) ResolveDeploymentStrategy(ctx context.Context, strategyName string) (string, error) {
	output, err := r.client.ListDeploymentStrategies(ctx, &appconfig.ListDeploymentStrategiesInput{})
	if err != nil {
		return "", fmt.Errorf("failed to list deployment strategies: %w", err)
	}

	return resolveByName(
		output.Items,
		strategyName,
		"deployment strategy",
		func(strategy types.DeploymentStrategy) *string { return strategy.Name },
		func(strategy types.DeploymentStrategy) *string { return strategy.Id },
	)
}

// ResolveDeploymentStrategyIDToName resolves a deployment strategy ID to its name
// If the ID starts with "AppConfig.", it's a predefined strategy and returns it as is
// Otherwise, looks up the custom strategy name from the AWS API
func (r *Resolver) ResolveDeploymentStrategyIDToName(ctx context.Context, strategyID string) (string, error) {
	// If it's a predefined strategy (starts with "AppConfig."), return as is
	if strings.HasPrefix(strategyID, config.StrategyPrefixPredefined) {
		return strategyID, nil
	}

	output, err := r.client.ListDeploymentStrategies(ctx, &appconfig.ListDeploymentStrategiesInput{})
	if err != nil {
		return "", fmt.Errorf("failed to list deployment strategies: %w", err)
	}

	// Find the strategy with matching ID
	for _, strategy := range output.Items {
		if strategy.Id != nil && *strategy.Id == strategyID {
			if strategy.Name != nil {
				return *strategy.Name, nil
			}
			// If no name, return the ID
			return strategyID, nil
		}
	}

	// If not found, return the ID as is
	return strategyID, nil
}

// ResolvedResources contains all resolved AWS resource IDs and details
type ResolvedResources struct {
	ApplicationID        string
	Profile              *ProfileInfo
	EnvironmentID        string
	DeploymentStrategyID string
}

// ResolveAll resolves all AWS AppConfig resources (application, profile, environment, strategy) concurrently
func (r *Resolver) ResolveAll(ctx context.Context, appName, profileName, envName, strategyName string) (*ResolvedResources, error) {
	// Resolve application first as other resources depend on it
	appID, err := r.ResolveApplication(ctx, appName)
	if err != nil {
		return nil, err
	}

	// Resolve profile (needs appID)
	profile, err := r.ResolveConfigurationProfile(ctx, appID, profileName)
	if err != nil {
		return nil, err
	}

	// Resolve environment (needs appID)
	envID, err := r.ResolveEnvironment(ctx, appID, envName)
	if err != nil {
		return nil, err
	}

	// Resolve deployment strategy (independent)
	strategyID, err := r.ResolveDeploymentStrategy(ctx, strategyName)
	if err != nil {
		return nil, err
	}

	return &ResolvedResources{
		ApplicationID:        appID,
		Profile:              profile,
		EnvironmentID:        envID,
		DeploymentStrategyID: strategyID,
	}, nil
}
