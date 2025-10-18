package aws

import (
	"context"
	"fmt"
	"sort"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/appconfig"
)

// ConfigVersionInfo contains information about a configuration version
type ConfigVersionInfo struct {
	VersionNumber int32
	Content       []byte
	ContentType   string
}

// ConfigVersionFetcher handles fetching configuration versions
type ConfigVersionFetcher struct {
	client AppConfigAPI
}

// NewConfigVersionFetcher creates a new config version fetcher
func NewConfigVersionFetcher(client *Client) *ConfigVersionFetcher {
	return &ConfigVersionFetcher{
		client: client,
	}
}

// GetLatestVersion retrieves the latest configuration version
func (f *ConfigVersionFetcher) GetLatestVersion(ctx context.Context, appID, profileID string) (*ConfigVersionInfo, error) {
	// List all versions
	versions, err := f.client.ListAllHostedConfigurationVersions(ctx, appID, profileID)
	if err != nil {
		return nil, fmt.Errorf("failed to list configuration versions: %w", err)
	}

	if len(versions) == 0 {
		return nil, fmt.Errorf("no configuration versions found")
	}

	// Sort by version number to get the latest
	sort.Slice(versions, func(i, j int) bool {
		return versions[i].VersionNumber > versions[j].VersionNumber
	})

	latestItem := versions[0]

	// Get the full version content
	versionOutput, err := f.client.GetHostedConfigurationVersion(ctx, &appconfig.GetHostedConfigurationVersionInput{
		ApplicationId:          aws.String(appID),
		ConfigurationProfileId: aws.String(profileID),
		VersionNumber:          aws.Int32(latestItem.VersionNumber),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get configuration version: %w", err)
	}

	contentType := ""
	if versionOutput.ContentType != nil {
		contentType = *versionOutput.ContentType
	}

	return &ConfigVersionInfo{
		VersionNumber: latestItem.VersionNumber,
		Content:       versionOutput.Content,
		ContentType:   contentType,
	}, nil
}
