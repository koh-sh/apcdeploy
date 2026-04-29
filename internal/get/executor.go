package get

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/koh-sh/apcdeploy/internal/config"
	"github.com/koh-sh/apcdeploy/internal/prompt"
	"github.com/koh-sh/apcdeploy/internal/reporter"
)

// ErrUserDeclined is returned when the user declines to proceed with the operation
var ErrUserDeclined = errors.New("operation declined by user")

// Executor handles the configuration retrieval orchestration
type Executor struct {
	reporter      reporter.Reporter
	prompter      prompt.Prompter
	getterFactory func(context.Context, *config.Config) (*Getter, error)
}

// NewExecutor creates a new get executor
func NewExecutor(rep reporter.Reporter, prom prompt.Prompter) *Executor {
	return &Executor{
		reporter:      rep,
		prompter:      prom,
		getterFactory: New,
	}
}

// NewExecutorWithFactory creates a new get executor with a custom getter factory
// This is useful for testing with mock getters
func NewExecutorWithFactory(rep reporter.Reporter, prom prompt.Prompter, factory func(context.Context, *config.Config) (*Getter, error)) *Executor {
	return &Executor{
		reporter:      rep,
		prompter:      prom,
		getterFactory: factory,
	}
}

// Execute performs the complete configuration retrieval workflow
func (e *Executor) Execute(ctx context.Context, opts *Options) error {
	cfg, err := config.LoadConfig(opts.ConfigFile)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	getter, err := e.getterFactory(ctx, cfg)
	if err != nil {
		return fmt.Errorf("failed to create getter: %w", err)
	}

	sp := e.reporter.Spin("Resolving AWS resources...")
	resolved, err := getter.ResolveResources(ctx)
	if err != nil {
		sp.Stop()
		return fmt.Errorf("failed to resolve resources: %w", err)
	}
	sp.Done(fmt.Sprintf("Resolved resources: App=%s, Profile=%s, Env=%s",
		resolved.ApplicationID, resolved.Profile.ID, resolved.EnvironmentID))

	if !opts.SkipConfirmation {
		if err := e.prompter.CheckTTY(); err != nil {
			return fmt.Errorf("%w: use --yes to skip confirmation", err)
		}

		message := "This operation uses AWS AppConfig Data API (incurs charges). Proceed? (Y/Yes)"
		response, err := e.prompter.Input(message, "")
		if err != nil {
			return fmt.Errorf("failed to get user confirmation: %w", err)
		}

		normalized := strings.ToLower(strings.TrimSpace(response))
		if normalized != "y" && normalized != "yes" {
			return ErrUserDeclined
		}
	}

	sp = e.reporter.Spin("Fetching latest configuration...")
	configData, err := getter.GetConfiguration(ctx, resolved)
	if err != nil {
		sp.Stop()
		return fmt.Errorf("failed to get configuration for profile '%s' in environment '%s': %w",
			cfg.ConfigurationProfile, cfg.Environment, err)
	}
	sp.Done("Fetched configuration")

	e.reporter.Data(configData)

	return nil
}
