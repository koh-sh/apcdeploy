package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/appconfig"
)

// This file contains delegate methods that forward calls from the AppConfigAPI interface
// to the underlying AWS SDK AppConfig client. These methods enable Client to satisfy the
// AppConfigAPI interface while maintaining a concrete *appconfig.Client field.
//
// For pagination-aware List methods, see client_list_paginated.go instead. Raw SDK List
// methods are not delegated here because external code should always use the ListAll*
// helpers; the SDK List methods are reachable internally through the AppConfigSDKAPI
// interface used by client_list_paginated.go.

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
