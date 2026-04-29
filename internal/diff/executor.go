package diff

import (
	"context"
	"errors"
	"fmt"

	"github.com/koh-sh/apcdeploy/internal/aws"
	"github.com/koh-sh/apcdeploy/internal/config"
	"github.com/koh-sh/apcdeploy/internal/reporter"
)

// ErrDiffFound is returned when differences are found and ExitNonzero is true
var ErrDiffFound = errors.New("differences found")

// Executor handles the diff operation orchestration
type Executor struct {
	reporter      reporter.Reporter
	clientFactory func(context.Context, string) (*aws.Client, error)
}

// NewExecutor creates a new diff executor
func NewExecutor(rep reporter.Reporter) *Executor {
	return &Executor{
		reporter:      rep,
		clientFactory: aws.NewClient,
	}
}

// NewExecutorWithFactory creates a new diff executor with a custom client factory
// This is useful for testing with mock clients
func NewExecutorWithFactory(rep reporter.Reporter, factory func(context.Context, string) (*aws.Client, error)) *Executor {
	return &Executor{
		reporter:      rep,
		clientFactory: factory,
	}
}

// Execute performs the complete diff workflow
func (e *Executor) Execute(ctx context.Context, opts *Options) error {
	cfg, err := config.LoadConfig(opts.ConfigFile)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	awsClient, err := e.clientFactory(ctx, cfg.Region)
	if err != nil {
		return fmt.Errorf("failed to initialize AWS client: %w", err)
	}

	localData, err := config.LoadDataFile(cfg.DataFile)
	if err != nil {
		return fmt.Errorf("failed to load local configuration file: %w", err)
	}

	// Resolve + GetLatestDeployment + GetHostedConfigurationVersion are folded
	// into one user-facing phase ("load deployment data") because the user
	// thinks of them as a single step — fetching the remote side of the diff.
	sp := e.reporter.Spin("Loading deployment data...")
	resolver := aws.NewResolver(awsClient)
	resources, err := resolver.ResolveAll(ctx, cfg.Application, cfg.ConfigurationProfile, cfg.Environment, cfg.DeploymentStrategy)
	if err != nil {
		sp.Stop()
		return fmt.Errorf("failed to resolve resources: %w", err)
	}

	deployment, err := aws.GetLatestDeployment(ctx, awsClient, resources.ApplicationID, resources.EnvironmentID, resources.Profile.ID)
	if err != nil {
		sp.Stop()
		return fmt.Errorf("failed to get latest deployment: %w", err)
	}

	// No prior deployment: emit the local data as the stdout payload (acts as
	// the "right side" of the would-be diff) and surface the next-step hint
	// via the Reporter so silent mode suppresses the human-facing parts
	// automatically.
	if deployment == nil {
		sp.Done("No prior deployment — this will be the initial deployment")
		e.reporter.Header("Local configuration")
		e.reporter.Data(localData)
		if len(localData) > 0 && localData[len(localData)-1] != '\n' {
			e.reporter.Data([]byte("\n"))
		}
		e.reporter.Info("Run 'apcdeploy run' to create the first deployment.")
		return nil
	}

	remoteData, err := aws.GetHostedConfigurationVersion(ctx, awsClient, resources.ApplicationID, resources.Profile.ID, deployment.ConfigurationVersion)
	if err != nil {
		sp.Stop()
		return fmt.Errorf("failed to get deployed configuration: %w", err)
	}
	sp.Done(fmt.Sprintf("Loaded deployment data (deployment #%d, version %s)", deployment.DeploymentNumber, deployment.ConfigurationVersion))

	diffResult, err := calculate(string(remoteData), string(localData), cfg.DataFile, resources.Profile.Type)
	if err != nil {
		return fmt.Errorf("failed to calculate diff: %w", err)
	}

	display(e.reporter, diffResult, cfg, resources, deployment)

	if opts.ExitNonzero && diffResult.HasChanges {
		return ErrDiffFound
	}

	return nil
}
