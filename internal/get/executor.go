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

// Execute performs the complete configuration retrieval workflow.
//
// Output shape (docs/design/output.md §7.5):
//   - interactive: identifier shown as Header, cost notice via Warn, prompt;
//     the configuration body lands on stdout after the user accepts.
//   - --yes (TTY): a single Targets row finalised as ✓ fetched, plus the
//     configuration body on stdout.
//   - --silent --yes: stdout-only — the user has explicitly opted out of
//     stderr noise (output.md §7.5 (c)).
//
// Resource resolution happens before the cost prompt because List APIs do
// not incur per-call charges and it gives the user a clearer error path
// when names don't match (output.md §7.5 (a) shows the prompt first, but
// the resolve step is invisible to the user when it succeeds).
func (e *Executor) Execute(ctx context.Context, opts *Options) error {
	cfg, err := config.LoadConfig(opts.ConfigFile)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	getter, err := e.getterFactory(ctx, cfg)
	if err != nil {
		return fmt.Errorf("failed to create getter: %w", err)
	}

	resolved, err := getter.ResolveResources(ctx)
	if err != nil {
		return fmt.Errorf("failed to resolve resources: %w", err)
	}

	id := config.Identifier(getter.Region(), cfg)

	if !opts.SkipConfirmation {
		if err := e.prompter.CheckTTY(); err != nil {
			return fmt.Errorf("%w: use --yes to skip confirmation", err)
		}

		e.reporter.Header(id)
		e.reporter.Warn("Note: fetching the latest deployed configuration uses the AppConfig Data API which incurs cost per call.")

		response, err := e.prompter.Input("Continue? (y/N)", "")
		if err != nil {
			return fmt.Errorf("failed to get user confirmation: %w", err)
		}
		normalized := strings.ToLower(strings.TrimSpace(response))
		if normalized != "y" && normalized != "yes" {
			return ErrUserDeclined
		}

		// Fetch silently after acceptance: the user already sees the
		// identifier in the Header above; an extra Targets row would just
		// duplicate it (output.md §7.5 (a) — the body lands on stdout with
		// no completion line on stderr).
		configData, err := getter.GetConfiguration(ctx, resolved)
		if err != nil {
			return fmt.Errorf("failed to get configuration for profile %q in environment %q: %w",
				cfg.ConfigurationProfile, cfg.Environment, err)
		}
		e.reporter.Data(configData)
		return nil
	}

	// Non-interactive flow (--yes, possibly with --silent): a single Targets
	// row shows the fetch lifecycle, then the body lands on stdout.
	tg := e.reporter.Targets([]string{id})
	defer tg.Close()
	tg.SetPhase(id, "fetching", "")
	configData, err := getter.GetConfiguration(ctx, resolved)
	if err != nil {
		tg.Fail(id, err)
		return fmt.Errorf("failed to get configuration for profile %q in environment %q: %w",
			cfg.ConfigurationProfile, cfg.Environment, err)
	}
	e.reporter.Data(configData)
	tg.Done(id, "fetched")
	return nil
}
