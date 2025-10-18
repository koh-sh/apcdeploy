package get

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/koh-sh/apcdeploy/internal/config"
	"github.com/koh-sh/apcdeploy/internal/prompt"
	"github.com/koh-sh/apcdeploy/internal/reporter"
)

// ErrUserDeclined is returned when the user declines to proceed with the operation
var ErrUserDeclined = errors.New("operation declined by user")

// Executor handles the configuration retrieval orchestration
type Executor struct {
	reporter      reporter.ProgressReporter
	prompter      prompt.Prompter
	getterFactory func(context.Context, *config.Config) (*Getter, error)
}

// NewExecutor creates a new get executor
func NewExecutor(rep reporter.ProgressReporter, prom prompt.Prompter) *Executor {
	return &Executor{
		reporter:      rep,
		prompter:      prom,
		getterFactory: New,
	}
}

// NewExecutorWithFactory creates a new get executor with a custom getter factory
// This is useful for testing with mock getters
func NewExecutorWithFactory(rep reporter.ProgressReporter, prom prompt.Prompter, factory func(context.Context, *config.Config) (*Getter, error)) *Executor {
	return &Executor{
		reporter:      rep,
		prompter:      prom,
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

	// Step 4: Prompt for confirmation unless skipped
	if !opts.SkipConfirmation {
		// Check TTY availability before interactive prompt
		if err := e.prompter.CheckTTY(); err != nil {
			return fmt.Errorf("%w: use --yes to skip confirmation", err)
		}

		message := "This operation uses AWS AppConfig Data API (incurs charges). Proceed? (Y/Yes)"
		response, err := e.prompter.Input(message, "")
		if err != nil {
			return fmt.Errorf("failed to get user confirmation: %w", err)
		}

		// Accept Y, y, Yes, yes
		normalized := strings.ToLower(strings.TrimSpace(response))
		if normalized != "y" && normalized != "yes" {
			return ErrUserDeclined
		}
	}

	// Step 5: Get latest configuration
	e.reporter.Progress("Fetching latest configuration...")
	configData, err := getter.GetConfiguration(ctx, resolved)
	if err != nil {
		return fmt.Errorf("failed to get configuration for profile '%s' in environment '%s': %w",
			cfg.ConfigurationProfile, cfg.Environment, err)
	}
	e.reporter.Success("Configuration retrieved successfully")

	// Step 6: Output configuration to stdout
	if _, err := os.Stdout.Write(configData); err != nil {
		return fmt.Errorf("failed to write configuration to stdout: %w", err)
	}

	return nil
}
