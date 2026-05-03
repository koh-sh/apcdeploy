package status

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/appconfig/types"
	"github.com/koh-sh/apcdeploy/internal/aws"
	"github.com/koh-sh/apcdeploy/internal/config"
	"github.com/koh-sh/apcdeploy/internal/display"
	"github.com/koh-sh/apcdeploy/internal/reporter"
)

// Executor handles the status operation orchestration
type Executor struct {
	reporter      reporter.Reporter
	clientFactory func(context.Context, string) (*aws.Client, error)
}

// NewExecutor creates a new status executor
func NewExecutor(rep reporter.Reporter) *Executor {
	return &Executor{
		reporter:      rep,
		clientFactory: aws.NewClient,
	}
}

// NewExecutorWithFactory creates a new status executor with a custom client factory
// This is useful for testing with mock clients
func NewExecutorWithFactory(rep reporter.Reporter, factory func(context.Context, string) (*aws.Client, error)) *Executor {
	return &Executor{
		reporter:      rep,
		clientFactory: factory,
	}
}

// Execute performs the status check workflow.
//
// Output shape (docs/design/output.md §7.4):
//   - found: ✓ <STATE> — v<N> [(deployed/started X ago)] on the Targets row,
//     followed by display.DeploymentStatus's Header + Table on stderr and the
//     state name on stdout.
//   - no deployment: ⊘ no deployment on the Targets row, NONE on stdout, a
//     short Box of next-step guidance on stderr, and aws.ErrNoDeployment as
//     the returned error so cmd/root.go exits 2.
func (e *Executor) Execute(ctx context.Context, opts *Options) error {
	cfg, err := config.LoadConfig(opts.ConfigFile)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	awsClient, err := e.clientFactory(ctx, cfg.Region)
	if err != nil {
		return fmt.Errorf("failed to initialize AWS client: %w", err)
	}

	id := config.Identifier(awsClient.Region, cfg)
	tg := e.reporter.Targets([]string{id})
	defer tg.Close()
	detail := ""
	if opts.DeploymentID != "" {
		detail = "(deployment #" + opts.DeploymentID + ")"
	}
	tg.SetPhase(id, "fetching", detail)

	resolver := aws.NewResolver(awsClient)
	resources, err := resolver.ResolveAll(ctx, cfg.Application, cfg.ConfigurationProfile, cfg.Environment, cfg.DeploymentStrategy)
	if err != nil {
		tg.Fail(id, err)
		return fmt.Errorf("failed to resolve resources: %w", err)
	}

	var deploymentInfo *aws.DeploymentDetails
	if opts.DeploymentID != "" {
		deploymentInfo, err = e.getDeploymentByID(ctx, awsClient, resources, opts.DeploymentID)
	} else {
		deploymentInfo, err = e.getLatestDeployment(ctx, awsClient, resources)
	}
	if err != nil {
		tg.Fail(id, err)
		return fmt.Errorf("failed to get deployment: %w", err)
	}

	if deploymentInfo == nil {
		tg.Skip(id, "no deployment")
		// stdout payload is fixed at "NONE\n" so scripts can branch on it
		// (output.md §7.4 (b)). Always emitted, even under --silent.
		e.reporter.Data([]byte("NONE\n"))
		e.reporter.Box("", []string{
			"No deployment has been created yet for this profile/environment.",
			"Run 'apcdeploy run -c " + opts.ConfigFile + "' to create the initial deployment.",
		})
		return fmt.Errorf("status: %w", aws.ErrNoDeployment)
	}

	tg.Done(id, summarizeDeployment(deploymentInfo))
	// Two output systems intentionally coexist on the success path
	// (output.md §11 Q-2 — to be revisited when status grows multi-target
	// support). The Targets row is the at-a-glance summary
	// ("✓ COMPLETE — v42 (2h ago)"); display.DeploymentStatus follows with
	// the structured Header + Table that surfaces the full deployment
	// metadata (Deployment Number, strategy, started/completed timestamps).
	// In TTY mode the Targets renderer has already finalised by the time
	// the table prints, so the two views stack cleanly without competing
	// for the cursor.
	display.DeploymentStatus(e.reporter, deploymentInfo, cfg, resources)
	return nil
}

