package cmd

import (
	"fmt"

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

	return rootCmd
}

// Execute runs the root command
func Execute() {
	rootCmd := NewRootCommand()
	if err := rootCmd.Execute(); err != nil {
		// Error is already printed by cobra
		return
	}
}
