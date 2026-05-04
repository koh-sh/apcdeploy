package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/koh-sh/apcdeploy/internal/batch"
	"github.com/koh-sh/apcdeploy/internal/cli"
	"github.com/koh-sh/apcdeploy/internal/diff"
	"github.com/koh-sh/apcdeploy/internal/reporter"
	"github.com/spf13/cobra"
)

var (
	diffExitNonzero     bool
	diffParallel        int
	diffContinueOnError bool
)

// DiffCommand returns the diff command
func DiffCommand() *cobra.Command {
	return newDiffCmd()
}

func newDiffCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "diff",
		Short: "Show differences between local configuration and deployed configuration",
		Long: `Show differences between local configuration and the currently deployed configuration in AWS AppConfig.

This command compares your local configuration file with the latest deployed version
and displays the differences in unified diff format.

Pass -c multiple times to diff several configurations in one invocation
(see docs/design/multi-config.md). Multi-target diffs prefix each body
with "=== <region>/<app>/<profile>/<env> ===" so the combined stream
stays unambiguous; single-target output omits the header so it can be
piped into patch/git apply.`,
		RunE:         runDiff,
		SilenceUsage: true, // Don't show usage on runtime errors
	}

	cmd.Flags().BoolVar(&diffExitNonzero, "exit-nonzero", false, "Exit with code 1 if differences exist")
	cmd.Flags().IntVar(&diffParallel, "parallel", 0, "Maximum concurrent targets when -c is repeated (0 = all in parallel)")
	cmd.Flags().BoolVar(&diffContinueOnError, "continue-on-error", false, "Run remaining targets after one fails (default: fail-fast)")

	return cmd
}

func runDiff(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	rep := cli.GetReporter(isSilent())

	// Single-config keeps the existing path so the row identifier still
	// shows the SDK-resolved default region when cfg.Region is empty
	// (multi-config requires region in yml — multi-config.md §6.2).
	if len(configFiles) <= 1 {
		path := defaultConfigFile
		if len(configFiles) == 1 {
			path = configFiles[0]
		}
		opts := &diff.Options{
			ConfigFile:  path,
			ExitNonzero: diffExitNonzero,
			Silent:      isSilent(),
		}
		err := diff.NewExecutor(rep).Execute(ctx, opts)
		if errors.Is(err, diff.ErrDiffFound) {
			os.Exit(1)
		}
		return err
	}

	targets, err := batch.LoadAll(configFiles)
	if err != nil {
		return fmt.Errorf("failed to load configurations: %w", err)
	}

	executor := diff.NewExecutor(rep)
	// Per-target stdout payloads are collected in argument order so the
	// combined stdout stays deterministic regardless of completion order
	// (output.md §10.4). Indexed by target slot — distinct goroutines
	// write to distinct slots so a sync.Mutex around the slice header
	// is sufficient.
	payloads := make([][]byte, len(targets))
	hasChanges := make([]bool, len(targets))
	indexByID := make(map[string]int, len(targets))
	for i, t := range targets {
		indexByID[t.Identifier] = i
	}
	var mu sync.Mutex

	o := &batch.Orchestrator{
		Targets:         targets,
		Parallel:        diffParallel,
		ContinueOnError: diffContinueOnError,
		Reporter:        rep,
		Execute: func(ctx context.Context, t *batch.Target, tr reporter.TargetReporter) error {
			payload, changed, runErr := executor.RunOnTarget(ctx, t, tr)
			mu.Lock()
			idx := indexByID[t.Identifier]
			payloads[idx] = payload
			hasChanges[idx] = changed
			mu.Unlock()
			return runErr
		},
	}
	summary, runErr := o.Run(ctx)

	flushDiffPayloads(rep, targets, payloads)

	renderBatchSummary(rep, summary, len(targets), summaryConfig{noopVerb: "no-op"})

	// --exit-nonzero collapses "any change" across all targets, matching
	// the single-target contract (output.md §7.2 (e)). Even if some
	// targets failed we still exit 1 — both conditions yield non-zero.
	if diffExitNonzero {
		for _, c := range hasChanges {
			if c {
				os.Exit(1)
			}
		}
	}
	return runErr
}

// flushDiffPayloads writes per-target diff bodies to stdout in argument
// order with a `=== <id> ===` header per target (output.md §7.2 stdout
// header rules — N>=2 always carries headers). Empty payloads (no-op
// targets / failed targets) are skipped.
func flushDiffPayloads(rep reporter.Reporter, targets []*batch.Target, payloads [][]byte) {
	for i, t := range targets {
		body := payloads[i]
		if len(body) == 0 {
			continue
		}
		rep.Diff([]byte(fmt.Sprintf("=== %s ===\n", t.Identifier)))
		rep.Diff(body)
		// Insert a blank line between adjacent target diffs so the
		// stream stays readable when piped through `less`.
		if i < len(targets)-1 {
			rep.Diff([]byte("\n"))
		}
	}
}
