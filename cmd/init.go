package cmd

import (
	"context"

	"github.com/koh-sh/apcdeploy/internal/cli"
	initPkg "github.com/koh-sh/apcdeploy/internal/init"
	"github.com/koh-sh/apcdeploy/internal/prompt"
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
		Long: `Initialize apcdeploy configuration by fetching existing AWS AppConfig resources
and generating apcdeploy.yml and data files.

Flags can be provided to skip interactive prompts. If omitted, you will be prompted
to select from available resources.`,
		RunE:         runInit,
		SilenceUsage: true, // Don't show usage on runtime errors
	}

	cmd.Flags().StringVar(&initApp, "app", "", "Application name")
	cmd.Flags().StringVar(&initProfile, "profile", "", "Configuration Profile name")
	cmd.Flags().StringVar(&initEnv, "env", "", "Environment name")
	cmd.Flags().StringVar(&initRegion, "region", "", "AWS region")
	cmd.Flags().StringVarP(&initConfig, "config", "c", "apcdeploy.yml", "Output config file path")
	cmd.Flags().StringVarP(&initOutputData, "output-data", "o", "", "Output data file path")
	cmd.Flags().BoolVarP(&initForce, "force", "f", false, "Overwrite existing files")

	return cmd
}

func runInit(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

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

	// Create reporter and prompter
	reporter := cli.NewReporter()
	prompter := &prompt.HuhPrompter{}

	// Run initialization
	executor := initPkg.NewExecutor(reporter, prompter)
	_, err := executor.Execute(ctx, opts)
	return err
}
