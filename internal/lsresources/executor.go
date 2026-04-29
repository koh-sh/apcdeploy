package lsresources

import (
	"context"
	"fmt"

	awsInternal "github.com/koh-sh/apcdeploy/internal/aws"
	"github.com/koh-sh/apcdeploy/internal/reporter"
)

// ClientFactory is a function type that creates an AWS client for a given region
type ClientFactory func(ctx context.Context, region string) (*awsInternal.Client, error)

// Executor handles the resource listing orchestration
type Executor struct {
	reporter      reporter.Reporter
	clientFactory ClientFactory
}

// NewExecutor creates a new list-resources executor
func NewExecutor(rep reporter.Reporter) *Executor {
	return &Executor{
		reporter:      rep,
		clientFactory: awsInternal.NewClient,
	}
}

// NewExecutorWithFactory creates a new list-resources executor with a custom client factory
// This is useful for testing with mock clients
func NewExecutorWithFactory(rep reporter.Reporter, factory ClientFactory) *Executor {
	return &Executor{
		reporter:      rep,
		clientFactory: factory,
	}
}

// Execute performs the complete resource listing workflow.
//
// In JSON mode the encoded payload is written to stdout via Reporter.Data;
// in normal mode the tree is rendered through Reporter.Header / Reporter.Table
// (stderr, suppressed under --silent).
func (e *Executor) Execute(ctx context.Context, opts *Options) error {
	client, err := e.clientFactory(ctx, opts.Region)
	if err != nil {
		return fmt.Errorf("failed to create AWS client: %w", err)
	}
	region := client.Region

	sp := e.reporter.Spin(fmt.Sprintf("Fetching AppConfig resources (%s)...", region))
	lister := New(client, region)
	tree, err := lister.ListResources(ctx)
	if err != nil {
		sp.Stop()
		return fmt.Errorf("failed to list resources: %w", err)
	}
	sp.Done(fmt.Sprintf("Found %d application(s) in %s", len(tree.Applications), region))

	if opts.JSON {
		payload, err := FormatJSON(tree, opts.ShowStrategies)
		if err != nil {
			return fmt.Errorf("failed to format JSON output: %w", err)
		}
		e.reporter.Data(payload)
		return nil
	}

	RenderHumanReadable(e.reporter, tree, opts.ShowStrategies)
	return nil
}
