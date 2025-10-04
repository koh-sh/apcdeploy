package run

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

// loadConfiguration loads the configuration file and data file
func loadConfiguration(configPath string) (*config.Config, []byte, error) {
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
	if len(data) > config.MaxConfigSize {
		return fmt.Errorf("configuration data size %d bytes exceeds maximum allowed size of %d bytes (2MB)", len(data), config.MaxConfigSize)
	}

	// Validate syntax based on content type
	switch contentType {
	case config.ContentTypeJSON:
		var js any
		if err := json.Unmarshal(data, &js); err != nil {
			return fmt.Errorf("invalid JSON syntax: %w", err)
		}
	case config.ContentTypeYAML:
		var ym any
		if err := yaml.Unmarshal(data, &ym); err != nil {
			return fmt.Errorf("invalid YAML syntax: %w", err)
		}
	case config.ContentTypeText:
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
		return config.ContentTypeJSON, nil
	}

	// For Freeform, determine from file extension
	ext := strings.ToLower(filepath.Ext(dataPath))
	switch ext {
	case ".json":
		return config.ContentTypeJSON, nil
	case ".yaml", ".yml":
		return config.ContentTypeYAML, nil
	case ".txt":
		return config.ContentTypeText, nil
	default:
		// Default to text/plain for unknown extensions
		return config.ContentTypeText, nil
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

	// Normalize both contents for comparison (handle JSON/YAML formatting differences)
	ext := filepath.Ext(fileName)
	normalizedRemote, err := normalizeContentForComparison(string(remoteContent), ext, resolved.Profile.Type)
	if err != nil {
		return false, fmt.Errorf("failed to normalize remote content: %w", err)
	}

	normalizedLocal, err := normalizeContentForComparison(string(localContent), ext, resolved.Profile.Type)
	if err != nil {
		return false, fmt.Errorf("failed to normalize local content: %w", err)
	}

	// Compare normalized contents
	return normalizedRemote != normalizedLocal, nil
}

// normalizeContentForComparison normalizes content for comparison
// This reuses the logic from the diff package to handle JSON/YAML formatting
func normalizeContentForComparison(content, ext, profileType string) (string, error) {
	switch ext {
	case ".json":
		return normalizeJSON(content, profileType)
	case ".yaml", ".yml":
		return normalizeYAML(content)
	default:
		// For text files, just ensure consistent line endings
		return normalizeText(content), nil
	}
}

// normalizeJSON normalizes JSON content by parsing and re-formatting with sorted keys
// For FeatureFlags profile type, it removes _updatedAt and _createdAt fields recursively
func normalizeJSON(content string, profileType string) (string, error) {
	var data any
	if err := json.Unmarshal([]byte(content), &data); err != nil {
		return "", fmt.Errorf("invalid JSON: %w", err)
	}

	// For FeatureFlags, remove _updatedAt and _createdAt fields recursively
	if profileType == "AWS.AppConfig.FeatureFlags" {
		data = removeTimestampFieldsRecursive(data)
	}

	// Re-marshal with indentation for consistent formatting
	normalized, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to format JSON: %w", err)
	}

	return string(normalized), nil
}

// removeTimestampFieldsRecursive recursively removes _updatedAt and _createdAt from all maps in the object
func removeTimestampFieldsRecursive(obj any) any {
	switch v := obj.(type) {
	case map[string]any:
		// Remove timestamp fields from this map
		delete(v, "_updatedAt")
		delete(v, "_createdAt")
		// Recursively process all values in the map
		for key, value := range v {
			v[key] = removeTimestampFieldsRecursive(value)
		}
		return v
	case []any:
		// Recursively process all elements in the array
		for i, value := range v {
			v[i] = removeTimestampFieldsRecursive(value)
		}
		return v
	default:
		// Return primitive values as-is
		return v
	}
}

// normalizeYAML normalizes YAML content by parsing and re-formatting
func normalizeYAML(content string) (string, error) {
	var data any
	if err := yaml.Unmarshal([]byte(content), &data); err != nil {
		return "", fmt.Errorf("invalid YAML: %w", err)
	}

	// Re-marshal with consistent formatting
	normalized, err := yaml.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("failed to format YAML: %w", err)
	}

	return string(normalized), nil
}

// normalizeText normalizes text content by ensuring consistent line endings
func normalizeText(content string) string {
	// Convert CRLF to LF
	content = strings.ReplaceAll(content, "\r\n", "\n")
	// Ensure single trailing newline
	content = strings.TrimRight(content, "\n") + "\n"
	return content
}
