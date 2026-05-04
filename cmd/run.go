package cmd

import (
	"context"
	"fmt"

	"github.com/koh-sh/apcdeploy/internal/batch"
	"github.com/koh-sh/apcdeploy/internal/cli"
	"github.com/koh-sh/apcdeploy/internal/reporter"
	"github.com/koh-sh/apcdeploy/internal/run"
	"github.com/spf13/cobra"
)

const (
	// DefaultDeploymentTimeout is the default timeout for deployments in seconds.
	// Set to 30 minutes to safely cover AppConfig.AllAtOnce (10 min bake) and
	// AppConfig.Canary10Percent20Minutes (20 min deploy + 10 min bake) under
	// --wait-bake. Strategies with longer total durations (e.g.
	// AppConfig.Linear20PercentEvery6Minutes) require an explicit --timeout.
	//
	// In the multi-config path the same value applies INDEPENDENTLY to each
	// target (multi-config.md §7.4) — it is not a global wall-clock cap.
	DefaultDeploymentTimeout = 1800
)

var (
	runWaitDeploy      bool
	runWaitBake        bool
	runTimeout         int
	runForce           bool
	runDescription     string
	runParallel        int
	runContinueOnError bool
)

// RunCommand returns the run command
func RunCommand() *cobra.Command {
	return newRunCmd()
}

func newRunCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run configuration deployment to AWS AppConfig",
		Long: `Run configuration deployment to AWS AppConfig.

This command will:
1. Load the local configuration file (apcdeploy.yml)
2. Validate the configuration data
3. Create a new hosted configuration version
4. Start a deployment to the specified environment
5. Optionally wait for the deployment phase (--wait-deploy) or full completion (--wait-bake)

Pass -c multiple times to run several deployments in one invocation
(see docs/design/multi-config.md). Multi-target runs honour --parallel
and --continue-on-error; --timeout is per-target, not global.`,
		RunE:         runRun,
		SilenceUsage: true, // Don't show usage on runtime errors
	}

	cmd.Flags().BoolVar(&runWaitDeploy, "wait-deploy", false, "Wait for deployment phase to complete (until baking starts)")
	cmd.Flags().BoolVar(&runWaitBake, "wait-bake", false, "Wait for complete deployment including baking phase")
	cmd.Flags().IntVar(&runTimeout, "timeout", DefaultDeploymentTimeout, "Per-target timeout in seconds for deployment")
	cmd.Flags().BoolVar(&runForce, "force", false, "Force deployment even when there are no changes")
	cmd.Flags().StringVar(&runDescription, "description", "", fmt.Sprintf(`Description attached to the configuration version and deployment (max %d chars; defaults to %q, pass "" to clear)`, maxDescriptionLength, defaultDescription))
	cmd.Flags().IntVar(&runParallel, "parallel", 0, "Maximum concurrent targets when -c is repeated (0 = all in parallel)")
	cmd.Flags().BoolVar(&runContinueOnError, "continue-on-error", false, "Run remaining targets after one fails (default: fail-fast)")

	return cmd
}

func runRun(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	if err := validateDescription(runDescription); err != nil {
		return err
	}
	description := resolveDescription(cmd, runDescription)

	rep := cli.GetReporter(isSilent())

	// Single-config keeps the existing path so the row identifier still
	// shows the SDK-resolved default region when cfg.Region is empty
	// (multi-config requires region in yml — multi-config.md §6.2).
	if len(configFiles) <= 1 {
		path := defaultConfigFile
		if len(configFiles) == 1 {
			path = configFiles[0]
		}
		opts := &run.Options{
			ConfigFile:  path,
			WaitDeploy:  runWaitDeploy,
			WaitBake:    runWaitBake,
			Timeout:     runTimeout,
			Force:       runForce,
			Description: description,
		}
		executor := run.NewExecutor(rep)
		return executor.Execute(ctx, opts)
	}

	targets, err := batch.LoadAll(configFiles)
	if err != nil {
		return fmt.Errorf("failed to load configurations: %w", err)
	}

	executor := run.NewExecutor(rep)
	baseOpts := &run.Options{
		WaitDeploy:  runWaitDeploy,
		WaitBake:    runWaitBake,
		Timeout:     runTimeout,
		Force:       runForce,
		Description: description,
	}

	o := &batch.Orchestrator{
		Targets:         targets,
		Parallel:        runParallel,
		ContinueOnError: runContinueOnError,
		Reporter:        rep,
		Execute: func(ctx context.Context, t *batch.Target, tr reporter.TargetReporter) error {
			// Each call gets its own Options copy via the per-target
			// ConfigFile (kept for parity with single-config — the
			// executor reads it for diagnostics but the data file path
			// already lives in t.Config.DataFile).
			perTarget := *baseOpts
			perTarget.ConfigFile = t.Path
			return executor.RunOnTarget(ctx, t, tr, &perTarget)
		},
	}
	summary, runErr := o.Run(ctx)
	withElapsed := runWaitDeploy || runWaitBake
	renderBatchSummary(rep, summary, len(targets), summaryConfig{noopVerb: "no-op", withElapsed: withElapsed})
	return runErr
}
