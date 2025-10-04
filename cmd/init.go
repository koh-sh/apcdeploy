package cmd

import (
	"context"
	"fmt"

	awsInternal "github.com/koh-sh/apcdeploy/internal/aws"
	"github.com/koh-sh/apcdeploy/internal/display"
	initPkg "github.com/koh-sh/apcdeploy/internal/init"
	"github.com/spf13/cobra"
)

var (
	initApp        string
	initProfile    string
	initEnv        string
	initRegion     string
	initConfig     string
	initOutputData string

	// initializerFactory allows dependency injection for testing
	initializerFactory func(context.Context, string) (*initPkg.Initializer, error)
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

	initializer, err := createInitializer(ctx)
	if err != nil {
		return fmt.Errorf("failed to create initializer: %w", err)
	}

	opts := &initPkg.Options{
		Application: initApp,
		Profile:     initProfile,
		Environment: initEnv,
		Region:      initRegion,
		ConfigFile:  initConfig,
		OutputData:  initOutputData,
	}

	result, err := initializer.Run(ctx, opts)
	if err != nil {
		return err
	}

	showNextSteps(result)
	return nil
}

func createInitializer(ctx context.Context) (*initPkg.Initializer, error) {
	if initializerFactory != nil {
		return initializerFactory(ctx, initRegion)
	}
	return createDefaultInitializer(ctx)
}

func createDefaultInitializer(ctx context.Context) (*initPkg.Initializer, error) {
	awsClient, err := awsInternal.NewClient(ctx, initRegion)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize AWS client: %w", err)
	}

	reporter := &cliReporter{}
	return initPkg.New(awsClient, reporter), nil
}

func showNextSteps(result *initPkg.Result) {
	fmt.Println("\n" + display.Success("Initialization complete!"))
	fmt.Println("\nNext steps:")
	fmt.Println("  1. Review the generated configuration files")
	fmt.Println("  2. Modify the data file as needed")
	fmt.Println("  3. Run 'apcdeploy diff' to preview changes")
	fmt.Println("  4. Run 'apcdeploy deploy' to deploy your configuration")

	// Suppress unused variable warning
	_ = result
}
