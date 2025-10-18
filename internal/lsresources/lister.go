package lsresources

import (
	"context"
	"fmt"
	"sort"

	awsInternal "github.com/koh-sh/apcdeploy/internal/aws"
)

// Lister handles listing of AppConfig resources
type Lister struct {
	client *awsInternal.Client
	region string
}

// New creates a new Lister
func New(client *awsInternal.Client, region string) *Lister {
	return &Lister{
		client: client,
		region: region,
	}
}

// ListResources fetches all AppConfig resources in a hierarchical structure
func (l *Lister) ListResources(ctx context.Context) (*ResourcesTree, error) {
	tree := &ResourcesTree{
		Region:               l.region,
		Applications:         []Application{},
		DeploymentStrategies: []DeploymentStrategy{},
	}

	// List deployment strategies
	strategies, err := l.listDeploymentStrategies(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list deployment strategies: %w", err)
	}
	tree.DeploymentStrategies = strategies

	// List applications
	applications, err := l.client.ListAllApplications(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list applications: %w", err)
	}

	// Process each application
	for _, appSummary := range applications {
		if appSummary.Id == nil || appSummary.Name == nil {
			continue
		}

		app := Application{
			Name:         *appSummary.Name,
			ID:           *appSummary.Id,
			Profiles:     []ConfigurationProfile{},
			Environments: []Environment{},
		}

		// List configuration profiles for this application
		profiles, err := l.listProfiles(ctx, *appSummary.Id)
		if err != nil {
			return nil, fmt.Errorf("failed to list profiles for application %s: %w", *appSummary.Name, err)
		}
		app.Profiles = profiles

		// List environments for this application
		environments, err := l.listEnvironments(ctx, *appSummary.Id)
		if err != nil {
			return nil, fmt.Errorf("failed to list environments for application %s: %w", *appSummary.Name, err)
		}
		app.Environments = environments

		tree.Applications = append(tree.Applications, app)
	}

	// Sort applications by name for consistent output
	sort.Slice(tree.Applications, func(i, j int) bool {
		return tree.Applications[i].Name < tree.Applications[j].Name
	})

	return tree, nil
}

// listProfiles fetches all configuration profiles for an application
func (l *Lister) listProfiles(ctx context.Context, appID string) ([]ConfigurationProfile, error) {
	items, err := l.client.ListAllConfigurationProfiles(ctx, appID)
	if err != nil {
		return nil, err
	}

	profiles := make([]ConfigurationProfile, 0, len(items))
	for _, item := range items {
		if item.Id != nil && item.Name != nil {
			profiles = append(profiles, ConfigurationProfile{
				Name: *item.Name,
				ID:   *item.Id,
			})
		}
	}

	// Sort profiles by name
	sort.Slice(profiles, func(i, j int) bool {
		return profiles[i].Name < profiles[j].Name
	})

	return profiles, nil
}

// listEnvironments fetches all environments for an application
func (l *Lister) listEnvironments(ctx context.Context, appID string) ([]Environment, error) {
	items, err := l.client.ListAllEnvironments(ctx, appID)
	if err != nil {
		return nil, err
	}

	environments := make([]Environment, 0, len(items))
	for _, item := range items {
		if item.Id != nil && item.Name != nil {
			environments = append(environments, Environment{
				Name: *item.Name,
				ID:   *item.Id,
			})
		}
	}

	// Sort environments by name
	sort.Slice(environments, func(i, j int) bool {
		return environments[i].Name < environments[j].Name
	})

	return environments, nil
}

// listDeploymentStrategies fetches all deployment strategies
func (l *Lister) listDeploymentStrategies(ctx context.Context) ([]DeploymentStrategy, error) {
	items, err := l.client.ListAllDeploymentStrategies(ctx)
	if err != nil {
		return nil, err
	}

	strategies := make([]DeploymentStrategy, 0, len(items))
	for _, item := range items {
		if item.Id != nil && item.Name != nil {
			strategy := DeploymentStrategy{
				Name: *item.Name,
				ID:   *item.Id,
			}
			// Add description if available
			if item.Description != nil {
				strategy.Description = *item.Description
			}
			// Add deployment duration (int32 field, not a pointer)
			strategy.DeploymentDurationInMinutes = item.DeploymentDurationInMinutes
			// Add final bake time (int32 field, not a pointer)
			strategy.FinalBakeTimeInMinutes = item.FinalBakeTimeInMinutes
			// Add growth factor (pointer field)
			if item.GrowthFactor != nil {
				strategy.GrowthFactor = *item.GrowthFactor
			}
			// Add growth type
			if item.GrowthType != "" {
				strategy.GrowthType = string(item.GrowthType)
			}
			// Add replicate to
			if item.ReplicateTo != "" {
				strategy.ReplicateTo = string(item.ReplicateTo)
			}
			strategies = append(strategies, strategy)
		}
	}

	// Sort strategies by name
	sort.Slice(strategies, func(i, j int) bool {
		return strategies[i].Name < strategies[j].Name
	})

	return strategies, nil
}
