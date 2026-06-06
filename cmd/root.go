package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"unicode/utf8"

	"github.com/koh-sh/apcdeploy/internal/apcerrors"
	awsInternal "github.com/koh-sh/apcdeploy/internal/aws"
	"github.com/koh-sh/apcdeploy/internal/batch"
	"github.com/koh-sh/apcdeploy/internal/cli"
	"github.com/koh-sh/apcdeploy/internal/reporter"
	"github.com/koh-sh/apcdeploy/internal/rollback"
	"github.com/spf13/cobra"
)

// Exit codes used by the CLI. Anything other than 0/1 is considered a
// distinguishable condition that scripts can branch on.
const (
	// exitNoDeployment is returned for the "no relevant deployment to operate
	// on" family of errors:
	//   - awsInternal.ErrNoDeployment (pull / edit / status: never been deployed)
	//   - rollback.ErrNoOngoingDeployment (rollback: nothing in flight to stop)
	// Scripts can use this code to branch on "no work to do" without parsing
	// stderr.
	exitNoDeployment = 2
	// exitInterrupted matches the Unix convention `128 + SIGINT (2)`. Used
	// for both the graceful-cancellation path (first Ctrl+C → ctx canceled
	// → command returns context.Canceled) and the force-quit path (second
	// Ctrl+C → os.Exit straight from the signal-watcher goroutine).
	exitInterrupted = 130
)

var (
	// Version information
	version string
	commit  string
	date    string

	// Global flags. configFiles is a slice so the run/diff/pull commands can
	// accept `-c` repeatedly. Single-config commands
	// (init/get/status/rollback/edit) call requireSingleConfig() to enforce
	// `len == 1` themselves.
	configFiles []string
	silent      bool
)

// defaultConfigFile is the implicit -c value when the user passes no
// `-c` flag. Kept as a constant so tests and the flag default agree.
const defaultConfigFile = "apcdeploy.yml"

// NewRootCommand creates and returns the root command
func NewRootCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "apcdeploy",
		Short: "AWS AppConfig deployment tool",
		Long: `apcdeploy is a CLI tool for managing AWS AppConfig deployments.
It provides commands to initialize, deploy, diff, and check the status of configurations.`,
		Version: fmt.Sprintf("%s (Built on %s from Git SHA %s)", version, date, commit),
	}

	// Global flags. `-c` is a string slice so run/diff/pull can accept it
	// multiple times for multi-config; single-config commands validate
	// the count via requireSingleConfig().
	rootCmd.PersistentFlags().StringSliceVarP(&configFiles, "config", "c", []string{defaultConfigFile}, "config file path (run/diff/pull may pass -c multiple times)")
	rootCmd.PersistentFlags().BoolVarP(&silent, "silent", "s", false, "suppress verbose output, show only essential information")

	// Add subcommands
	rootCmd.AddCommand(InitCommand())
	rootCmd.AddCommand(RunCommand())
	rootCmd.AddCommand(DiffCommand())
	rootCmd.AddCommand(StatusCommand())
	rootCmd.AddCommand(GetCommand())
	rootCmd.AddCommand(PullCommand())
	rootCmd.AddCommand(ValidateCommand())
	rootCmd.AddCommand(RollbackCommand())
	rootCmd.AddCommand(LsResourcesCommand())
	rootCmd.AddCommand(ContextCommand())
	rootCmd.AddCommand(EditCommand())

	return rootCmd
}

// SetVersionInfo sets version information from build-time variables
func SetVersionInfo(v, c, d string) {
	version = v
	commit = c
	date = d
}

// Execute runs the root command. The os.Exit call lives here so the rest
// of the program can return exit codes through runRoot(), keeping defer'd
// cleanup (signal handler unregister, watcher goroutine teardown)
// reachable. Named runRoot rather than run to avoid colliding with the
// `internal/run` package identifier used elsewhere in cmd/.
func Execute() {
	os.Exit(runRoot())
}

