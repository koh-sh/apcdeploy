package aws

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/appconfig"
	"github.com/aws/aws-sdk-go-v2/service/appconfig/types"
)

// CheckOngoingDeployment checks if there is an ongoing deployment
func (c *Client) CheckOngoingDeployment(ctx context.Context, applicationID, environmentID string) (bool, *types.DeploymentSummary, error) {
	input := &appconfig.ListDeploymentsInput{
		ApplicationId: aws.String(applicationID),
		EnvironmentId: aws.String(environmentID),
	}

	output, err := c.AppConfig.ListDeployments(ctx, input)
	if err != nil {
		return false, nil, WrapAWSError(err, "failed to list deployments")
	}

	// Check for ongoing deployments (DEPLOYING or BAKING state)
	for _, deployment := range output.Items {
		if deployment.State == types.DeploymentStateDeploying ||
			deployment.State == types.DeploymentStateBaking {
			return true, &deployment, nil
		}
	}

	return false, nil, nil
}

// CreateHostedConfigurationVersion creates a new hosted configuration version
func (c *Client) CreateHostedConfigurationVersion(
	ctx context.Context,
	applicationID, profileID string,
	content []byte,
	contentType, description string,
) (int32, error) {
	input := &appconfig.CreateHostedConfigurationVersionInput{
		ApplicationId:          aws.String(applicationID),
		ConfigurationProfileId: aws.String(profileID),
		Content:                content,
		ContentType:            aws.String(contentType),
	}

	if description != "" {
		input.Description = aws.String(description)
	}

	output, err := c.AppConfig.CreateHostedConfigurationVersion(ctx, input)
	if err != nil {
		return 0, WrapAWSError(err, "failed to create hosted configuration version")
	}

	return output.VersionNumber, nil
}

// StartDeployment starts a new deployment
func (c *Client) StartDeployment(
	ctx context.Context,
	applicationID, environmentID, profileID, strategyID string,
	versionNumber int32,
	description string,
) (int32, error) {
	versionStr := fmt.Sprintf("%d", versionNumber)

	input := &appconfig.StartDeploymentInput{
		ApplicationId:          aws.String(applicationID),
		EnvironmentId:          aws.String(environmentID),
		ConfigurationProfileId: aws.String(profileID),
		ConfigurationVersion:   aws.String(versionStr),
		DeploymentStrategyId:   aws.String(strategyID),
	}

	if description != "" {
		input.Description = aws.String(description)
	}

	output, err := c.AppConfig.StartDeployment(ctx, input)
	if err != nil {
		return 0, WrapAWSError(err, "failed to start deployment")
	}

	return output.DeploymentNumber, nil
}

// WaitForDeployment waits for a deployment to complete
func (c *Client) WaitForDeployment(
	ctx context.Context,
	applicationID, environmentID string,
	deploymentNumber int32,
	timeout time.Duration,
) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Use configured polling interval, default to 5s if not set
	pollingInterval := c.PollingInterval
	if pollingInterval == 0 {
		pollingInterval = 5 * time.Second
	}
	ticker := time.NewTicker(pollingInterval)
	defer ticker.Stop()

	checkDeployment := func() (bool, error) {
		input := &appconfig.GetDeploymentInput{
			ApplicationId:    aws.String(applicationID),
			EnvironmentId:    aws.String(environmentID),
			DeploymentNumber: &deploymentNumber,
		}

		output, err := c.AppConfig.GetDeployment(ctx, input)
		if err != nil {
			return false, WrapAWSError(err, "failed to get deployment status")
		}

		switch output.State {
		case types.DeploymentStateComplete:
			return true, nil
		case types.DeploymentStateRolledBack:
			return false, fmt.Errorf("deployment was rolled back")
		case types.DeploymentStateDeploying, types.DeploymentStateBaking:
			return false, nil
		default:
			return false, fmt.Errorf("unexpected deployment state: %s", output.State)
		}
	}

	// Check immediately first
	if complete, err := checkDeployment(); err != nil || complete {
		return err
	}

	// Then check periodically
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("deployment timed out after %v", timeout)
		case <-ticker.C:
			if complete, err := checkDeployment(); err != nil || complete {
				return err
			}
		}
	}
}

// DeploymentInfo contains information about a deployment
type DeploymentInfo struct {
	DeploymentNumber     int32
	ConfigurationVersion string
	State                types.DeploymentState
	Description          string
}

// GetLatestDeployment retrieves the latest deployment for the specified configuration profile
func GetLatestDeployment(ctx context.Context, client *Client, applicationID, environmentID, profileID string) (*DeploymentInfo, error) {
	input := &appconfig.ListDeploymentsInput{
		ApplicationId: aws.String(applicationID),
		EnvironmentId: aws.String(environmentID),
	}

	output, err := client.AppConfig.ListDeployments(ctx, input)
	if err != nil {
		return nil, WrapAWSError(err, "failed to list deployments")
	}

	// Find the latest deployment for this configuration profile
	// We need to get full deployment details to access ConfigurationProfileId
	var latestDeployment *DeploymentInfo
	for i := range output.Items {
		summary := &output.Items[i]

		// Get full deployment details
		getInput := &appconfig.GetDeploymentInput{
			ApplicationId:    aws.String(applicationID),
			EnvironmentId:    aws.String(environmentID),
			DeploymentNumber: &summary.DeploymentNumber,
		}

		deployment, err := client.AppConfig.GetDeployment(ctx, getInput)
		if err != nil {
			continue // Skip this deployment if we can't get details
		}

		// Check if this is for the target configuration profile
		if aws.ToString(deployment.ConfigurationProfileId) == profileID {
			if latestDeployment == nil || summary.DeploymentNumber > latestDeployment.DeploymentNumber {
				latestDeployment = &DeploymentInfo{
					DeploymentNumber:     summary.DeploymentNumber,
					ConfigurationVersion: aws.ToString(deployment.ConfigurationVersion),
					State:                deployment.State,
					Description:          aws.ToString(deployment.Description),
				}
			}
		}
	}

	return latestDeployment, nil
}

// GetHostedConfigurationVersion retrieves the content of a specific hosted configuration version
func GetHostedConfigurationVersion(ctx context.Context, client *Client, applicationID, profileID, versionNumber string) ([]byte, error) {
	input := &appconfig.GetHostedConfigurationVersionInput{
		ApplicationId:          aws.String(applicationID),
		ConfigurationProfileId: aws.String(profileID),
		VersionNumber:          aws.Int32(0), // Will be replaced with parsed version
	}

	// Parse version number
	var version int32
	if _, err := fmt.Sscanf(versionNumber, "%d", &version); err != nil {
		return nil, fmt.Errorf("invalid version number: %s", versionNumber)
	}
	input.VersionNumber = aws.Int32(version)

	output, err := client.AppConfig.GetHostedConfigurationVersion(ctx, input)
	if err != nil {
		return nil, WrapAWSError(err, "failed to get hosted configuration version")
	}

	return output.Content, nil
}
