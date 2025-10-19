package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
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
	rootCmd.AddCommand(LsResourcesCommand())
	rootCmd.AddCommand(ContextCommand())

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
		// Print error message (only when not in silent mode or always show errors)
		if silent {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		} else {
			// Print error with blank lines before and after for better visibility
			fmt.Fprintln(os.Stderr)
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			fmt.Fprintln(os.Stderr)
		}
		os.Exit(1)
	}
}

// isSilent returns whether silent mode is enabled
func isSilent() bool {
	return silent
}
