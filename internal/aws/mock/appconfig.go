package mock

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/appconfig"
	"github.com/aws/aws-sdk-go-v2/service/appconfig/types"
)

// MockAppConfigClient is a mock implementation of aws.AppConfigAPI.
// The interface is defined in the consumer package (internal/aws)
// following Go best practice: "Accept interfaces, return structs".
type MockAppConfigClient struct {
	// Raw SDK List methods
	ListApplicationsFunc                func(ctx context.Context, params *appconfig.ListApplicationsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListApplicationsOutput, error)
	ListConfigurationProfilesFunc       func(ctx context.Context, params *appconfig.ListConfigurationProfilesInput, optFns ...func(*appconfig.Options)) (*appconfig.ListConfigurationProfilesOutput, error)
	ListEnvironmentsFunc                func(ctx context.Context, params *appconfig.ListEnvironmentsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListEnvironmentsOutput, error)
	ListDeploymentStrategiesFunc        func(ctx context.Context, params *appconfig.ListDeploymentStrategiesInput, optFns ...func(*appconfig.Options)) (*appconfig.ListDeploymentStrategiesOutput, error)
	ListHostedConfigurationVersionsFunc func(ctx context.Context, params *appconfig.ListHostedConfigurationVersionsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListHostedConfigurationVersionsOutput, error)
	ListDeploymentsFunc                 func(ctx context.Context, params *appconfig.ListDeploymentsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListDeploymentsOutput, error)

	// Get methods
	GetConfigurationProfileFunc       func(ctx context.Context, params *appconfig.GetConfigurationProfileInput, optFns ...func(*appconfig.Options)) (*appconfig.GetConfigurationProfileOutput, error)
	GetHostedConfigurationVersionFunc func(ctx context.Context, params *appconfig.GetHostedConfigurationVersionInput, optFns ...func(*appconfig.Options)) (*appconfig.GetHostedConfigurationVersionOutput, error)
	GetDeploymentFunc                 func(ctx context.Context, params *appconfig.GetDeploymentInput, optFns ...func(*appconfig.Options)) (*appconfig.GetDeploymentOutput, error)

	// Create methods
	CreateHostedConfigurationVersionFunc func(ctx context.Context, params *appconfig.CreateHostedConfigurationVersionInput, optFns ...func(*appconfig.Options)) (*appconfig.CreateHostedConfigurationVersionOutput, error)

	// Start methods
	StartDeploymentFunc func(ctx context.Context, params *appconfig.StartDeploymentInput, optFns ...func(*appconfig.Options)) (*appconfig.StartDeploymentOutput, error)

	// Pagination-aware List methods
	ListAllApplicationsFunc                func(ctx context.Context) ([]types.Application, error)
	ListAllConfigurationProfilesFunc       func(ctx context.Context, appID string) ([]types.ConfigurationProfileSummary, error)
	ListAllEnvironmentsFunc                func(ctx context.Context, appID string) ([]types.Environment, error)
	ListAllDeploymentStrategiesFunc        func(ctx context.Context) ([]types.DeploymentStrategy, error)
	ListAllDeploymentsFunc                 func(ctx context.Context, appID, envID string) ([]types.DeploymentSummary, error)
	ListAllHostedConfigurationVersionsFunc func(ctx context.Context, appID, profileID string) ([]types.HostedConfigurationVersionSummary, error)
}

// Raw SDK List methods

func (m *MockAppConfigClient) ListApplications(ctx context.Context, params *appconfig.ListApplicationsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListApplicationsOutput, error) {
	return m.ListApplicationsFunc(ctx, params, optFns...)
}

func (m *MockAppConfigClient) ListConfigurationProfiles(ctx context.Context, params *appconfig.ListConfigurationProfilesInput, optFns ...func(*appconfig.Options)) (*appconfig.ListConfigurationProfilesOutput, error) {
	return m.ListConfigurationProfilesFunc(ctx, params, optFns...)
}

