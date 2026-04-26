package edit

import (
	"context"
	"errors"
	"fmt"

	"github.com/koh-sh/apcdeploy/internal/prompt"
	"github.com/koh-sh/apcdeploy/internal/reporter"
)

// workflowFactory constructs a workflow. It is injectable for tests.
type workflowFactory func(context.Context, *Options, prompt.Prompter, reporter.Reporter) (*workflow, error)

// Executor drives the edit command.
type Executor struct {
	reporter        reporter.Reporter
	prompter        prompt.Prompter
	workflowFactory workflowFactory
}

// NewExecutor creates a new edit executor with the default workflow factory.
func NewExecutor(rep reporter.Reporter, prom prompt.Prompter) *Executor {
	return &Executor{
		reporter:        rep,
		prompter:        prom,
		workflowFactory: newWorkflow,
	}
}

// NewExecutorWithFactory creates a new edit executor with a custom workflow factory.
// This is useful for testing with mock dependencies.
func NewExecutorWithFactory(rep reporter.Reporter, prom prompt.Prompter, factory workflowFactory) *Executor {
	return &Executor{
		reporter:        rep,
		prompter:        prom,
		workflowFactory: factory,
	}
}

// Execute runs the edit command.
func (e *Executor) Execute(ctx context.Context, opts *Options) error {
	if opts.Timeout < 0 {
		return fmt.Errorf("timeout must be a non-negative value")
	}
	if opts.WaitDeploy && opts.WaitBake {
		return fmt.Errorf("--wait-deploy and --wait-bake cannot be used together")
	}

	wf, err := e.workflowFactory(ctx, opts, e.prompter, e.reporter)
	if err != nil {
		// Return TTY errors as-is without wrapping.
		if errors.Is(err, prompt.ErrNoTTY) {
			return err
		}
		return fmt.Errorf("failed to create edit workflow: %w", err)
	}

	return wf.Run(ctx, opts)
}
