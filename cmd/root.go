package cmd

import (
	"errors"
	"fmt"
	"os"
	"unicode/utf8"

	awsInternal "github.com/koh-sh/apcdeploy/internal/aws"
	"github.com/koh-sh/apcdeploy/internal/cli"
	apcerrors "github.com/koh-sh/apcdeploy/internal/errors"
	"github.com/spf13/cobra"
)

// Exit codes used by the CLI. Anything other than 0/1 is considered a
// distinguishable condition that scripts can branch on.
const (
	exitNoDeployment = 2
)

var (
	// Version information
	version string
	commit  string
	date    string

	// Global flags
	configFile string
	silent     bool
)

// NewRootCommand creates and returns the root command
func NewRootCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "apcdeploy",
		Short: "AWS AppConfig deployment tool",
		Long: `apcdeploy is a CLI tool for managing AWS AppConfig deployments.
It provides commands to initialize, deploy, diff, and check the status of configurations.`,
		Version: fmt.Sprintf("%s (Built on %s from Git SHA %s)", version, date, commit),
	}

	// Global flags
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "apcdeploy.yml", "config file path")
	rootCmd.PersistentFlags().BoolVarP(&silent, "silent", "s", false, "suppress verbose output, show only essential information")

	// Add subcommands
	rootCmd.AddCommand(InitCommand())
	rootCmd.AddCommand(RunCommand())
	rootCmd.AddCommand(DiffCommand())
	rootCmd.AddCommand(StatusCommand())
	rootCmd.AddCommand(GetCommand())
	rootCmd.AddCommand(PullCommand())
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

// Execute runs the root command
func Execute() {
	rootCmd := NewRootCommand()

	// Enable custom error formatting
	rootCmd.SilenceErrors = true

	if err := rootCmd.Execute(); err != nil {
		// Funnel the top-level error through the Reporter so the styled "✗"
		// prefix is consistent with the rest of stderr output. Both real and
		// silent reporters always emit Error.
		rep := cli.GetReporter(silent)
		rep.Error(err.Error())
		// Append a Resolution: <hint> line when the underlying AWS error code
		// has a documented remediation (output.md §8.3 / internal/errors).
		// Emitted via Warn (⚠) instead of Error (✗) so the visual hierarchy is
		// "what failed" first, "how to recover" second; both lines reach
		// stderr even under --silent because Warn-via-the-real-reporter is
		// suppressed by the silent variant — so the hint shows in interactive
		// runs but not in piped/automation output.
		if hint := apcerrors.Resolution(err); hint != "" {
			rep.Warn("Resolution: " + hint)
		}
		// Exit 2 when the failure is "no prior deployment" so scripts can
		// distinguish that condition (e.g. first-time setup) from real errors.
		if errors.Is(err, awsInternal.ErrNoDeployment) {
			os.Exit(exitNoDeployment)
		}
		os.Exit(1)
	}
}

// isSilent returns whether silent mode is enabled
func isSilent() bool {
	return silent
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
func validateDescription(s string) error {
	n := utf8.RuneCountInString(s)
	if n > maxDescriptionLength {
		return fmt.Errorf("--description exceeds maximum length of %d characters (got %d)", maxDescriptionLength, n)
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
