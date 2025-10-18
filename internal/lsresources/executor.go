package lsresources

import (
	"context"
	"fmt"
	"io"
	"os"

	awsInternal "github.com/koh-sh/apcdeploy/internal/aws"
	"github.com/koh-sh/apcdeploy/internal/reporter"
)

// ClientFactory is a function type that creates an AWS client for a given region
type ClientFactory func(ctx context.Context, region string) (*awsInternal.Client, error)

// Executor handles the resource listing orchestration
type Executor struct {
	reporter      reporter.ProgressReporter
	clientFactory ClientFactory
}

// NewExecutor creates a new list-resources executor
func NewExecutor(rep reporter.ProgressReporter) *Executor {
	return &Executor{
		reporter:      rep,
		clientFactory: awsInternal.NewClient,
	}
}

// NewExecutorWithFactory creates a new list-resources executor with a custom client factory
// This is useful for testing with mock clients
func NewExecutorWithFactory(rep reporter.ProgressReporter, factory ClientFactory) *Executor {
	return &Executor{
		reporter:      rep,
		clientFactory: factory,
	}
}

// Execute performs the complete resource listing workflow
func (e *Executor) Execute(ctx context.Context, opts *Options) error {
	return e.ExecuteWithWriter(ctx, opts, os.Stdout)
}

// ExecuteWithWriter performs the complete resource listing workflow with a custom writer
func (e *Executor) ExecuteWithWriter(ctx context.Context, opts *Options, w io.Writer) error {
	// Step 1: Create AWS client (with or without explicit region)
	e.reporter.Progress("Creating AWS client...")
	client, err := e.clientFactory(ctx, opts.Region)
	if err != nil {
		return fmt.Errorf("failed to create AWS client: %w", err)
	}
	e.reporter.Success("AWS client created")

	// Get the actual region used (either provided or SDK default)
	region := client.Region
	e.reporter.Progress(fmt.Sprintf("Using region: %s", region))

	// Step 2: List resources
	e.reporter.Progress("Fetching AppConfig resources...")
	lister := New(client, region)
	tree, err := lister.ListResources(ctx)
	if err != nil {
		return fmt.Errorf("failed to list resources: %w", err)
	}
	e.reporter.Success(fmt.Sprintf("Found %d application(s)", len(tree.Applications)))

	// Step 3: Format and output results
	e.reporter.Progress("Formatting output...")
	if opts.JSON {
		if err := FormatJSON(tree, w, opts.ShowStrategies); err != nil {
			return fmt.Errorf("failed to format JSON output: %w", err)
		}
	} else {
		if err := FormatHumanReadable(tree, w, opts.ShowStrategies); err != nil {
			return fmt.Errorf("failed to format output: %w", err)
		}
	}
	e.reporter.Success("Output generated successfully")

	return nil
}
