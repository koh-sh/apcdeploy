package pull

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/koh-sh/apcdeploy/internal/aws"
	"github.com/koh-sh/apcdeploy/internal/config"
	"github.com/koh-sh/apcdeploy/internal/reporter"
)

// Executor handles the pull operation orchestration
type Executor struct {
	reporter      reporter.Reporter
	clientFactory func(context.Context, string) (*aws.Client, error)
}

// NewExecutor creates a new pull executor
func NewExecutor(rep reporter.Reporter) *Executor {
	return &Executor{
		reporter:      rep,
		clientFactory: aws.NewClient,
	}
}

// NewExecutorWithFactory creates a new pull executor with a custom client factory
// This is useful for testing with mock clients
func NewExecutorWithFactory(rep reporter.Reporter, factory func(context.Context, string) (*aws.Client, error)) *Executor {
	return &Executor{
		reporter:      rep,
		clientFactory: factory,
	}
}

// Execute performs the complete pull workflow
func (e *Executor) Execute(ctx context.Context, opts *Options) error {
	// Step 1: Load configuration
	e.reporter.Step("Loading configuration...")
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
	e.reporter.Step("Resolving resources...")
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

	// Step 4: Get latest deployed configuration
	e.reporter.Step("Fetching latest deployed configuration...")
	deployedConfig, err := aws.GetLatestDeployedConfiguration(ctx, awsClient, resources.ApplicationID, resources.EnvironmentID, resources.Profile.ID)
	if err != nil {
		return fmt.Errorf("failed to get latest deployed configuration: %w", err)
	}

	// Handle case when no deployment exists
	if deployedConfig == nil {
		return fmt.Errorf("%w: run 'apcdeploy run' to create the first deployment", aws.ErrNoDeployment)
	}

	e.reporter.Success(fmt.Sprintf("Found deployment #%d (version %d)",
		deployedConfig.DeploymentNumber,
		deployedConfig.VersionNumber,
	))

	// Step 5: Check for changes between local and remote
	e.reporter.Step("Checking for changes...")

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
		e.reporter.Warn(fmt.Sprintf("Could not read local data file: %v", err))
	} else {
		// Compare local and remote content after normalization
		ext := filepath.Ext(dataFilePath)
		hasChanges, err := config.HasContentChanged(localData, deployedConfig.Content, ext, resources.Profile.Type)
		if err != nil {
			return fmt.Errorf("failed to check for changes: %w", err)
		}

		if !hasChanges {
			e.reporter.Success("No changes detected - local data file is already up to date")
			e.reporter.Success(fmt.Sprintf("Local file matches deployment #%d", deployedConfig.DeploymentNumber))
			return nil
		}

		e.reporter.Success("Changes detected")
	}

	// Step 6: Update data file
	e.reporter.Step("Updating data file...")

	// Write data file (force=true to overwrite existing file)
	if err := config.WriteDataFile(deployedConfig.Content, deployedConfig.ContentType, dataFilePath, resources.Profile.Type, true); err != nil {
		return fmt.Errorf("failed to write data file: %w", err)
	}

	e.reporter.Success(fmt.Sprintf("Data file updated: %s", dataFilePath))
	e.reporter.Success(fmt.Sprintf("Successfully pulled configuration from deployment #%d", deployedConfig.DeploymentNumber))

	return nil
}
