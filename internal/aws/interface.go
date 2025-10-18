package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/account"
	"github.com/aws/aws-sdk-go-v2/service/appconfig"
	"github.com/aws/aws-sdk-go-v2/service/appconfig/types"
	"github.com/aws/aws-sdk-go-v2/service/appconfigdata"
)

// AppConfigSDKAPI defines the minimal AWS SDK interface needed for Client's internal operations.
// This interface is implemented by:
//   - *appconfig.Client (the real AWS SDK client)
//   - mock.MockAppConfigClient (for testing Client's internal methods)
//
// Note: This interface only includes methods used internally by Client. Create and Start
// methods are not included because they're exposed through convenience wrapper methods on
// *Client (see deployment.go) that have different, simpler signatures.
type AppConfigSDKAPI interface {
	// Raw SDK List methods - return paginated results (used by ListAll* pagination methods)
	ListApplications(ctx context.Context, params *appconfig.ListApplicationsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListApplicationsOutput, error)
	ListConfigurationProfiles(ctx context.Context, params *appconfig.ListConfigurationProfilesInput, optFns ...func(*appconfig.Options)) (*appconfig.ListConfigurationProfilesOutput, error)
	ListEnvironments(ctx context.Context, params *appconfig.ListEnvironmentsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListEnvironmentsOutput, error)
	ListDeploymentStrategies(ctx context.Context, params *appconfig.ListDeploymentStrategiesInput, optFns ...func(*appconfig.Options)) (*appconfig.ListDeploymentStrategiesOutput, error)
	ListHostedConfigurationVersions(ctx context.Context, params *appconfig.ListHostedConfigurationVersionsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListHostedConfigurationVersionsOutput, error)
	ListDeployments(ctx context.Context, params *appconfig.ListDeploymentsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListDeploymentsOutput, error)

	// Get methods (used by Resolver, ConfigVersionFetcher, and deployment helper functions)
	GetConfigurationProfile(ctx context.Context, params *appconfig.GetConfigurationProfileInput, optFns ...func(*appconfig.Options)) (*appconfig.GetConfigurationProfileOutput, error)
	GetHostedConfigurationVersion(ctx context.Context, params *appconfig.GetHostedConfigurationVersionInput, optFns ...func(*appconfig.Options)) (*appconfig.GetHostedConfigurationVersionOutput, error)
	GetDeployment(ctx context.Context, params *appconfig.GetDeploymentInput, optFns ...func(*appconfig.Options)) (*appconfig.GetDeploymentOutput, error)

	// Create/Start methods (used by convenience wrappers in deployment.go)
	CreateHostedConfigurationVersion(ctx context.Context, params *appconfig.CreateHostedConfigurationVersionInput, optFns ...func(*appconfig.Options)) (*appconfig.CreateHostedConfigurationVersionOutput, error)
	StartDeployment(ctx context.Context, params *appconfig.StartDeploymentInput, optFns ...func(*appconfig.Options)) (*appconfig.StartDeploymentOutput, error)
}

// AppConfigAPI defines the interface for external code that needs AppConfig operations.
// This interface includes:
//   - Pagination-aware ListAll* methods (automatically handle pagination)
//   - Raw SDK Get methods (for retrieving individual resources)
//
// Notably, this does NOT include Create/Start methods because those are exposed through
// convenience wrapper methods on *Client (see deployment.go) with simplified signatures.
// External code should use *Client directly when deploying configurations.
type AppConfigAPI interface {
	// Pagination-aware List methods - automatically handle pagination to retrieve all resources
	ListAllApplications(ctx context.Context) ([]types.Application, error)
	ListAllConfigurationProfiles(ctx context.Context, appID string) ([]types.ConfigurationProfileSummary, error)
	ListAllEnvironments(ctx context.Context, appID string) ([]types.Environment, error)
	ListAllDeploymentStrategies(ctx context.Context) ([]types.DeploymentStrategy, error)
	ListAllDeployments(ctx context.Context, appID, envID string) ([]types.DeploymentSummary, error)
	ListAllHostedConfigurationVersions(ctx context.Context, appID, profileID string) ([]types.HostedConfigurationVersionSummary, error)

	// Raw SDK Get methods - for retrieving individual resource details
	GetConfigurationProfile(ctx context.Context, params *appconfig.GetConfigurationProfileInput, optFns ...func(*appconfig.Options)) (*appconfig.GetConfigurationProfileOutput, error)
	GetHostedConfigurationVersion(ctx context.Context, params *appconfig.GetHostedConfigurationVersionInput, optFns ...func(*appconfig.Options)) (*appconfig.GetHostedConfigurationVersionOutput, error)
	GetDeployment(ctx context.Context, params *appconfig.GetDeploymentInput, optFns ...func(*appconfig.Options)) (*appconfig.GetDeploymentOutput, error)
}

// AppConfigDataAPI defines the interface for AppConfigData operations.
// This interface is used to fetch configuration data from an application perspective.
type AppConfigDataAPI interface {
	StartConfigurationSession(ctx context.Context, params *appconfigdata.StartConfigurationSessionInput, optFns ...func(*appconfigdata.Options)) (*appconfigdata.StartConfigurationSessionOutput, error)
	GetLatestConfiguration(ctx context.Context, params *appconfigdata.GetLatestConfigurationInput, optFns ...func(*appconfigdata.Options)) (*appconfigdata.GetLatestConfigurationOutput, error)
}

// AccountAPI defines the interface for AWS Account operations
type AccountAPI interface {
	ListRegions(ctx context.Context, params *account.ListRegionsInput, optFns ...func(*account.Options)) (*account.ListRegionsOutput, error)
}
