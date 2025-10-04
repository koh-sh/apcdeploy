package cmd

import (
	"context"
	"fmt"
	"path/filepath"

	awsInternal "github.com/koh-sh/apcdeploy/internal/aws"
	"github.com/koh-sh/apcdeploy/internal/config"
	"github.com/koh-sh/apcdeploy/internal/display"
	"github.com/spf13/cobra"
)

var (
	initApp        string
	initProfile    string
	initEnv        string
	initRegion     string
	initConfig     string
	initOutputData string
)

// newInitCommand creates a new init command
func newInitCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize apcdeploy configuration from existing AppConfig resources",
		Long: `Initialize apcdeploy configuration by fetching an existing AWS AppConfig
configuration and generating apcdeploy.yml and data files.`,
		RunE: runInit,
	}

	cmd.Flags().StringVar(&initApp, "app", "", "Application name (required)")
	cmd.Flags().StringVar(&initProfile, "profile", "", "Configuration Profile name (required)")
	cmd.Flags().StringVar(&initEnv, "env", "", "Environment name (required)")
	cmd.Flags().StringVar(&initRegion, "region", "", "AWS region (optional, uses default from AWS config)")
	cmd.Flags().StringVarP(&initConfig, "config", "c", "apcdeploy.yml", "Output config file path")
	cmd.Flags().StringVar(&initOutputData, "output-data", "", "Output data file path (optional, auto-detected from ContentType)")

	cmd.MarkFlagRequired("app")
	cmd.MarkFlagRequired("profile")
	cmd.MarkFlagRequired("env")

	return cmd
}

// InitCommand returns the init command to be added to root
func InitCommand() *cobra.Command {
	return newInitCommand()
}

func runInit(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Show initialization message
	fmt.Println(display.Progress("Initializing apcdeploy configuration..."))

	// Initialize AWS client
	awsClient, err := awsInternal.NewClient(ctx, initRegion)
	if err != nil {
		return fmt.Errorf("failed to initialize AWS client: %w", err)
	}

	// Resolve resources
	resolver := awsInternal.NewResolver(awsClient)

	fmt.Println(display.Progress("Resolving AWS resources..."))

	appID, err := resolver.ResolveApplication(ctx, initApp)
	if err != nil {
		return fmt.Errorf("failed to resolve application: %w", err)
	}

	profileInfo, err := resolver.ResolveConfigurationProfile(ctx, appID, initProfile)
	if err != nil {
		return fmt.Errorf("failed to resolve configuration profile: %w", err)
	}

	envID, err := resolver.ResolveEnvironment(ctx, appID, initEnv)
	if err != nil {
		return fmt.Errorf("failed to resolve environment: %w", err)
	}

	// Show resource information
	fmt.Println(display.Success(fmt.Sprintf("Application: %s (ID: %s)", initApp, appID)))
	fmt.Println(display.Success(fmt.Sprintf("Configuration Profile: %s (ID: %s)", initProfile, profileInfo.ID)))
	fmt.Println(display.Success(fmt.Sprintf("Environment: %s (ID: %s)", initEnv, envID)))
	fmt.Println(display.Success(fmt.Sprintf("Profile Type: %s", profileInfo.Type)))

	// Fetch latest configuration version
	fmt.Println(display.Progress("Fetching latest configuration version..."))

	versionFetcher := awsInternal.NewConfigVersionFetcher(awsClient)
	versionInfo, err := versionFetcher.GetLatestVersion(ctx, appID, profileInfo.ID)
	if err != nil {
		// If no version exists, we'll create config without data file
		fmt.Println(display.Warning("No configuration versions found - config file will be created without data"))
		versionInfo = nil
	} else {
		fmt.Println(display.Success(fmt.Sprintf("Found version: %d (ContentType: %s)", versionInfo.VersionNumber, versionInfo.ContentType)))
	}

	// Determine data file name
	var dataFileName string
	switch {
	case initOutputData != "":
		dataFileName = initOutputData
	case versionInfo != nil:
		dataFileName = config.DetermineDataFileName(versionInfo.ContentType)
	default:
		dataFileName = "data.json" // Default if no version exists
	}

	// Generate apcdeploy.yml
	fmt.Println(display.Progress(fmt.Sprintf("Generating configuration file: %s", initConfig)))

	if err := config.GenerateConfigFile(initApp, initProfile, initEnv, dataFileName, initConfig); err != nil {
		return fmt.Errorf("failed to generate config file: %w", err)
	}

	fmt.Println(display.Success(fmt.Sprintf("Created: %s", initConfig)))

	// Write data file if version exists
	if versionInfo != nil {
		dataFilePath := filepath.Join(filepath.Dir(initConfig), dataFileName)
		fmt.Println(display.Progress(fmt.Sprintf("Writing configuration data: %s", dataFilePath)))

		if err := config.WriteDataFile(versionInfo.Content, versionInfo.ContentType, dataFilePath); err != nil {
			return fmt.Errorf("failed to write data file: %w", err)
		}

		fmt.Println(display.Success(fmt.Sprintf("Created: %s", dataFilePath)))
	}

	// Show next steps
	fmt.Println("\n" + display.Success("Initialization complete!"))
	fmt.Println("\nNext steps:")
	fmt.Println("  1. Review the generated configuration files")
	fmt.Println("  2. Modify the data file as needed")
	fmt.Println("  3. Run 'apcdeploy diff' to preview changes")
	fmt.Println("  4. Run 'apcdeploy deploy' to deploy your configuration")

	return nil
}
