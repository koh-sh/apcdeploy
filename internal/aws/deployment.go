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
