package cmd

import (
	"fmt"

	"github.com/koh-sh/apcdeploy/internal/batch"
	"github.com/koh-sh/apcdeploy/internal/cli"
	"github.com/koh-sh/apcdeploy/internal/validate"
	"github.com/spf13/cobra"
)

var (
	validateParallel        int
	validateContinueOnError bool
)

// ValidateCommand returns the validate command
func ValidateCommand() *cobra.Command {
	return newValidateCmd()
}

func newValidateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate local configuration data against its schema",
		Long: `Validate the local data file against its AppConfig schema without deploying.

This is a read-only check: it resolves the profile, reads your local data file,
and validates it without creating a configuration version (no write APIs are
called).

Validation performed:
  - FeatureFlags: structure against the built-in AWS schema, plus each value
    against the constraints declared in the data.
  - Freeform JSON: validated against the profile's JSON_SCHEMA validator fetched
    from AWS (the same validator AWS enforces at deploy time). If the profile has
    no JSON_SCHEMA validator, only JSON syntax is checked.
  - Freeform YAML/text: syntax only (JSON Schema cannot apply).

LAMBDA validators are not supported and are skipped (they can only run in AWS).

Pass -c multiple times to validate several configurations in one invocation.`,
		RunE:         runValidate,
		SilenceUsage: true, // Don't show usage on runtime errors
	}

	cmd.Flags().IntVar(&validateParallel, "parallel", 0, "Maximum concurrent targets when -c is repeated (0 = all in parallel)")
	cmd.Flags().BoolVar(&validateContinueOnError, "continue-on-error", false, "Run remaining targets after one fails (default: fail-fast)")

	return cmd
}

func runValidate(cmd *cobra.Command, args []string) error {
	ctx := commandContext(cmd)
	rep := cli.GetReporter(isSilent())

	targets, err := batch.LoadAll(configFiles)
	if err != nil {
		return fmt.Errorf("failed to load configurations: %w", err)
	}

	executor := validate.NewExecutor(rep)
	o := &batch.Orchestrator{
		Targets:         targets,
		Parallel:        validateParallel,
		ContinueOnError: validateContinueOnError,
		Reporter:        rep,
		Execute:         executor.RunOnTarget,
	}
	summary, runErr := o.Run(ctx)
	renderBatchSummary(summary, summaryConfig{noopVerb: "no-op"}, isSilent())
	return runErr
}
