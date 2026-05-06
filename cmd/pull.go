package cmd

import (
	"fmt"

	"github.com/koh-sh/apcdeploy/internal/batch"
	"github.com/koh-sh/apcdeploy/internal/cli"
	"github.com/koh-sh/apcdeploy/internal/pull"
	"github.com/spf13/cobra"
)

var (
	pullParallel        int
	pullContinueOnError bool
)

// PullCommand returns the pull command
func PullCommand() *cobra.Command {
	return newPullCmd()
}

func newPullCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pull",
		Short: "Pull latest deployed configuration and update local data file",
		Long: `Pull the latest deployed configuration from AWS AppConfig and update the local data file.

This command retrieves the currently deployed configuration and overwrites your local data file.
Useful when configuration changes are made directly in the AWS Console and you want to sync
your local files with the deployed state.

Pass -c multiple times to pull several configurations in one invocation.

Note: This command does NOT use the AppConfig Data API, so it does not incur per-call charges.`,
		RunE:         runPull,
		SilenceUsage: true, // Don't show usage on runtime errors
	}

	cmd.Flags().IntVar(&pullParallel, "parallel", 0, "Maximum concurrent targets when -c is repeated (0 = all in parallel)")
	cmd.Flags().BoolVar(&pullContinueOnError, "continue-on-error", false, "Run remaining targets after one fails (default: fail-fast)")

	return cmd
}

func runPull(cmd *cobra.Command, args []string) error {
	ctx := commandContext(cmd)
	rep := cli.GetReporter(isSilent())

	targets, err := batch.LoadAll(configFiles)
	if err != nil {
		return fmt.Errorf("failed to load configurations: %w", err)
	}

	executor := pull.NewExecutor(rep)
	o := &batch.Orchestrator{
		Targets:         targets,
		Parallel:        pullParallel,
		ContinueOnError: pullContinueOnError,
		Reporter:        rep,
		Execute:         executor.RunOnTarget,
	}
	summary, runErr := o.Run(ctx)
	renderBatchSummary(summary, summaryConfig{noopVerb: "no-op"}, isSilent())
	return runErr
}
