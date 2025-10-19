package pull

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	awsSdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/appconfig"
	"github.com/koh-sh/apcdeploy/internal/aws"
	"github.com/koh-sh/apcdeploy/internal/config"
	"github.com/koh-sh/apcdeploy/internal/reporter"
)

// ErrNoDeployment is returned when no deployment is found for the configuration profile
var ErrNoDeployment = errors.New("no deployment found for this configuration profile")

// Executor handles the pull operation orchestration
type Executor struct {
	reporter      reporter.ProgressReporter
	clientFactory func(context.Context, string) (*aws.Client, error)
}

// NewExecutor creates a new pull executor
func NewExecutor(rep reporter.ProgressReporter) *Executor {
	return &Executor{
		reporter:      rep,
		clientFactory: aws.NewClient,
	}
}

// NewExecutorWithFactory creates a new pull executor with a custom client factory
// This is useful for testing with mock clients
func NewExecutorWithFactory(rep reporter.ProgressReporter, factory func(context.Context, string) (*aws.Client, error)) *Executor {
	return &Executor{
		reporter:      rep,
		clientFactory: factory,
	}
}

// Execute performs the complete pull workflow
func (e *Executor) Execute(ctx context.Context, opts *Options) error {
	// Step 1: Load configuration
	e.reporter.Progress("Loading configuration...")
	cfg, err := config.LoadConfig(opts.ConfigFile)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}
	e.reporter.Success("Configuration loaded")

	// Step 2: Initialize AWS client
	awsClient, err := e.clientFactory(ctx, cfg.Region)
	if err != nil {
		return fmt.Errorf("failed to initialize AWS client: %w", err)
	}

	// Step 3: Resolve resources
	e.reporter.Progress("Resolving resources...")
	resolver := aws.NewResolver(awsClient)
	// Deployment strategy not needed for pull operation
	resources, err := resolver.ResolveAll(ctx, cfg.Application, cfg.ConfigurationProfile, cfg.Environment, "")
	if err != nil {
		return fmt.Errorf("failed to resolve resources: %w", err)
	}
	e.reporter.Success(fmt.Sprintf("Resolved resources: App=%s, Profile=%s, Env=%s",
		resources.ApplicationID,
		resources.Profile.ID,
		resources.EnvironmentID,
	))

	// Step 4: Get latest deployment
	e.reporter.Progress("Fetching latest deployment...")
	deployment, err := aws.GetLatestDeployment(ctx, awsClient, resources.ApplicationID, resources.EnvironmentID, resources.Profile.ID)
	if err != nil {
		return fmt.Errorf("failed to get latest deployment: %w", err)
	}

	// Handle case when no deployment exists
	if deployment == nil {
		return fmt.Errorf("%w: run 'apcdeploy run' to create the first deployment", ErrNoDeployment)
	}

	e.reporter.Success(fmt.Sprintf("Found deployment #%d (version %s)",
		deployment.DeploymentNumber,
		deployment.ConfigurationVersion,
	))

	// Step 5: Get deployed configuration
	e.reporter.Progress("Fetching deployed configuration...")

	// Parse version number
	versionNum, err := strconv.ParseInt(deployment.ConfigurationVersion, 10, 32)
	if err != nil {
		return fmt.Errorf("invalid version number %s: %w", deployment.ConfigurationVersion, err)
	}

	// Get hosted configuration version with content type
	versionInput := &appconfig.GetHostedConfigurationVersionInput{
		ApplicationId:          awsSdk.String(resources.ApplicationID),
		ConfigurationProfileId: awsSdk.String(resources.Profile.ID),
		VersionNumber:          awsSdk.Int32(int32(versionNum)),
	}

	versionOutput, err := awsClient.GetHostedConfigurationVersion(ctx, versionInput)
	if err != nil {
		return fmt.Errorf("failed to get deployed configuration: %w", err)
	}
	e.reporter.Success("Deployed configuration retrieved")

	// Step 6: Check for changes between local and remote
	e.reporter.Progress("Checking for changes...")

	// Get content type from the version output
	contentType := "application/json" // Default fallback
	if versionOutput.ContentType != nil {
		contentType = *versionOutput.ContentType
	}

	// Determine the full path to the data file
	dataFilePath := cfg.DataFile
	if !filepath.IsAbs(dataFilePath) {
		configDir := filepath.Dir(opts.ConfigFile)
		dataFilePath = filepath.Join(configDir, cfg.DataFile)
	}

	// Load current local data file
	localData, err := config.LoadDataFile(dataFilePath)
	if err != nil {
		// If file doesn't exist or can't be read, proceed with update
		e.reporter.Warning(fmt.Sprintf("Could not read local data file: %v", err))
	} else {
		// Compare local and remote content after normalization
		hasChanges, err := e.hasChanges(string(localData), string(versionOutput.Content), dataFilePath, resources.Profile.Type)
		if err != nil {
			return fmt.Errorf("failed to check for changes: %w", err)
		}

		if !hasChanges {
			e.reporter.Success("No changes detected - local data file is already up to date")
			e.reporter.Success(fmt.Sprintf("Local file matches deployment #%d", deployment.DeploymentNumber))
			return nil
		}

		e.reporter.Success("Changes detected")
	}

	// Step 7: Update data file
	e.reporter.Progress("Updating data file...")

	// Write data file (force=true to overwrite existing file)
	if err := config.WriteDataFile(versionOutput.Content, contentType, dataFilePath, resources.Profile.Type, true); err != nil {
		return fmt.Errorf("failed to write data file: %w", err)
	}

	e.reporter.Success(fmt.Sprintf("Data file updated: %s", dataFilePath))
	e.reporter.Success(fmt.Sprintf("Successfully pulled configuration from deployment #%d", deployment.DeploymentNumber))

	return nil
}

// hasChanges compares local and remote content after normalization.
// Returns true if there are differences, false if they are identical.
// For FeatureFlags profile type, it removes _updatedAt and _createdAt fields before comparing.
func (e *Executor) hasChanges(localContent, remoteContent, fileName, profileType string) (bool, error) {
	// Determine file extension for normalization
	ext := strings.ToLower(filepath.Ext(fileName))

	// Normalize remote content
	normalizedRemote, err := normalizeContent(remoteContent, ext, profileType)
	if err != nil {
		return false, fmt.Errorf("failed to normalize remote content: %w", err)
	}

	// Normalize local content
	normalizedLocal, err := normalizeContent(localContent, ext, profileType)
	if err != nil {
		return false, fmt.Errorf("failed to normalize local content: %w", err)
	}

	// Compare normalized contents
	return normalizedRemote != normalizedLocal, nil
}

// normalizeContent normalizes content based on file type
// For FeatureFlags profile type, it removes _updatedAt and _createdAt from JSON
func normalizeContent(content, ext, profileType string) (string, error) {
	switch ext {
	case ".json":
		return config.NormalizeJSON(content, profileType)
	case ".yaml", ".yml":
		return config.NormalizeYAML(content)
	default:
		// For text files, just ensure consistent line endings
		return config.NormalizeText(content), nil
	}
}
