package aws

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/appconfig"
)

// Client wraps the AWS AppConfig client
type Client struct {
	AppConfig *appconfig.Client
	Region    string
}

// NewClient creates a new AWS client with the specified region
func NewClient(ctx context.Context, region string) (*Client, error) {
	// Determine region from parameter or environment
	finalRegion := region
	if finalRegion == "" {
		finalRegion = os.Getenv("AWS_REGION")
	}
	if finalRegion == "" {
		finalRegion = os.Getenv("AWS_DEFAULT_REGION")
	}
	if finalRegion == "" {
		return nil, fmt.Errorf("region must be specified either via --region flag or AWS_REGION/AWS_DEFAULT_REGION environment variable")
	}

	// Load AWS config
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(finalRegion),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create AppConfig client
	appconfigClient := appconfig.NewFromConfig(cfg)

	return &Client{
		AppConfig: appconfigClient,
		Region:    finalRegion,
	}, nil
}
