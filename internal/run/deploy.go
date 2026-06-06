package run

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/koh-sh/apcdeploy/internal/aws"
	"github.com/koh-sh/apcdeploy/internal/config"
)

// Deployer handles deployment operations
type Deployer struct {
	cfg       *config.Config
	awsClient *aws.Client
}

// New creates a new Deployer instance
func New(ctx context.Context, cfg *config.Config) (*Deployer, error) {
	// Initialize AWS client
	awsClient, err := aws.NewClient(ctx, cfg.Region)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize AWS client: %w", err)
	}

	return NewWithClient(cfg, awsClient), nil
}

// NewWithClient creates a new Deployer instance with a provided AWS client
// This is useful for testing with mock clients
func NewWithClient(cfg *config.Config, awsClient *aws.Client) *Deployer {
	return &Deployer{
		cfg:       cfg,
		awsClient: awsClient,
	}
}

// ValidateLocalData validates the configuration data locally
func (d *Deployer) ValidateLocalData(data []byte, contentType string) error {
	return config.ValidateData(data, contentType)
}

// ResolveResources resolves all resource names to IDs
func (d *Deployer) ResolveResources(ctx context.Context) (*aws.ResolvedResources, error) {
	// Create a resolver
	resolver := aws.NewResolver(d.awsClient)

	// Resolve all resources
	resolved, err := resolver.ResolveAll(ctx,
		d.cfg.Application,
		d.cfg.ConfigurationProfile,
		d.cfg.Environment,
		d.cfg.DeploymentStrategy,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve resources: %w", err)
	}

	return resolved, nil
}

// CheckOngoingDeployment checks if there is an ongoing deployment
func (d *Deployer) CheckOngoingDeployment(ctx context.Context, resolved *aws.ResolvedResources) (bool, any, error) {
	return d.awsClient.CheckOngoingDeployment(ctx, resolved.ApplicationID, resolved.EnvironmentID)
}

// CreateVersion creates a new hosted configuration version. The description
// (when non-empty) is forwarded to AppConfig and shown in the console / on
// `apcdeploy status`.
func (d *Deployer) CreateVersion(ctx context.Context, resolved *aws.ResolvedResources, content []byte, contentType, description string) (int32, error) {
	return d.awsClient.CreateHostedConfigurationVersion(ctx, resolved.ApplicationID, resolved.Profile.ID, content, contentType, description)
}

// StartDeployment starts a deployment. The description (when non-empty) is
// forwarded to AppConfig and shown in the console / on `apcdeploy status`.
func (d *Deployer) StartDeployment(ctx context.Context, resolved *aws.ResolvedResources, versionNumber int32, description string) (int32, error) {
	return d.awsClient.StartDeployment(ctx, resolved.ApplicationID, resolved.EnvironmentID, resolved.Profile.ID, resolved.DeploymentStrategyID, versionNumber, description)
}

// HasConfigurationChanges checks if the local configuration differs from the deployed version
func (d *Deployer) HasConfigurationChanges(ctx context.Context, resolved *aws.ResolvedResources, localContent []byte, fileName, contentType string) (bool, error) {
	// Get the latest deployment to find the deployed version number
	deployment, err := aws.GetLatestDeployment(ctx, d.awsClient, resolved.ApplicationID, resolved.EnvironmentID, resolved.Profile.ID)
	if err != nil {
		return false, fmt.Errorf("failed to get latest deployment: %w", err)
	}

	// If no deployment exists, this is the first deployment - has changes
	if deployment == nil {
		return true, nil
	}

	// Get the deployed configuration version content
	remoteContent, err := aws.GetHostedConfigurationVersion(ctx, d.awsClient, resolved.ApplicationID, resolved.Profile.ID, deployment.ConfigurationVersion)
	if err != nil {
		return false, fmt.Errorf("failed to get deployed configuration: %w", err)
	}

	return config.HasContentChanged(remoteContent, localContent, filepath.Ext(fileName), resolved.Profile.Type)
}
