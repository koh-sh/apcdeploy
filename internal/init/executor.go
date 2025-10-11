package init

import (
	"context"
	"fmt"

	"github.com/koh-sh/apcdeploy/internal/prompt"
	"github.com/koh-sh/apcdeploy/internal/reporter"
)

// Executor handles the init command orchestration
type Executor struct {
	reporter           reporter.ProgressReporter
	prompter           prompt.Prompter
	initializerFactory func(context.Context, *Options, prompt.Prompter, reporter.ProgressReporter) (*InitWorkflow, error)
}

// NewExecutor creates a new init executor
func NewExecutor(rep reporter.ProgressReporter, prom prompt.Prompter) *Executor {
	return &Executor{
		reporter:           rep,
		prompter:           prom,
		initializerFactory: NewInitWorkflow,
	}
}

// NewExecutorWithFactory creates a new init executor with a custom initializer factory
// This is useful for testing with mock dependencies
func NewExecutorWithFactory(rep reporter.ProgressReporter, prom prompt.Prompter, factory func(context.Context, *Options, prompt.Prompter, reporter.ProgressReporter) (*InitWorkflow, error)) *Executor {
	return &Executor{
		reporter:           rep,
		prompter:           prom,
		initializerFactory: factory,
	}
}

// Execute performs the complete initialization workflow
func (e *Executor) Execute(ctx context.Context, opts *Options) error {
	// Create workflow with all dependencies
	workflow, err := e.initializerFactory(ctx, opts, e.prompter, e.reporter)
	if err != nil {
		return fmt.Errorf("failed to create init workflow: %w", err)
	}

	// Run initialization
	if err := workflow.Run(ctx, opts); err != nil {
		return err
	}

	return nil
}