// runRoot drives the root command and returns the process exit code.
//
// Signal handling: the first SIGINT/SIGTERM cancels the context so the
// command can clean up (AWS Wait* polling exits via ctx.Done()). A second
// signal force-exits with code 130 — matching the kubectl/docker
// convention — so users can escape if a Go-external operation (DNS lookup,
// stalled syscall, child editor process) keeps the program from returning
// after the first Ctrl+C.
//
// CLAUDE.md / output-contract.md document the implicit contract that all
// long-running operations honor ctx.Done(); the second-signal goroutine is
// the safety net for the Go-external cases where that contract can't reach.
func runRoot() int {
	rootCmd := NewRootCommand()

	// Enable custom error formatting
	rootCmd.SilenceErrors = true

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// sigCh is registered up-front (before ctx.Done() can fire) so the
	// second-signal watcher cannot lose a fast Ctrl+C double-tap to a
	// race window between ctx cancellation and signal.Notify. Buffer 2
	// holds both signals if they land back-to-back; we drain the first
	// one (which corresponds to the ctx-cancel signal) so only signals
	// arriving AFTER ctx.Done count as "second".
	sigCh := make(chan os.Signal, 2)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	// done terminates the watcher goroutine when runRoot returns normally
	// so it doesn't leak across test runs.
	done := make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-ctx.Done():
		case <-done:
			return
		}
		// Discard the signal that triggered ctx.Done (it landed in sigCh
		// too because signal.Notify was already registered). Non-blocking:
		// when ctx was cancelled by something other than a signal (parent
		// cancellation, test) sigCh is empty and we fall through.
		select {
		case <-sigCh:
		default:
		}
		select {
		case <-sigCh:
			// Second signal: bypass deferred cleanup intentionally — the
			// whole process is going away, so unregistering signal handlers
			// is moot, and the parent goroutine may already be wedged on a
			// syscall (the very thing this watcher exists to escape).
			os.Exit(exitInterrupted)
		case <-done:
		}
	}()

	return classifyAndReport(rootCmd.ExecuteContext(ctx), cli.GetReporter(silent))
}

// classifyAndReport renders the top-level error through the Reporter and
// returns the process exit code. Split out from runRoot so the exit-code
// branches (context.Canceled → 130, ErrNoDeployment → 2, generic → 1) can
// be unit-tested without exercising the cobra/signal plumbing.
//
// nil err returns 0 so callers can pass rootCmd.ExecuteContext(ctx)
// directly without a special case.
func classifyAndReport(err error, rep reporter.Reporter) int {
	if err == nil {
		return 0
	}
	// User-cancelled run (first Ctrl+C). Distinct exit code, friendlier
	// line via Warn so the visual cue isn't a red ✗ (this isn't a failure
	// in the AWS sense — the user asked for it).
	if errors.Is(err, context.Canceled) {
		rep.Warn("cancelled by user")
		return exitInterrupted
	}
	// Funnel the top-level error through the Reporter so the styled "✗"
	// prefix is consistent with the rest of stderr output. Both real and
	// silent reporters always emit Error.
	//
	// Exception: a single-target batch failure is already surfaced in full
	// by the per-target Targets.Fail (which forwards the detailed error
	// through Error even under --silent). Emitting the aggregate
	// "1 of 1 targets failed" summary on top would just duplicate that line,
	// so suppress the top-level Error for that case. The multi-target
	// "N of M" count is informative, so it is still emitted. See issue #109.
	var aggErr *batch.AggregateError
	singleTargetBatch := errors.As(err, &aggErr) && aggErr.Total == 1
	if !singleTargetBatch {
		rep.Error(err.Error())
	}
	// Append a Resolution: <hint> line when the underlying AWS error code
	// has a documented remediation.
	// Emitted via Warn (⚠) instead of Error (✗) so the visual hierarchy is
	// "what failed" first, "how to recover" second. Warn is suppressed by
	// the silent variant, so the hint surfaces in interactive runs but not
	// under --silent — automation should rely on the exit code and the
	// (always-emitted) Error line above instead.
	if hint := apcerrors.Resolution(err); hint != "" {
		rep.Warn("Resolution: " + hint)
	}
	// Exit 2 for the "no relevant deployment" family — pull/edit when no
	// prior deployment exists, and rollback when nothing is in flight to
	// stop. Scripts can distinguish these "no work" conditions from real
	// errors without parsing stderr.
	if errors.Is(err, awsInternal.ErrNoDeployment) || errors.Is(err, rollback.ErrNoOngoingDeployment) {
		return exitNoDeployment
	}
	return 1
}

// isSilent returns whether silent mode is enabled
func isSilent() bool {
	return silent
}

// commandContext returns cmd.Context() with a context.Background() fallback.
// In production, Execute wires a signal-aware context via ExecuteContext, so
// every RunE receives a non-nil context. Tests that invoke RunE handlers
// directly may pass nil for cmd or skip SetContext, where cmd.Context() would
// panic or return nil — fall back to context.Background() so those tests
// don't have to know about the signal plumbing.
func commandContext(cmd *cobra.Command) context.Context {
	if cmd != nil {
		if ctx := cmd.Context(); ctx != nil {
			return ctx
		}
	}
	return context.Background()
}

