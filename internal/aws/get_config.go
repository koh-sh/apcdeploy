package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/appconfigdata"
)

// GetConfiguration retrieves the latest configuration from AppConfig using AppConfigData API.
// This is the client-side API for retrieving configuration from an application perspective.
//
// Parameters:
//   - ctx: Context for the request
//   - applicationID: The application ID
//   - environmentID: The environment ID
//   - configurationProfileID: The configuration profile ID
//
// Returns:
//   - []byte: The configuration content
//   - error: Any error encountered during retrieval
func (c *Client) GetConfiguration(ctx context.Context, applicationID, environmentID, configurationProfileID string) ([]byte, error) {
	// Step 1: Start configuration session
	sessionInput := &appconfigdata.StartConfigurationSessionInput{
		ApplicationIdentifier:          aws.String(applicationID),
		EnvironmentIdentifier:          aws.String(environmentID),
		ConfigurationProfileIdentifier: aws.String(configurationProfileID),
	}

	sessionOutput, err := c.AppConfigData.StartConfigurationSession(ctx, sessionInput)
	if err != nil {
		return nil, fmt.Errorf("failed to start configuration session: %w", err)
	}

	// Step 2: Get latest configuration
	configInput := &appconfigdata.GetLatestConfigurationInput{
		ConfigurationToken: sessionOutput.InitialConfigurationToken,
	}

	configOutput, err := c.AppConfigData.GetLatestConfiguration(ctx, configInput)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest configuration: %w", err)
	}

	return configOutput.Configuration, nil
}
