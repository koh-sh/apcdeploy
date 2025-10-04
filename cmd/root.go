package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	// Version information
	Version   = "dev"
	GitCommit = "none"
	BuildDate = "unknown"

	// Global flags
	configFile string
	region     string
)

// NewRootCommand creates and returns the root command
func NewRootCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "apcdeploy",
		Short: "AWS AppConfig deployment tool",
		Long: `apcdeploy is a CLI tool for managing AWS AppConfig deployments.
It provides commands to initialize, deploy, diff, and check the status of configurations.`,
		Version: fmt.Sprintf("%s (commit: %s, built: %s)", Version, GitCommit, BuildDate),
	}

	// Global flags
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "apcdeploy.yml", "config file path")
	rootCmd.PersistentFlags().StringVar(&region, "region", "", "AWS region (overrides config file)")

	// Add subcommands
	rootCmd.AddCommand(InitCommand())
	rootCmd.AddCommand(DeployCommand())
	rootCmd.AddCommand(DiffCommand())
	rootCmd.AddCommand(StatusCommand())

	return rootCmd
}

// Execute runs the root command
func Execute() {
	rootCmd := NewRootCommand()

	// Enable custom error formatting
	rootCmd.SilenceErrors = true

	if err := rootCmd.Execute(); err != nil {
		// Print error with blank lines before and after for better visibility
		fmt.Fprintln(os.Stderr)
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		fmt.Fprintln(os.Stderr)
		os.Exit(1)
	}
}