// requireSingleConfig enforces that single-config commands receive exactly
// one `-c` value. It returns the single path when valid; otherwise it
// returns an error explaining that the command does not support
// multi-config (only run/diff/pull do).
func requireSingleConfig(cmdName string) (string, error) {
	switch len(configFiles) {
	case 0:
		// Cobra's default keeps the slice non-empty, but be defensive in
		// case a test or pre-run hook clears it.
		return defaultConfigFile, nil
	case 1:
		return configFiles[0], nil
	default:
		return "", fmt.Errorf("%s does not support multiple -c flags (got %d)", cmdName, len(configFiles))
	}
}

// resolveConfigTargets turns the -c values into a concrete, ordered,
// de-duplicated list of config paths for the multi-config commands
// (run/diff/pull/validate). Config files MUST be supplied via -c; glob
// patterns MUST be quoted so the shell hands them to apcdeploy intact
// (e.g. -c 'environments/*.yml'). An unquoted glob is expanded by the
// shell into positional arguments instead, so any positional argument is
// rejected with a hint to quote — this prevents the silent truncation that
// would otherwise occur (only the first shell-expanded path reaching -c).
func resolveConfigTargets(args []string) ([]string, error) {
	if len(args) > 0 {
		return nil, fmt.Errorf("unexpected arguments %q: pass config files via -c and quote glob patterns so the shell does not expand them, e.g. -c 'environments/*.yml'", args)
	}
	return expandConfigGlobs(configFiles)
}

// expandConfigGlobs expands each glob pattern in paths via filepath.Glob,
// preserving order and removing duplicates. Entries without a glob
// metacharacter pass through unchanged so a missing literal path still
// produces LoadAll's clear "file not found"; a pattern that matches nothing
// is an error.
func expandConfigGlobs(paths []string) ([]string, error) {
	out := make([]string, 0, len(paths))
	seen := make(map[string]bool)
	add := func(p string) {
		if !seen[p] {
			seen[p] = true
			out = append(out, p)
		}
	}
	for _, p := range paths {
		if !strings.ContainsAny(p, "*?[") {
			add(p)
			continue
		}
		matches, err := filepath.Glob(p)
		if err != nil {
			return nil, fmt.Errorf("invalid glob pattern %q: %w", p, err)
		}
		if len(matches) == 0 {
			return nil, fmt.Errorf("no files matched pattern %q", p)
		}
		for _, m := range matches {
			add(m)
		}
	}
	return out, nil
}

// maxDescriptionLength matches the AppConfig API limit on the Description
// field of CreateHostedConfigurationVersion / StartDeployment. Validating
// locally produces a clearer error than the AWS-side ValidationException.
const maxDescriptionLength = 1024

// defaultDescription is attached to AppConfig configuration versions and
// deployments when the user did not pass --description. It marks the change
// as originating from apcdeploy so it can be distinguished from manual edits
// in the AppConfig console.
const defaultDescription = "Deployed by apcdeploy"

// validateDescription enforces the AppConfig 1024-char limit on --description
// values before the AWS round-trip. AppConfig's limit is in Unicode characters,
// not bytes, so multibyte input (e.g. Japanese) is counted by rune.
// Empty values are allowed — the AWS wrappers omit the field entirely when
// description is "".
//
// Control characters (other than tab/newline/carriage return) are rejected
// up-front: the value flows into the AppConfig Description field, the
// AppConfig console UI, and any downstream log scraper, where embedded
// ANSI/control sequences would either corrupt the display or open a log
// injection vector. AWS may or may not sanitise — don't rely on it.
func validateDescription(s string) error {
	n := utf8.RuneCountInString(s)
	if n > maxDescriptionLength {
		return fmt.Errorf("--description exceeds maximum length of %d characters (got %d)", maxDescriptionLength, n)
	}
	for _, r := range s {
		if r < 0x20 && r != '\t' && r != '\n' && r != '\r' {
			return fmt.Errorf("--description contains invalid control character U+%04X", r)
		}
		if r == 0x7f {
			return fmt.Errorf("--description contains invalid control character U+%04X", r)
		}
	}
	return nil
}

// resolveDescription returns the description to attach to the configuration
// version / deployment. When the user did not pass --description, the default
// marker is used. An explicit --description "" keeps the empty value (opt-out)
// — Cobra's Changed() flag distinguishes "not set" from "set to empty".
func resolveDescription(cmd *cobra.Command, value string) string {
	if cmd.Flags().Changed("description") {
		return value
	}
	return defaultDescription
}
