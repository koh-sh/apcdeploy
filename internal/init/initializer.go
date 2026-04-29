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
	reporter  reporter.Reporter
}

// New creates a new Initializer
func New(awsClient *awsInternal.Client, rep reporter.Reporter) *Initializer {
	return &Initializer{
		awsClient: awsClient,
		reporter:  rep,
	}
}

// Run executes the initialization process
func (i *Initializer) Run(ctx context.Context, opts *Options) (*Result, error) {
	// The "Initializing apcdeploy configuration" banner is intentionally not
	// emitted: the user already knows they ran `apcdeploy init`. Each phase
	// reports its own progress.

	result, err := i.resolveResources(ctx, opts)
	if err != nil {
		return nil, err
	}

	if err := i.fetchConfigVersion(ctx, result); err != nil {
		return nil, err
	}

	i.fetchDeploymentStrategy(ctx, result)

	i.determineDataFileName(opts, result)

	if err := i.generateFiles(opts, result); err != nil {
		return nil, err
	}

	i.showNextSteps()

	return result, nil
}

// resolveResources resolves AWS resources (Application, Profile, Environment)
func (i *Initializer) resolveResources(ctx context.Context, opts *Options) (*Result, error) {
	sp := i.reporter.Spin("Resolving AWS resources...")
	resolver := awsInternal.NewResolver(i.awsClient)
	resolved, err := resolver.ResolveAll(ctx, opts.Application, opts.Profile, opts.Environment, "")
	if err != nil {
		sp.Stop()
		return nil, err
	}
	// Single-line summary replaces the prior 4-line stack of Success calls.
	// The detailed IDs are not surfaced because the user just selected these
	// resources by name and rarely needs to see their AWS IDs.
	sp.Done(fmt.Sprintf("Resolved resources: App=%s, Profile=%s, Env=%s, Type=%s",
		opts.Application, opts.Profile, opts.Environment, resolved.Profile.Type))

	return &Result{
		AppID:       resolved.ApplicationID,
		AppName:     opts.Application,
		ProfileID:   resolved.Profile.ID,
		ProfileName: opts.Profile,
		ProfileType: resolved.Profile.Type,
		EnvID:       resolved.EnvironmentID,
		EnvName:     opts.Environment,
		ConfigFile:  opts.ConfigFile,
	}, nil
}

// fetchConfigVersion fetches the latest deployed configuration version
func (i *Initializer) fetchConfigVersion(ctx context.Context, result *Result) error {
	sp := i.reporter.Spin("Fetching latest deployed configuration...")
	deployedConfig, err := awsInternal.GetLatestDeployedConfiguration(ctx, i.awsClient, result.AppID, result.EnvID, result.ProfileID)
	if err != nil {
		sp.Stop()
		return fmt.Errorf("failed to get latest deployed configuration: %w", err)
	}

	if deployedConfig == nil {
		// "No deployment" is reported as the spinner's Done message rather
		// than a separate Warn so the absence of a prior deployment shows up
		// inline with the fetch phase.
		sp.Done("No prior deployment — config file will be created without data")
		result.DeployedConfig = nil
		return nil
	}

	sp.Done(fmt.Sprintf("Loaded deployed configuration (deployment #%d, version %d, %s)",
		deployedConfig.DeploymentNumber,
		deployedConfig.VersionNumber,
		deployedConfig.ContentType))

	result.DeployedConfig = deployedConfig
	return nil
}

// fetchDeploymentStrategy fetches the deployment strategy from the latest deployment
func (i *Initializer) fetchDeploymentStrategy(ctx context.Context, result *Result) {
	sp := i.reporter.Spin("Fetching latest deployment strategy...")

	latestDeployment, err := awsInternal.GetLatestDeployment(ctx, i.awsClient, result.AppID, result.EnvID, result.ProfileID)
	if err != nil || latestDeployment == nil {
		result.DeploymentStrategy = config.DefaultDeploymentStrategy
		sp.Done(fmt.Sprintf("Using default deployment strategy: %s", config.DefaultDeploymentStrategy))
		return
	}

	deploymentDetails, err := awsInternal.GetDeploymentDetails(ctx, i.awsClient, result.AppID, result.EnvID, latestDeployment.DeploymentNumber)
	if err != nil {
		result.DeploymentStrategy = config.DefaultDeploymentStrategy
		sp.Done(fmt.Sprintf("Could not retrieve previous strategy — using default: %s", config.DefaultDeploymentStrategy))
		return
	}

	resolver := awsInternal.NewResolver(i.awsClient)
	strategyName, err := resolver.ResolveDeploymentStrategyIDToName(ctx, deploymentDetails.DeploymentStrategyID)
	if err != nil {
		// Fall back to the raw ID; the user can rename it in apcdeploy.yml.
		result.DeploymentStrategy = deploymentDetails.DeploymentStrategyID
	} else {
		result.DeploymentStrategy = strategyName
	}
	sp.Done(fmt.Sprintf("Using deployment strategy from latest deployment: %s", result.DeploymentStrategy))
}

// determineDataFileName determines the appropriate data file name
func (i *Initializer) determineDataFileName(opts *Options, result *Result) {
	switch {
	case opts.OutputData != "":
		result.DataFile = opts.OutputData
	case result.DeployedConfig != nil:
		result.DataFile = config.DetermineDataFileName(result.DeployedConfig.ContentType)
	default:
		result.DataFile = "data.json" // Default if no version exists
	}
}

// generateFiles generates the configuration and data files
func (i *Initializer) generateFiles(opts *Options, result *Result) error {
	// File writes are instant local operations — no spinner needed; a single
	// Success line per file is the user-facing signal that the file landed.
	if err := config.GenerateConfigFile(result.AppName, result.ProfileName, result.EnvName, result.DataFile, i.awsClient.Region, result.DeploymentStrategy, result.ConfigFile, opts.Force); err != nil {
		return fmt.Errorf("failed to generate config file: %w", err)
	}
	i.reporter.Success(fmt.Sprintf("Generated %s", result.ConfigFile))

	if result.DeployedConfig != nil {
		dataFilePath := filepath.Join(filepath.Dir(result.ConfigFile), result.DataFile)
		if err := config.WriteDataFile(result.DeployedConfig.Content, result.DeployedConfig.ContentType, dataFilePath, result.ProfileType, opts.Force); err != nil {
			return fmt.Errorf("failed to write data file: %w", err)
		}
		i.reporter.Success(fmt.Sprintf("Wrote %s", dataFilePath))
	}

	return nil
}

// showNextSteps displays next steps after initialization.
func (i *Initializer) showNextSteps() {
	i.reporter.Success("Initialization complete!")
	i.reporter.Box("Next steps", []string{
		"  1. Review the generated configuration files",
		"  2. Modify the data file as needed",
		"  3. Run 'apcdeploy diff' to preview changes",
		"  4. Run 'apcdeploy deploy' to deploy your configuration",
	})
}
