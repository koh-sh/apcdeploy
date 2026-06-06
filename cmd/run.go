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
	// target — it is not a global wall-clock cap.
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

// validateRunFlags checks the run-command flags whose validity does not
// depend on per-target state. Run once at the cmd layer (rather than
// inside each per-target RunOnTarget call) so the user gets the same
// error regardless of -c count and the orchestrator never starts when
// the flags are bogus.
func validateRunFlags() error {
	if runTimeout <= 0 {
		return fmt.Errorf("timeout must be greater than 0")
	}
	if runWaitDeploy && runWaitBake {
		return fmt.Errorf("--wait-deploy and --wait-bake cannot be used together")
	}
	return nil
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

Pass -c multiple times to run several deployments in one invocation.
Multi-target runs honour --parallel and --continue-on-error; --timeout
is per-target, not global.`,
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
	ctx := commandContext(cmd)

	if err := validateDescription(runDescription); err != nil {
		return err
	}
	if err := validateRunFlags(); err != nil {
		return err
	}
	description := resolveDescription(cmd, runDescription)

	rep := cli.GetReporter(isSilent())

	paths, err := resolveConfigTargets(args)
	if err != nil {
		return err
	}
	targets, err := batch.LoadAll(paths)
	if err != nil {
		return fmt.Errorf("failed to load configurations: %w", err)
	}

	executor := run.NewExecutor(rep)
	// opts is shared (by pointer) across all per-target goroutines launched
	// by the orchestrator below. All fields MUST be treated as read-only
	// after construction. If a future change needs per-target mutation,
	// switch to a per-target copy inside the Execute closure.
	opts := &run.Options{
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
			return executor.RunOnTarget(ctx, t, tr, opts)
		},
	}
	summary, runErr := o.Run(ctx)
	withElapsed := runWaitDeploy || runWaitBake
	renderBatchSummary(summary, summaryConfig{noopVerb: "no-op", withElapsed: withElapsed}, isSilent())
	return runErr
}
