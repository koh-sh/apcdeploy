package get

import (
	"context"
	"fmt"
	"os"

	"github.com/koh-sh/apcdeploy/internal/config"
	"github.com/koh-sh/apcdeploy/internal/reporter"
)

// Executor handles the configuration retrieval orchestration
type Executor struct {
	reporter      reporter.ProgressReporter
	getterFactory func(context.Context, *config.Config) (*Getter, error)
}

// NewExecutor creates a new get executor
func NewExecutor(rep reporter.ProgressReporter) *Executor {
	return &Executor{
		reporter:      rep,
		getterFactory: New,
	}
}

// NewExecutorWithFactory creates a new get executor with a custom getter factory
// This is useful for testing with mock getters
func NewExecutorWithFactory(rep reporter.ProgressReporter, factory func(context.Context, *config.Config) (*Getter, error)) *Executor {
	return &Executor{
		reporter:      rep,
		getterFactory: factory,
	}
}

// Execute performs the complete configuration retrieval workflow
func (e *Executor) Execute(ctx context.Context, opts *Options) error {
	// Step 1: Load configuration
	e.reporter.Progress("Loading configuration...")
	cfg, err := config.LoadConfig(opts.ConfigFile)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}
	e.reporter.Success("Configuration loaded")

	// Step 2: Create getter
	getter, err := e.getterFactory(ctx, cfg)
	if err != nil {
		return fmt.Errorf("failed to create getter: %w", err)
	}

	// Step 3: Resolve resources
	e.reporter.Progress("Resolving AWS resources...")
	resolved, err := getter.ResolveResources(ctx)
	if err != nil {
		return fmt.Errorf("failed to resolve resources: %w", err)
	}
	e.reporter.Success(fmt.Sprintf("Resolved resources: App=%s, Profile=%s, Env=%s",
		resolved.ApplicationID,
		resolved.Profile.ID,
		resolved.EnvironmentID,
	))

	// Step 4: Get latest configuration
	e.reporter.Progress("Fetching latest configuration...")
	configData, err := getter.GetConfiguration(ctx, resolved)
	if err != nil {
		return fmt.Errorf("failed to get configuration for profile '%s' in environment '%s': %w",
			cfg.ConfigurationProfile, cfg.Environment, err)
	}
	e.reporter.Success("Configuration retrieved successfully")

	// Step 5: Output configuration to stdout
	if _, err := os.Stdout.Write(configData); err != nil {
		return fmt.Errorf("failed to write configuration to stdout: %w", err)
	}

	return nil
}
