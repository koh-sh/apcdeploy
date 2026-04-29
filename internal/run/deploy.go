package run

import (
	"context"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/appconfig/types"
	"github.com/koh-sh/apcdeploy/internal/aws"
	"github.com/koh-sh/apcdeploy/internal/config"
	"github.com/koh-sh/apcdeploy/internal/reporter"
)

// Deployer handles deployment operations
type Deployer struct {
	cfg       *config.Config
	awsClient *aws.Client
}

// New creates a new Deployer instance
func New(ctx context.Context, cfg *config.Config) (*Deployer, error) {
	// Initialize AWS client
	awsClient, err := aws.NewClient(ctx, cfg.Region)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize AWS client: %w", err)
	}

	return NewWithClient(cfg, awsClient), nil
}

// NewWithClient creates a new Deployer instance with a provided AWS client
// This is useful for testing with mock clients
func NewWithClient(cfg *config.Config, awsClient *aws.Client) *Deployer {
	return &Deployer{
		cfg:       cfg,
		awsClient: awsClient,
	}
}

// loadConfiguration loads the configuration file and data file.
// It returns the parsed Config, the raw data file content, and any error encountered.
// The data file path in the returned Config is resolved to an absolute path.
//
// Parameters:
//   - configPath: Path to the apcdeploy.yml configuration file
//
// Returns:
//   - *config.Config: Parsed configuration with resolved paths
//   - []byte: Raw content of the data file
//   - error: Any error during loading or parsing
func loadConfiguration(configPath string) (*config.Config, []byte, error) {
	// Load the config file
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	// Read data file (path is already resolved by LoadConfig)
	dataContent, err := os.ReadFile(cfg.DataFile)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read data file %s: %w", cfg.DataFile, err)
	}

	return cfg, dataContent, nil
}

// ValidateLocalData validates the configuration data locally
func (d *Deployer) ValidateLocalData(data []byte, contentType string) error {
	return config.ValidateData(data, contentType)
}

// DetermineContentType determines the content type based on profile type and file extension
func (d *Deployer) DetermineContentType(profileType, dataPath string) (string, error) {
	// Feature Flags always use JSON
	if profileType == config.ProfileTypeFeatureFlags {
		return config.ContentTypeJSON, nil
	}

	// For Freeform, determine from file extension
	ext := strings.ToLower(filepath.Ext(dataPath))
	switch ext {
	case ".json":
		return config.ContentTypeJSON, nil
	case ".yaml", ".yml":
		return config.ContentTypeYAML, nil
	case ".txt":
		return config.ContentTypeText, nil
	default:
		// Default to text/plain for unknown extensions
		return config.ContentTypeText, nil
	}
}

// ResolveResources resolves all resource names to IDs
func (d *Deployer) ResolveResources(ctx context.Context) (*aws.ResolvedResources, error) {
	// Create a resolver
	resolver := aws.NewResolver(d.awsClient)

	// Resolve all resources
	resolved, err := resolver.ResolveAll(ctx,
		d.cfg.Application,
		d.cfg.ConfigurationProfile,
		d.cfg.Environment,
		d.cfg.DeploymentStrategy,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve resources: %w", err)
	}

	return resolved, nil
}

// CheckOngoingDeployment checks if there is an ongoing deployment
func (d *Deployer) CheckOngoingDeployment(ctx context.Context, resolved *aws.ResolvedResources) (bool, any, error) {
	return d.awsClient.CheckOngoingDeployment(ctx, resolved.ApplicationID, resolved.EnvironmentID)
}

// CreateVersion creates a new hosted configuration version
func (d *Deployer) CreateVersion(ctx context.Context, resolved *aws.ResolvedResources, content []byte, contentType string) (int32, error) {
	return d.awsClient.CreateHostedConfigurationVersion(ctx, resolved.ApplicationID, resolved.Profile.ID, content, contentType, "")
}

// StartDeployment starts a deployment
func (d *Deployer) StartDeployment(ctx context.Context, resolved *aws.ResolvedResources, versionNumber int32) (int32, error) {
	return d.awsClient.StartDeployment(ctx, resolved.ApplicationID, resolved.EnvironmentID, resolved.Profile.ID, resolved.DeploymentStrategyID, versionNumber, "")
}

// WaitForDeploymentPhase waits for a deployment to reach a specific phase.
// onTick is invoked on each polling tick; nil is allowed.
func (d *Deployer) WaitForDeploymentPhase(ctx context.Context, resolved *aws.ResolvedResources, deploymentNumber int32, waitForBaking bool, timeoutSeconds int, onTick aws.DeploymentTickFunc) error {
	timeout := time.Duration(timeoutSeconds) * time.Second
	return d.awsClient.WaitForDeploymentPhase(ctx, resolved.ApplicationID, resolved.EnvironmentID, deploymentNumber, waitForBaking, timeout, onTick)
}

