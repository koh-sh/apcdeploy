package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/appconfig"
	"github.com/aws/aws-sdk-go-v2/service/appconfig/types"
)

// ListAllApplications retrieves all applications with pagination handling
func (c *Client) ListAllApplications(ctx context.Context) ([]types.Application, error) {
	var allItems []types.Application
	var nextToken *string

	for {
		output, err := c.AppConfig.ListApplications(ctx, &appconfig.ListApplicationsInput{
			NextToken: nextToken,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to list applications: %w", err)
		}

		allItems = append(allItems, output.Items...)

		if output.NextToken == nil {
			break
		}
		nextToken = output.NextToken
	}

	return allItems, nil
}

// ListAllConfigurationProfiles retrieves all configuration profiles for an application with pagination handling
func (c *Client) ListAllConfigurationProfiles(ctx context.Context, appID string) ([]types.ConfigurationProfileSummary, error) {
	var allItems []types.ConfigurationProfileSummary
	var nextToken *string

	for {
		output, err := c.AppConfig.ListConfigurationProfiles(ctx, &appconfig.ListConfigurationProfilesInput{
			ApplicationId: &appID,
			NextToken:     nextToken,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to list configuration profiles: %w", err)
		}

		allItems = append(allItems, output.Items...)

		if output.NextToken == nil {
			break
		}
		nextToken = output.NextToken
	}

	return allItems, nil
}

// ListAllEnvironments retrieves all environments for an application with pagination handling
func (c *Client) ListAllEnvironments(ctx context.Context, appID string) ([]types.Environment, error) {
	var allItems []types.Environment
	var nextToken *string

	for {
		output, err := c.AppConfig.ListEnvironments(ctx, &appconfig.ListEnvironmentsInput{
			ApplicationId: &appID,
			NextToken:     nextToken,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to list environments: %w", err)
		}

		allItems = append(allItems, output.Items...)

		if output.NextToken == nil {
			break
		}
		nextToken = output.NextToken
	}

	return allItems, nil
}

// ListAllDeploymentStrategies retrieves all deployment strategies with pagination handling
func (c *Client) ListAllDeploymentStrategies(ctx context.Context) ([]types.DeploymentStrategy, error) {
	var allItems []types.DeploymentStrategy
	var nextToken *string

	for {
		output, err := c.AppConfig.ListDeploymentStrategies(ctx, &appconfig.ListDeploymentStrategiesInput{
			NextToken: nextToken,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to list deployment strategies: %w", err)
		}

		allItems = append(allItems, output.Items...)

		if output.NextToken == nil {
			break
		}
		nextToken = output.NextToken
	}

	return allItems, nil
}

// ListAllDeployments retrieves all deployments for an application and environment with pagination handling
func (c *Client) ListAllDeployments(ctx context.Context, appID, envID string) ([]types.DeploymentSummary, error) {
	var allItems []types.DeploymentSummary
	var nextToken *string

	for {
		output, err := c.AppConfig.ListDeployments(ctx, &appconfig.ListDeploymentsInput{
			ApplicationId: &appID,
			EnvironmentId: &envID,
			NextToken:     nextToken,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to list deployments: %w", err)
		}

		allItems = append(allItems, output.Items...)

		if output.NextToken == nil {
			break
		}
		nextToken = output.NextToken
	}

	return allItems, nil
}

// ListAllHostedConfigurationVersions retrieves all hosted configuration versions with pagination handling
func (c *Client) ListAllHostedConfigurationVersions(ctx context.Context, appID, profileID string) ([]types.HostedConfigurationVersionSummary, error) {
	var allItems []types.HostedConfigurationVersionSummary
	var nextToken *string

	for {
		output, err := c.AppConfig.ListHostedConfigurationVersions(ctx, &appconfig.ListHostedConfigurationVersionsInput{
			ApplicationId:          &appID,
			ConfigurationProfileId: &profileID,
			NextToken:              nextToken,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to list hosted configuration versions: %w", err)
		}

		allItems = append(allItems, output.Items...)

		if output.NextToken == nil {
			break
		}
		nextToken = output.NextToken
	}

	return allItems, nil
}
