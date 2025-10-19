package aws

import (
	"context"
	"fmt"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/appconfig"
	"github.com/aws/aws-sdk-go-v2/service/appconfig/types"
	"github.com/koh-sh/apcdeploy/internal/config"
)

// DeployedConfigInfo contains information about a deployed configuration
type DeployedConfigInfo struct {
	DeploymentNumber int32
	VersionNumber    int32
	Content          []byte
	ContentType      string
	State            types.DeploymentState
}

// GetLatestDeployedConfiguration retrieves the latest deployed configuration for a given profile.
// Returns nil if no deployment exists (not an error condition).
func GetLatestDeployedConfiguration(ctx context.Context, client *Client, appID, envID, profileID string) (*DeployedConfigInfo, error) {
	// Get latest deployment
	deployment, err := GetLatestDeployment(ctx, client, appID, envID, profileID)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest deployment: %w", err)
	}

	// No deployment found - return nil (not an error)
	if deployment == nil {
		return nil, nil
	}

	// Parse version number from deployment
	versionNum, err := strconv.ParseInt(deployment.ConfigurationVersion, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid version number %s: %w", deployment.ConfigurationVersion, err)
	}

	// Get configuration version content
	versionOutput, err := client.GetHostedConfigurationVersion(ctx, &appconfig.GetHostedConfigurationVersionInput{
		ApplicationId:          aws.String(appID),
		ConfigurationProfileId: aws.String(profileID),
		VersionNumber:          aws.Int32(int32(versionNum)),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get deployed configuration: %w", err)
	}

	// Extract content type
	contentType := config.ContentTypeJSON // Default fallback
	if versionOutput.ContentType != nil {
		contentType = *versionOutput.ContentType
	}

	return &DeployedConfigInfo{
		DeploymentNumber: deployment.DeploymentNumber,
		VersionNumber:    int32(versionNum),
		Content:          versionOutput.Content,
		ContentType:      contentType,
		State:            deployment.State,
	}, nil
}
