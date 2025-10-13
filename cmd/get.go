package cmd

import (
	"context"

	"github.com/koh-sh/apcdeploy/internal/cli"
	"github.com/koh-sh/apcdeploy/internal/get"
	"github.com/koh-sh/apcdeploy/internal/prompt"
	"github.com/spf13/cobra"
)

// getSkipConfirmation controls whether to skip the confirmation prompt (--yes flag)
var getSkipConfirmation bool

// GetCommand returns the get command
func GetCommand() *cobra.Command {
	return newGetCmd()
}

func newGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get configuration from AWS AppConfig",
		Long: `Get the latest deployed configuration from AWS AppConfig and output to stdout.

WARNING: This command uses AWS AppConfig Data API which incurs charges per API call.
Use --yes to skip the confirmation prompt (useful for scripts and automation).`,
		RunE:         runGet,
		SilenceUsage: true, // Don't show usage on runtime errors
	}

	cmd.Flags().BoolVarP(&getSkipConfirmation, "yes", "y", false, "Skip confirmation prompt")

	return cmd
}

func runGet(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Create options
	opts := &get.Options{
		ConfigFile:       configFile,
		Silent:           isSilent(),
		SkipConfirmation: getSkipConfirmation,
	}

	// Create reporter and prompter
	reporter := cli.GetReporter(isSilent())
	prompter := &prompt.HuhPrompter{}

	// Get configuration
	executor := get.NewExecutor(reporter, prompter)
	return executor.Execute(ctx, opts)
}
