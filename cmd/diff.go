package cmd

import (
	"context"
	"fmt"
	"os"

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

Pass -c multiple times to diff several configurations in one invocation.
Each body is prefixed with "=== <region>/<app>/<profile>/<env> ===" so the
combined stream stays unambiguous regardless of target count.`,
		RunE:         runDiff,
		SilenceUsage: true, // Don't show usage on runtime errors
	}

	cmd.Flags().BoolVar(&diffExitNonzero, "exit-nonzero", false, "Exit with code 1 if differences exist")
	cmd.Flags().IntVar(&diffParallel, "parallel", 0, "Maximum concurrent targets when -c is repeated (0 = all in parallel)")
	cmd.Flags().BoolVar(&diffContinueOnError, "continue-on-error", false, "Run remaining targets after one fails (default: fail-fast)")

	return cmd
}

func runDiff(cmd *cobra.Command, args []string) error {
	ctx := commandContext(cmd)
	rep := cli.GetReporter(isSilent())

	paths, err := resolveConfigTargets(args)
	if err != nil {
		return err
	}
	targets, err := batch.LoadAll(paths)
	if err != nil {
		return fmt.Errorf("failed to load configurations: %w", err)
	}

	executor := diff.NewExecutor(rep)
	// Per-target stdout payloads are collected via PayloadCollector so
	// the synchronisation needed to translate completion order into
	// argument order stays inside internal/batch — see
	// payload_collector.go for why this lives there rather than here.
	collector := batch.NewPayloadCollector(targets)

	o := &batch.Orchestrator{
		Targets:         targets,
		Parallel:        diffParallel,
		ContinueOnError: diffContinueOnError,
		Reporter:        rep,
		Execute: func(ctx context.Context, t *batch.Target, tr reporter.TargetReporter) error {
			res, runErr := executor.RunOnTarget(ctx, t, tr)
			collector.Set(t.Identifier, res.Payload, res.HasChanges, res.Warning)
			return runErr
		},
	}
	summary, runErr := o.Run(ctx)

	renderDiffResults(rep, targets, collector, summary, isSilent())

	// --exit-nonzero collapses "any change" across all targets. Even if
	// some targets failed we still exit 1 — both conditions yield non-zero.
	if diffExitNonzero {
		for _, c := range collector.HasChanges() {
			if c {
				os.Exit(1)
			}
		}
	}
	return runErr
}

// renderDiffResults emits everything diff defers until the orchestrator
// has returned (and so its Targets block is closed), in this order:
//
//  1. per-target diff bodies on stdout (flushDiffPayloads),
//  2. in-progress deployment warnings on stderr, in argument order,
//  3. the aggregate summary line and Errors: section on stderr.
//
// Warnings must not be written while the Targets block is open — the TTY
// renderer's redraw would overwrite them (issue #151). They go after the
// bodies so they stay on screen next to the summary even when a long diff
// scrolls past, and they are emitted regardless of silent (see the
// "diff in-progress warning" contract exception).
func renderDiffResults(rep reporter.Reporter, targets []*batch.Target, collector *batch.PayloadCollector, summary batch.Summary, silent bool) {
	flushDiffPayloads(rep, targets, collector.Payloads())
	diff.WriteDeploymentWarnings(targets, collector.Warnings())
	renderBatchSummary(summary, summaryConfig{noopVerb: "no-op"}, silent)
}

// flushDiffPayloads writes per-target diff bodies to stdout in argument
// order with a `=== <id> ===` header per target. Empty payloads (no-op
// targets / failed targets) are skipped. A blank line is inserted before
// every non-first non-empty payload so the stream stays readable when
// piped through `less`.
func flushDiffPayloads(rep reporter.Reporter, targets []*batch.Target, payloads [][]byte) {
	first := true
	for i, t := range targets {
		body := payloads[i]
		if len(body) == 0 {
			continue
		}
		if !first {
			rep.Diff([]byte("\n"))
		}
		first = false
		rep.Diff(fmt.Appendf(nil, "=== %s ===\n", t.Identifier))
		rep.Diff(body)
	}
}
