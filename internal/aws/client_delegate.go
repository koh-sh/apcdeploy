package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/appconfig"
)

// This file contains delegate methods that forward calls from the AppConfigAPI interface
// to the underlying AWS SDK AppConfig client. These methods enable Client to satisfy the
// AppConfigAPI interface while maintaining a concrete *appconfig.Client field.
//
// For pagination-aware List methods, see client_list_paginated.go instead.

// Raw SDK List methods - these return paginated results and should generally not be used directly.
// Prefer the ListAll* methods in client_list_paginated.go which handle pagination automatically.

func (c *Client) ListApplications(ctx context.Context, params *appconfig.ListApplicationsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListApplicationsOutput, error) {
	return c.appConfig.ListApplications(ctx, params, optFns...)
}

func (c *Client) ListConfigurationProfiles(ctx context.Context, params *appconfig.ListConfigurationProfilesInput, optFns ...func(*appconfig.Options)) (*appconfig.ListConfigurationProfilesOutput, error) {
	return c.appConfig.ListConfigurationProfiles(ctx, params, optFns...)
}

func (c *Client) ListEnvironments(ctx context.Context, params *appconfig.ListEnvironmentsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListEnvironmentsOutput, error) {
	return c.appConfig.ListEnvironments(ctx, params, optFns...)
}

func (c *Client) ListDeploymentStrategies(ctx context.Context, params *appconfig.ListDeploymentStrategiesInput, optFns ...func(*appconfig.Options)) (*appconfig.ListDeploymentStrategiesOutput, error) {
	return c.appConfig.ListDeploymentStrategies(ctx, params, optFns...)
}

func (c *Client) ListHostedConfigurationVersions(ctx context.Context, params *appconfig.ListHostedConfigurationVersionsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListHostedConfigurationVersionsOutput, error) {
	return c.appConfig.ListHostedConfigurationVersions(ctx, params, optFns...)
}

func (c *Client) ListDeployments(ctx context.Context, params *appconfig.ListDeploymentsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListDeploymentsOutput, error) {
	return c.appConfig.ListDeployments(ctx, params, optFns...)
}

// Get methods

func (c *Client) GetConfigurationProfile(ctx context.Context, params *appconfig.GetConfigurationProfileInput, optFns ...func(*appconfig.Options)) (*appconfig.GetConfigurationProfileOutput, error) {
	return c.appConfig.GetConfigurationProfile(ctx, params, optFns...)
}

func (c *Client) GetHostedConfigurationVersion(ctx context.Context, params *appconfig.GetHostedConfigurationVersionInput, optFns ...func(*appconfig.Options)) (*appconfig.GetHostedConfigurationVersionOutput, error) {
	return c.appConfig.GetHostedConfigurationVersion(ctx, params, optFns...)
}

func (c *Client) GetDeployment(ctx context.Context, params *appconfig.GetDeploymentInput, optFns ...func(*appconfig.Options)) (*appconfig.GetDeploymentOutput, error) {
	return c.appConfig.GetDeployment(ctx, params, optFns...)
}

// Note: CreateHostedConfigurationVersion and StartDeployment are not delegated here
// because they have convenience wrapper methods in deployment.go with different signatures.
// Those convenience wrappers call c.appConfig methods directly.
