package init

import (
	"context"
	"fmt"
	"path/filepath"

	awsInternal "github.com/koh-sh/apcdeploy/internal/aws"
	"github.com/koh-sh/apcdeploy/internal/config"
	"github.com/koh-sh/apcdeploy/internal/reporter"
)

// Initializer handles the initialization process
type Initializer struct {
	awsClient *awsInternal.Client
	reporter  reporter.ProgressReporter
}

// New creates a new Initializer
func New(awsClient *awsInternal.Client, rep reporter.ProgressReporter) *Initializer {
	return &Initializer{
		awsClient: awsClient,
		reporter:  rep,
	}
}

// Run executes the initialization process
func (i *Initializer) Run(ctx context.Context, opts *Options) (*Result, error) {
	i.reporter.Progress("Initializing apcdeploy configuration...")

	// Resolve AWS resources
	result, err := i.resolveResources(ctx, opts)
	if err != nil {
		return nil, err
	}

	// Fetch configuration version
	if err := i.fetchConfigVersion(ctx, result); err != nil {
		return nil, err
	}

	// Determine data file name
	i.determineDataFileName(opts, result)

	// Generate files
	if err := i.generateFiles(opts, result); err != nil {
		return nil, err
	}

	// Show next steps
	i.showNextSteps()

	return result, nil
}

// resolveResources resolves AWS resources (Application, Profile, Environment)
func (i *Initializer) resolveResources(ctx context.Context, opts *Options) (*Result, error) {
	i.reporter.Progress("Resolving AWS resources...")

	resolver := awsInternal.NewResolver(i.awsClient)

	// Resolve Application
	appID, err := resolver.ResolveApplication(ctx, opts.Application)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve application: %w", err)
	}

	// Resolve Configuration Profile
	profileInfo, err := resolver.ResolveConfigurationProfile(ctx, appID, opts.Profile)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve configuration profile: %w", err)
	}

	// Resolve Environment
	envID, err := resolver.ResolveEnvironment(ctx, appID, opts.Environment)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve environment: %w", err)
	}

	// Report success
	i.reporter.Success(fmt.Sprintf("Application: %s (ID: %s)", opts.Application, appID))
	i.reporter.Success(fmt.Sprintf("Configuration Profile: %s (ID: %s)", opts.Profile, profileInfo.ID))
	i.reporter.Success(fmt.Sprintf("Environment: %s (ID: %s)", opts.Environment, envID))
	i.reporter.Success(fmt.Sprintf("Profile Type: %s", profileInfo.Type))

	return &Result{
		AppID:       appID,
		AppName:     opts.Application,
		ProfileID:   profileInfo.ID,
		ProfileName: opts.Profile,
		ProfileType: profileInfo.Type,
		EnvID:       envID,
		EnvName:     opts.Environment,
		ConfigFile:  opts.ConfigFile,
	}, nil
}

// fetchConfigVersion fetches the latest configuration version
func (i *Initializer) fetchConfigVersion(ctx context.Context, result *Result) error {
	i.reporter.Progress("Fetching latest configuration version...")

	versionFetcher := awsInternal.NewConfigVersionFetcher(i.awsClient)
	versionInfo, err := versionFetcher.GetLatestVersion(ctx, result.AppID, result.ProfileID)
	if err != nil {
		// If no version exists, we'll create config without data file
		i.reporter.Warning("No configuration versions found - config file will be created without data")
		result.VersionInfo = nil
		return nil
	}

	i.reporter.Success(fmt.Sprintf("Found version: %d (ContentType: %s)", versionInfo.VersionNumber, versionInfo.ContentType))
	result.VersionInfo = versionInfo
	return nil
}

// determineDataFileName determines the appropriate data file name
func (i *Initializer) determineDataFileName(opts *Options, result *Result) {
	switch {
	case opts.OutputData != "":
		result.DataFile = opts.OutputData
	case result.VersionInfo != nil:
		result.DataFile = config.DetermineDataFileName(result.VersionInfo.ContentType)
	default:
		result.DataFile = "data.json" // Default if no version exists
	}
}

// generateFiles generates the configuration and data files
func (i *Initializer) generateFiles(opts *Options, result *Result) error {
	// Generate apcdeploy.yml
	i.reporter.Progress(fmt.Sprintf("Generating configuration file: %s", result.ConfigFile))

	if err := config.GenerateConfigFile(result.AppName, result.ProfileName, result.EnvName, result.DataFile, result.ConfigFile); err != nil {
		return fmt.Errorf("failed to generate config file: %w", err)
	}

	i.reporter.Success(fmt.Sprintf("Created: %s", result.ConfigFile))

	// Write data file if version exists
	if result.VersionInfo != nil {
		dataFilePath := filepath.Join(filepath.Dir(result.ConfigFile), result.DataFile)
		i.reporter.Progress(fmt.Sprintf("Writing configuration data: %s", dataFilePath))

		if err := config.WriteDataFile(result.VersionInfo.Content, result.VersionInfo.ContentType, dataFilePath); err != nil {
			return fmt.Errorf("failed to write data file: %w", err)
		}

		i.reporter.Success(fmt.Sprintf("Created: %s", dataFilePath))
	}

	return nil
}

// showNextSteps displays next steps after initialization
func (i *Initializer) showNextSteps() {
	i.reporter.Success("\nInitialization complete!")
	fmt.Println("\nNext steps:")
	fmt.Println("  1. Review the generated configuration files")
	fmt.Println("  2. Modify the data file as needed")
	fmt.Println("  3. Run 'apcdeploy diff' to preview changes")
	fmt.Println("  4. Run 'apcdeploy deploy' to deploy your configuration")
}