// WaitForBakingComplete waits for an already-baking deployment to reach
// COMPLETE. onTick is invoked on each polling tick with bake progress; nil is
// allowed.
func (d *Deployer) WaitForBakingComplete(ctx context.Context, resolved *aws.ResolvedResources, deploymentNumber int32, timeoutSeconds int, onTick aws.BakeTickFunc) error {
	timeout := time.Duration(timeoutSeconds) * time.Second
	return d.awsClient.WaitForBakingComplete(ctx, resolved.ApplicationID, resolved.EnvironmentID, deploymentNumber, timeout, onTick)
}

// MakeDeployTick returns an aws.DeploymentTickFunc that drives a deploy
// progress bar. While DEPLOYING the bar reflects AppConfig's
// PercentageComplete; once BAKING (or COMPLETE) is observed the percentage
// pins at 100% and the label switches to bakingLabel.
//
// bakingLabel encodes how the deploy bar terminates relative to the
// surrounding wait orchestration:
//   - "Baking..."     when this bar is the only progress UI (--wait-deploy):
//     the wait loop exits as soon as BAKING is observed, so
//     the user briefly sees the bar pinned at 100% with the
//     bake label before pb.Done() prints the success line.
//   - "Deploying..."  when a separate spinner takes over for bake
//     (--wait-bake): the deploy bar is finalized and a
//     dedicated bake spinner is started after pb.Done().
//
// The "(~N min left)" suffix is wall-clock based (totalDuration minus
// locally tracked elapsed time) so non-linear growth strategies
// (EXPONENTIAL) report honest remaining time. The bar percentage is left as
// AppConfig's PercentageComplete so the rollout-progress reading is not
// fabricated.
//
// Currently shared between `run` and `edit` only. If a third caller appears,
// move this to a neutral location (e.g. internal/aws or a new internal
// package) so feature packages stop reaching across to `run` for the helper.
func MakeDeployTick(pb reporter.ProgressBar, bakingLabel string) aws.DeploymentTickFunc {
	waitStart := time.Now()
	return func(state types.DeploymentState, percent float64, totalDuration time.Duration) {
		if state == types.DeploymentStateBaking || state == types.DeploymentStateComplete {
			pb.Update(100, bakingLabel)
			return
		}
		elapsed := time.Since(waitStart)
		pb.Update(percent, "Deploying..."+remainingFromElapsedSuffix(elapsed, totalDuration))
	}
}

// MakeBakeTick returns an aws.BakeTickFunc that updates a Spinner's
// label with the current "(~N min left)" countdown each tick. Bake is a
// monitoring wait rather than a quantified rollout, so the UX is a spinner
// (no bar): the % "filling" of a progress bar would falsely suggest that
// rollout work is still happening, when in fact the deployment is just
// waiting out FinalBakeTimeInMinutes.
func MakeBakeTick(s reporter.Spinner) aws.BakeTickFunc {
	return func(elapsed, total time.Duration) {
		s.Update("Baking..." + remainingFromElapsedSuffix(elapsed, total))
	}
}

// remainingFromElapsedSuffix renders a "(~N min left)" suffix from total
// minus locally observed elapsed time. Falls back to "(<1 min left)" when
// total is zero (e.g. AppConfig.AllAtOnce), when elapsed has already run
// past total, or when the remaining is below one minute. The function
// always returns a non-empty string so the bar always carries a time hint,
// and the threshold is on the duration itself (not math.Ceil) so 30 s and
// 59 s render honestly as "<1 min left" instead of being rounded up to
// "~1 min left".
func remainingFromElapsedSuffix(elapsed, total time.Duration) string {
	remaining := total - elapsed
	if total <= 0 || remaining < time.Minute {
		return " (<1 min left)"
	}
	return fmt.Sprintf(" (~%d min left)", int(math.Ceil(remaining.Minutes())))
}

// HasConfigurationChanges checks if the local configuration differs from the deployed version
func (d *Deployer) HasConfigurationChanges(ctx context.Context, resolved *aws.ResolvedResources, localContent []byte, fileName, contentType string) (bool, error) {
	// Get the latest deployment to find the deployed version number
	deployment, err := aws.GetLatestDeployment(ctx, d.awsClient, resolved.ApplicationID, resolved.EnvironmentID, resolved.Profile.ID)
	if err != nil {
		return false, fmt.Errorf("failed to get latest deployment: %w", err)
	}

	// If no deployment exists, this is the first deployment - has changes
	if deployment == nil {
		return true, nil
	}

	// Get the deployed configuration version content
	remoteContent, err := aws.GetHostedConfigurationVersion(ctx, d.awsClient, resolved.ApplicationID, resolved.Profile.ID, deployment.ConfigurationVersion)
	if err != nil {
		return false, fmt.Errorf("failed to get deployed configuration: %w", err)
	}

	return config.HasContentChanged(remoteContent, localContent, filepath.Ext(fileName), resolved.Profile.Type)
}
