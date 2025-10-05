package cmd

import (
	"context"

	awsInternal "github.com/koh-sh/apcdeploy/internal/aws"
	"github.com/koh-sh/apcdeploy/internal/cli"
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
	initForce      bool
)

// InitCommand returns the init command
func InitCommand() *cobra.Command {
	return newInitCmd()
}

func newInitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize apcdeploy configuration from existing AppConfig resources",
		Long: `Initialize apcdeploy configuration by fetching an existing AWS AppConfig
configuration and generating apcdeploy.yml and data files.`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			// Show usage help when required flags are missing
			if initApp == "" || initProfile == "" || initEnv == "" {
				cmd.SilenceUsage = false
			}
			return nil
		},
		RunE:         runInit,
		SilenceUsage: true, // Don't show usage on runtime errors (e.g., file exists)
	}

	cmd.Flags().StringVar(&initApp, "app", "", "Application name (required)")
	cmd.Flags().StringVar(&initProfile, "profile", "", "Configuration Profile name (required)")
	cmd.Flags().StringVar(&initEnv, "env", "", "Environment name (required)")
	cmd.Flags().StringVar(&initRegion, "region", "", "AWS region (optional, uses default from AWS config)")
	cmd.Flags().StringVarP(&initConfig, "config", "c", "apcdeploy.yml", "Output config file path")
	cmd.Flags().StringVar(&initOutputData, "output-data", "", "Output data file path (optional, auto-detected from ContentType)")
	cmd.Flags().BoolVar(&initForce, "force", false, "Overwrite existing files if they already exist")

	cmd.MarkFlagRequired("app")
	cmd.MarkFlagRequired("profile")
	cmd.MarkFlagRequired("env")

	return cmd
}

func runInit(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Create AWS client
	awsClient, err := awsInternal.NewClient(ctx, initRegion)
	if err != nil {
		return err
	}

	// Create options
	opts := &initPkg.Options{
		Application: initApp,
		Profile:     initProfile,
		Environment: initEnv,
		Region:      initRegion,
		ConfigFile:  initConfig,
		OutputData:  initOutputData,
		Force:       initForce,
	}

	// Create reporter
	reporter := cli.NewReporter()

	// Run initialization
	initializer := initPkg.New(awsClient, reporter)
	_, err = initializer.Run(ctx, opts)
	return err
}
