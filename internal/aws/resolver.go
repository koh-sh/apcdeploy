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
	var allItems []types.Application
	var nextToken *string

	// Loop through all pages
	for {
		output, err := r.client.ListApplications(ctx, &appconfig.ListApplicationsInput{
			NextToken: nextToken,
		})
		if err != nil {
			return "", fmt.Errorf("failed to list applications: %w", err)
		}

		allItems = append(allItems, output.Items...)

		// Check if there are more pages
		if output.NextToken == nil {
			break
		}
		nextToken = output.NextToken
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
	var allItems []types.ConfigurationProfileSummary
	var nextToken *string

	// Loop through all pages
	for {
		output, err := r.client.ListConfigurationProfiles(ctx, &appconfig.ListConfigurationProfilesInput{
			ApplicationId: &appID,
			NextToken:     nextToken,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to list configuration profiles: %w", err)
		}

		allItems = append(allItems, output.Items...)

		// Check if there are more pages
		if output.NextToken == nil {
			break
		}
		nextToken = output.NextToken
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
	var allItems []types.Environment
	var nextToken *string

	// Loop through all pages
	for {
		output, err := r.client.ListEnvironments(ctx, &appconfig.ListEnvironmentsInput{
			ApplicationId: &appID,
			NextToken:     nextToken,
		})
		if err != nil {
			return "", fmt.Errorf("failed to list environments: %w", err)
		}

		allItems = append(allItems, output.Items...)

		// Check if there are more pages
		if output.NextToken == nil {
			break
		}
		nextToken = output.NextToken
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
	var allItems []types.DeploymentStrategy
	var nextToken *string

	// Loop through all pages
	for {
		output, err := r.client.ListDeploymentStrategies(ctx, &appconfig.ListDeploymentStrategiesInput{
			NextToken: nextToken,
		})
		if err != nil {
			return "", fmt.Errorf("failed to list deployment strategies: %w", err)
		}

		allItems = append(allItems, output.Items...)

		// Check if there are more pages
		if output.NextToken == nil {
			break
		}
		nextToken = output.NextToken
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

	var allItems []types.DeploymentStrategy
	var nextToken *string

	// Loop through all pages
	for {
		output, err := r.client.ListDeploymentStrategies(ctx, &appconfig.ListDeploymentStrategiesInput{
			NextToken: nextToken,
		})
		if err != nil {
			return "", fmt.Errorf("failed to list deployment strategies: %w", err)
		}

		allItems = append(allItems, output.Items...)

		// Check if there are more pages
		if output.NextToken == nil {
			break
		}
		nextToken = output.NextToken
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
