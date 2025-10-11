package get

import (
	"context"
	"fmt"

	"github.com/koh-sh/apcdeploy/internal/aws"
	"github.com/koh-sh/apcdeploy/internal/config"
)

// Getter handles configuration retrieval operations
type Getter struct {
	cfg       *config.Config
	awsClient *aws.Client
}

// New creates a new Getter instance
func New(ctx context.Context, cfg *config.Config) (*Getter, error) {
	// Initialize AWS client
	awsClient, err := aws.NewClient(ctx, cfg.Region)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize AWS client: %w", err)
	}

	return NewWithClient(cfg, awsClient), nil
}

// NewWithClient creates a new Getter instance with a provided AWS client
// This is useful for testing with mock clients
func NewWithClient(cfg *config.Config, awsClient *aws.Client) *Getter {
	return &Getter{
		cfg:       cfg,
		awsClient: awsClient,
	}
}

// ResolveResources resolves resource names to their AWS resource IDs.
// It resolves application, configuration profile, and environment only.
// Deployment strategy is intentionally not resolved as it's not needed for the get command.
//
// Returns:
//   - *aws.ResolvedResources: Struct containing resolved resource IDs
//   - error: Any error during resolution process
func (g *Getter) ResolveResources(ctx context.Context) (*aws.ResolvedResources, error) {
	// Create a resolver
	resolver := aws.NewResolver(g.awsClient)

	// Resolve application first as other resources depend on it
	appID, err := resolver.ResolveApplication(ctx, g.cfg.Application)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve application: %w", err)
	}

	// Resolve profile (needs appID)
	profile, err := resolver.ResolveConfigurationProfile(ctx, appID, g.cfg.ConfigurationProfile)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve configuration profile: %w", err)
	}

	// Resolve environment (needs appID)
	envID, err := resolver.ResolveEnvironment(ctx, appID, g.cfg.Environment)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve environment: %w", err)
	}

	// Return resolved resources (deployment strategy is empty as it's not needed)
	return &aws.ResolvedResources{
		ApplicationID:        appID,
		Profile:              profile,
		EnvironmentID:        envID,
		DeploymentStrategyID: "", // Not needed for get command
	}, nil
}

// GetConfiguration retrieves the latest configuration from AppConfig using the appconfigdata API.
// It uses StartConfigurationSession followed by GetLatestConfiguration to fetch the current
// deployed configuration data.
//
// Parameters:
//   - ctx: Context for the API calls
//   - resolved: Resolved AWS resource IDs (application, environment, profile)
//
// Returns:
//   - []byte: Raw configuration data from AppConfig
//   - error: Any error during the retrieval process
func (g *Getter) GetConfiguration(ctx context.Context, resolved *aws.ResolvedResources) ([]byte, error) {
	return g.awsClient.GetConfiguration(ctx, resolved.ApplicationID, resolved.EnvironmentID, resolved.Profile.ID)
}