func (m *MockAppConfigClient) ListEnvironments(ctx context.Context, params *appconfig.ListEnvironmentsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListEnvironmentsOutput, error) {
	return m.ListEnvironmentsFunc(ctx, params, optFns...)
}

func (m *MockAppConfigClient) ListDeploymentStrategies(ctx context.Context, params *appconfig.ListDeploymentStrategiesInput, optFns ...func(*appconfig.Options)) (*appconfig.ListDeploymentStrategiesOutput, error) {
	return m.ListDeploymentStrategiesFunc(ctx, params, optFns...)
}

func (m *MockAppConfigClient) ListHostedConfigurationVersions(ctx context.Context, params *appconfig.ListHostedConfigurationVersionsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListHostedConfigurationVersionsOutput, error) {
	return m.ListHostedConfigurationVersionsFunc(ctx, params, optFns...)
}

func (m *MockAppConfigClient) ListDeployments(ctx context.Context, params *appconfig.ListDeploymentsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListDeploymentsOutput, error) {
	return m.ListDeploymentsFunc(ctx, params, optFns...)
}

// Get methods

func (m *MockAppConfigClient) GetConfigurationProfile(ctx context.Context, params *appconfig.GetConfigurationProfileInput, optFns ...func(*appconfig.Options)) (*appconfig.GetConfigurationProfileOutput, error) {
	return m.GetConfigurationProfileFunc(ctx, params, optFns...)
}

func (m *MockAppConfigClient) GetHostedConfigurationVersion(ctx context.Context, params *appconfig.GetHostedConfigurationVersionInput, optFns ...func(*appconfig.Options)) (*appconfig.GetHostedConfigurationVersionOutput, error) {
	return m.GetHostedConfigurationVersionFunc(ctx, params, optFns...)
}

func (m *MockAppConfigClient) GetDeployment(ctx context.Context, params *appconfig.GetDeploymentInput, optFns ...func(*appconfig.Options)) (*appconfig.GetDeploymentOutput, error) {
	return m.GetDeploymentFunc(ctx, params, optFns...)
}

// Create methods

func (m *MockAppConfigClient) CreateHostedConfigurationVersion(ctx context.Context, params *appconfig.CreateHostedConfigurationVersionInput, optFns ...func(*appconfig.Options)) (*appconfig.CreateHostedConfigurationVersionOutput, error) {
	return m.CreateHostedConfigurationVersionFunc(ctx, params, optFns...)
}

// Start methods

func (m *MockAppConfigClient) StartDeployment(ctx context.Context, params *appconfig.StartDeploymentInput, optFns ...func(*appconfig.Options)) (*appconfig.StartDeploymentOutput, error) {
	return m.StartDeploymentFunc(ctx, params, optFns...)
}

// Pagination-aware List methods

func (m *MockAppConfigClient) ListAllApplications(ctx context.Context) ([]types.Application, error) {
	return m.ListAllApplicationsFunc(ctx)
}

func (m *MockAppConfigClient) ListAllConfigurationProfiles(ctx context.Context, appID string) ([]types.ConfigurationProfileSummary, error) {
	return m.ListAllConfigurationProfilesFunc(ctx, appID)
}

func (m *MockAppConfigClient) ListAllEnvironments(ctx context.Context, appID string) ([]types.Environment, error) {
	return m.ListAllEnvironmentsFunc(ctx, appID)
}

func (m *MockAppConfigClient) ListAllDeploymentStrategies(ctx context.Context) ([]types.DeploymentStrategy, error) {
	return m.ListAllDeploymentStrategiesFunc(ctx)
}

func (m *MockAppConfigClient) ListAllDeployments(ctx context.Context, appID, envID string) ([]types.DeploymentSummary, error) {
	return m.ListAllDeploymentsFunc(ctx, appID, envID)
}

func (m *MockAppConfigClient) ListAllHostedConfigurationVersions(ctx context.Context, appID, profileID string) ([]types.HostedConfigurationVersionSummary, error) {
	return m.ListAllHostedConfigurationVersionsFunc(ctx, appID, profileID)
}