// summarizeDeployment renders the post-icon Targets summary for a deployment.
// Format: "<STATE> [<percent>%] — v<ConfigVersion>[ (<verb> <relative-time>)]"
// per docs/design/output.md §7.4 (a)/(a').
func summarizeDeployment(d *aws.DeploymentDetails) string {
	state := string(d.State)
	summary := state
	if (d.State == types.DeploymentStateDeploying || d.State == types.DeploymentStateBaking) && d.PercentageComplete > 0 {
		summary = fmt.Sprintf("%s %.0f%%", state, d.PercentageComplete)
	}
	if d.ConfigurationVersion != "" {
		summary += " — v" + d.ConfigurationVersion
	}
	switch d.State {
	case types.DeploymentStateComplete:
		if d.CompletedAt != nil {
			summary += " (deployed " + relativeTime(*d.CompletedAt) + ")"
		}
	case types.DeploymentStateDeploying, types.DeploymentStateBaking:
		if d.StartedAt != nil {
			summary += " (started " + relativeTime(*d.StartedAt) + ")"
		}
	}
	return summary
}

// relativeTime renders a time.Time as a coarse "Xs/Xm/Xh/Xd ago" label so
// the Targets summary stays compact. Future timestamps (clock skew) collapse
// to "just now".
func relativeTime(t time.Time) string {
	d := time.Since(t)
	if d < 0 {
		return "just now"
	}
	switch {
	case d < time.Minute:
		return strconv.Itoa(int(d.Seconds())) + "s ago"
	case d < time.Hour:
		return strconv.Itoa(int(d.Minutes())) + "m ago"
	case d < 24*time.Hour:
		return strconv.Itoa(int(d.Hours())) + "h ago"
	default:
		return strconv.Itoa(int(d.Hours()/24)) + "d ago"
	}
}

// getDeploymentByID retrieves a specific deployment by its ID
func (e *Executor) getDeploymentByID(ctx context.Context, client *aws.Client, resources *aws.ResolvedResources, deploymentID string) (*aws.DeploymentDetails, error) {
	deploymentNumber, err := strconv.ParseInt(deploymentID, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid deployment ID: %s", deploymentID)
	}

	deployment, err := aws.GetDeploymentDetails(ctx, client, resources.ApplicationID, resources.EnvironmentID, int32(deploymentNumber))
	if err != nil {
		return nil, err
	}

	if deployment.ConfigurationProfileID != resources.Profile.ID {
		return nil, fmt.Errorf("deployment #%d is not for configuration profile %s", deploymentNumber, resources.Profile.Name)
	}

	resolver := aws.NewResolver(client)
	strategyName, err := resolver.ResolveDeploymentStrategyIDToName(ctx, deployment.DeploymentStrategyID)
	if err != nil {
		strategyName = deployment.DeploymentStrategyID
	}
	deployment.DeploymentStrategyName = strategyName

	return deployment, nil
}

// getLatestDeployment retrieves the latest deployment for the configuration profile
// This includes ROLLED_BACK deployments for status command
func (e *Executor) getLatestDeployment(ctx context.Context, client *aws.Client, resources *aws.ResolvedResources) (*aws.DeploymentDetails, error) {
	deployment, err := aws.GetLatestDeploymentIncludingRollback(ctx, client, resources.ApplicationID, resources.EnvironmentID, resources.Profile.ID)
	if err != nil {
		return nil, err
	}
	if deployment == nil {
		return nil, nil
	}

	details, err := aws.GetDeploymentDetails(ctx, client, resources.ApplicationID, resources.EnvironmentID, deployment.DeploymentNumber)
	if err != nil {
		return nil, err
	}

	resolver := aws.NewResolver(client)
	strategyName, err := resolver.ResolveDeploymentStrategyIDToName(ctx, details.DeploymentStrategyID)
	if err != nil {
		strategyName = details.DeploymentStrategyID
	}
	details.DeploymentStrategyName = strategyName

	return details, nil
}
