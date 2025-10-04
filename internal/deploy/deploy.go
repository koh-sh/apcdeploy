package deploy

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/goccy/go-yaml"
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

	return &Deployer{
		cfg:       cfg,
		awsClient: awsClient,
	}, nil
}

// LoadConfiguration loads the configuration file and data file
func LoadConfiguration(configPath string) (*config.Config, []byte, error) {
	// Load the config file
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	// Read data file (path is already resolved by LoadConfig)
	dataContent, err := os.ReadFile(cfg.DataFile)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read data file %s: %w", cfg.DataFile, err)
	}

	return cfg, dataContent, nil
}

// ValidateLocalData validates the configuration data locally
func (d *Deployer) ValidateLocalData(data []byte, contentType string) error {
	// Check size (2MB limit)
	const maxSize = 2 * 1024 * 1024
	if len(data) > maxSize {
		return fmt.Errorf("configuration data size %d bytes exceeds maximum allowed size of %d bytes (2MB)", len(data), maxSize)
	}

	// Validate syntax based on content type
	switch contentType {
	case "application/json":
		var js any
		if err := json.Unmarshal(data, &js); err != nil {
			return fmt.Errorf("invalid JSON syntax: %w", err)
		}
	case "application/x-yaml":
		var ym any
		if err := yaml.Unmarshal(data, &ym); err != nil {
			return fmt.Errorf("invalid YAML syntax: %w", err)
		}
	case "text/plain":
		// Text content doesn't need syntax validation
	default:
		return fmt.Errorf("unsupported content type: %s", contentType)
	}

	return nil
}

// DetermineContentType determines the content type based on profile type and file extension
func (d *Deployer) DetermineContentType(profileType, dataPath string) (string, error) {
	// Feature Flags always use JSON
	if profileType == "AWS.AppConfig.FeatureFlags" {
		return "application/json", nil
	}

	// For Freeform, determine from file extension
	ext := strings.ToLower(filepath.Ext(dataPath))
	switch ext {
	case ".json":
		return "application/json", nil
	case ".yaml", ".yml":
		return "application/x-yaml", nil
	case ".txt":
		return "text/plain", nil
	default:
		// Default to text/plain for unknown extensions
		return "text/plain", nil
	}
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

// CreateVersion creates a new hosted configuration version
func (d *Deployer) CreateVersion(ctx context.Context, resolved *aws.ResolvedResources, content []byte, contentType string) (int32, error) {
	return d.awsClient.CreateHostedConfigurationVersion(ctx, resolved.ApplicationID, resolved.Profile.ID, content, contentType, "")
}

// StartDeployment starts a deployment
func (d *Deployer) StartDeployment(ctx context.Context, resolved *aws.ResolvedResources, versionNumber int32) (int32, error) {
	return d.awsClient.StartDeployment(ctx, resolved.ApplicationID, resolved.EnvironmentID, resolved.Profile.ID, resolved.DeploymentStrategyID, versionNumber, "")
}

// WaitForDeployment waits for a deployment to complete
func (d *Deployer) WaitForDeployment(ctx context.Context, resolved *aws.ResolvedResources, deploymentNumber int32, timeoutSeconds int) error {
	timeout := fmt.Sprintf("%ds", timeoutSeconds)
	duration, err := time.ParseDuration(timeout)
	if err != nil {
		return fmt.Errorf("invalid timeout: %w", err)
	}
	return d.awsClient.WaitForDeployment(ctx, resolved.ApplicationID, resolved.EnvironmentID, deploymentNumber, duration)
}

// IsValidationError checks if the error is a validation error
func (d *Deployer) IsValidationError(err error) bool {
	return aws.IsValidationError(err)
}

// FormatValidationError formats a validation error with detailed information
func (d *Deployer) FormatValidationError(err error) string {
	return aws.FormatValidationError(err)
}
