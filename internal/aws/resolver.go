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
	client *Client
}

// NewResolver creates a new resolver with the given client
func NewResolver(client *Client) *Resolver {
	return &Resolver{
		client: client,
	}
}

// ResolveApplication resolves an application name to its ID
func (r *Resolver) ResolveApplication(ctx context.Context, appName string) (string, error) {
	allItems, err := r.client.ListAllApplications(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to list applications: %w", err)
	}

	return resolveByName(
		allItems,
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
	allItems, err := r.client.ListAllConfigurationProfiles(ctx, appID)
	if err != nil {
		return nil, fmt.Errorf("failed to list configuration profiles: %w", err)
	}

	var matches []string
	for _, profile := range allItems {
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
	profileOutput, err := r.client.AppConfig.GetConfigurationProfile(ctx, &appconfig.GetConfigurationProfileInput{
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
	allItems, err := r.client.ListAllEnvironments(ctx, appID)
	if err != nil {
		return "", fmt.Errorf("failed to list environments: %w", err)
	}

	return resolveByName(
		allItems,
		envName,
		"environment",
		func(env types.Environment) *string { return env.Name },
		func(env types.Environment) *string { return env.Id },
	)
}

// ResolveDeploymentStrategy resolves a deployment strategy name to its ID
func (r *Resolver) ResolveDeploymentStrategy(ctx context.Context, strategyName string) (string, error) {
	allItems, err := r.client.ListAllDeploymentStrategies(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to list deployment strategies: %w", err)
	}

	return resolveByName(
		allItems,
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

	allItems, err := r.client.ListAllDeploymentStrategies(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to list deployment strategies: %w", err)
	}

	// Find the strategy with matching ID
	for _, strategy := range allItems {
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

// ResolveAll resolves all AWS AppConfig resources (application, profile, environment, strategy).
// If strategyName is empty, deployment strategy resolution is skipped (DeploymentStrategyID will be empty).
// This is useful for commands like 'get' and 'init' that don't require a deployment strategy.
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

	// Resolve deployment strategy (independent) - optional
	var strategyID string
	if strategyName != "" {
		strategyID, err = r.ResolveDeploymentStrategy(ctx, strategyName)
		if err != nil {
			return nil, err
		}
	}

	return &ResolvedResources{
		ApplicationID:        appID,
		Profile:              profile,
		EnvironmentID:        envID,
		DeploymentStrategyID: strategyID,
	}, nil
}
