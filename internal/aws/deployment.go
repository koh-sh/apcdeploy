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
	deployments, err := c.ListAllDeployments(ctx, applicationID, environmentID)
	if err != nil {
		return false, nil, wrapAWSError(err, "failed to list deployments")
	}

	// Check for ongoing deployments (DEPLOYING or BAKING state)
	for _, deployment := range deployments {
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

	output, err := c.appConfig.CreateHostedConfigurationVersion(ctx, input)
	if err != nil {
		return 0, wrapAWSError(err, "failed to create hosted configuration version")
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

	output, err := c.appConfig.StartDeployment(ctx, input)
	if err != nil {
		return 0, wrapAWSError(err, "failed to start deployment")
	}

	return output.DeploymentNumber, nil
}

// StopDeployment stops an in-progress deployment
func (c *Client) StopDeployment(
	ctx context.Context,
	applicationID, environmentID string,
	deploymentNumber int32,
) error {
	input := &appconfig.StopDeploymentInput{
		ApplicationId:    aws.String(applicationID),
		EnvironmentId:    aws.String(environmentID),
		DeploymentNumber: &deploymentNumber,
	}

	_, err := c.appConfig.StopDeployment(ctx, input)
	if err != nil {
		return wrapAWSError(err, "failed to stop deployment")
	}

	return nil
}

// extractRollbackReason extracts the rollback reason from deployment event log
func extractRollbackReason(eventLog []types.DeploymentEvent) string {
	// Look for rollback events in reverse order (most recent first)
	for i := len(eventLog) - 1; i >= 0; i-- {
		event := eventLog[i]
		// Check for rollback-related events
		if event.EventType == types.DeploymentEventTypeRollbackStarted ||
			event.EventType == types.DeploymentEventTypeRollbackCompleted {
			if event.Description != nil && *event.Description != "" {
				return *event.Description
			}
		}
	}
	return ""
}

// waitForDeploymentWithCondition is a generic wait function that polls deployment status
// until the provided checkComplete function returns true or an error occurs
func (c *Client) waitForDeploymentWithCondition(
	ctx context.Context,
	applicationID, environmentID string,
	deploymentNumber int32,
	timeout time.Duration,
	checkComplete func(types.DeploymentState) bool,
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

		output, err := c.appConfig.GetDeployment(ctx, input)
		if err != nil {
			return false, wrapAWSError(err, "failed to get deployment status")
		}

		// Check deployment state
		switch output.State {
		case types.DeploymentStateComplete, types.DeploymentStateBaking:
			// Check if deployment is complete based on the provided condition
			if checkComplete(output.State) {
				return true, nil
			}
			// Still in progress (e.g., BAKING when waiting for COMPLETE)
			return false, nil

		case types.DeploymentStateRolledBack:
			// Handle rollback state
			reason := extractRollbackReason(output.EventLog)
			if reason != "" {
				return false, fmt.Errorf("deployment was rolled back: %s", reason)
			}
			return false, fmt.Errorf("deployment was rolled back")

		case types.DeploymentStateDeploying:
			// Still deploying
			return false, nil

		default:
			// Unexpected state
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

// WaitForDeployment waits for a deployment to complete
func (c *Client) WaitForDeployment(
	ctx context.Context,
	applicationID, environmentID string,
	deploymentNumber int32,
	timeout time.Duration,
) error {
	// Use WaitForDeploymentPhase with waitForBaking=true to wait for full completion
	return c.WaitForDeploymentPhase(ctx, applicationID, environmentID, deploymentNumber, true, timeout)
}

// WaitForDeploymentPhase waits for a deployment to reach a specific phase
// If waitForBaking is false, it waits until the deployment enters BAKING state (deploy phase complete)
// If waitForBaking is true, it waits until the deployment reaches COMPLETE state (baking phase complete)
func (c *Client) WaitForDeploymentPhase(
	ctx context.Context,
	applicationID, environmentID string,
	deploymentNumber int32,
	waitForBaking bool,
	timeout time.Duration,
) error {
	return c.waitForDeploymentWithCondition(
		ctx,
		applicationID,
		environmentID,
		deploymentNumber,
		timeout,
		func(state types.DeploymentState) bool {
			// Always complete if deployment is COMPLETE
			if state == types.DeploymentStateComplete {
				return true
			}
			// If waiting for deploy phase only, BAKING means deploy is complete
			if state == types.DeploymentStateBaking && !waitForBaking {
				return true
			}
			return false
		},
	)
}

// DeploymentInfo contains information about a deployment
type DeploymentInfo struct {
	DeploymentNumber     int32
	ConfigurationVersion string
	State                types.DeploymentState
	Description          string
}

// GetLatestDeployment retrieves the latest deployment for the specified configuration profile
// This function skips ROLLED_BACK deployments and returns the last successful or in-progress deployment
func GetLatestDeployment(ctx context.Context, client *Client, applicationID, environmentID, profileID string) (*DeploymentInfo, error) {
	return getLatestDeploymentInternal(ctx, client, applicationID, environmentID, profileID, true)
}

// GetLatestDeploymentIncludingRollback retrieves the latest deployment for the specified configuration profile
// This function includes ROLLED_BACK deployments and returns the absolute latest deployment
func GetLatestDeploymentIncludingRollback(ctx context.Context, client *Client, applicationID, environmentID, profileID string) (*DeploymentInfo, error) {
	return getLatestDeploymentInternal(ctx, client, applicationID, environmentID, profileID, false)
}

// getLatestDeploymentInternal is the internal implementation for retrieving the latest deployment
func getLatestDeploymentInternal(ctx context.Context, client *Client, applicationID, environmentID, profileID string, skipRolledBack bool) (*DeploymentInfo, error) {
	deployments, err := client.ListAllDeployments(ctx, applicationID, environmentID)
	if err != nil {
		return nil, wrapAWSError(err, "failed to list deployments")
	}

	// Find the latest deployment for this configuration profile
	// We need to get full deployment details to access ConfigurationProfileId
	var latestDeployment *DeploymentInfo
	for i := range deployments {
		summary := &deployments[i]

		// Get full deployment details
		getInput := &appconfig.GetDeploymentInput{
			ApplicationId:    aws.String(applicationID),
			EnvironmentId:    aws.String(environmentID),
			DeploymentNumber: &summary.DeploymentNumber,
		}

		deployment, err := client.appConfig.GetDeployment(ctx, getInput)
		if err != nil {
			continue // Skip this deployment if we can't get details
		}

		// Check if this is for the target configuration profile
		if aws.ToString(deployment.ConfigurationProfileId) == profileID {
			// Skip ROLLED_BACK deployments if requested
			if skipRolledBack && deployment.State == types.DeploymentStateRolledBack {
				continue
			}

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

	output, err := client.appConfig.GetHostedConfigurationVersion(ctx, input)
	if err != nil {
		return nil, wrapAWSError(err, "failed to get hosted configuration version")
	}

	return output.Content, nil
}

// DeploymentDetails contains detailed information about a deployment
type DeploymentDetails struct {
	DeploymentNumber       int32
	ConfigurationProfileID string
	ConfigurationVersion   string
	DeploymentStrategyID   string
	DeploymentStrategyName string
	State                  types.DeploymentState
	Description            string
	EventLog               []types.DeploymentEvent
	StartedAt              *time.Time
	CompletedAt            *time.Time
	PercentageComplete     float32
	GrowthFactor           float32
	FinalBakeTimeInMinutes int32
}

// GetDeploymentDetails retrieves detailed information about a specific deployment
func GetDeploymentDetails(ctx context.Context, client *Client, applicationID, environmentID string, deploymentNumber int32) (*DeploymentDetails, error) {
	input := &appconfig.GetDeploymentInput{
		ApplicationId:    aws.String(applicationID),
		EnvironmentId:    aws.String(environmentID),
		DeploymentNumber: &deploymentNumber,
	}

	output, err := client.appConfig.GetDeployment(ctx, input)
	if err != nil {
		return nil, wrapAWSError(err, "failed to get deployment details")
	}

	var percentageComplete float32
	if output.PercentageComplete != nil {
		percentageComplete = *output.PercentageComplete
	}

	var growthFactor float32
	if output.GrowthFactor != nil {
		growthFactor = *output.GrowthFactor
	}

	details := &DeploymentDetails{
		DeploymentNumber:       output.DeploymentNumber,
		ConfigurationProfileID: aws.ToString(output.ConfigurationProfileId),
		ConfigurationVersion:   aws.ToString(output.ConfigurationVersion),
		DeploymentStrategyID:   aws.ToString(output.DeploymentStrategyId),
		State:                  output.State,
		Description:            aws.ToString(output.Description),
		EventLog:               output.EventLog,
		StartedAt:              output.StartedAt,
		CompletedAt:            output.CompletedAt,
		PercentageComplete:     percentageComplete,
		GrowthFactor:           growthFactor,
		FinalBakeTimeInMinutes: output.FinalBakeTimeInMinutes,
	}

	return details, nil
}
