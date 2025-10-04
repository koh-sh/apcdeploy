package aws

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/appconfig"
	"github.com/koh-sh/apcdeploy/internal/aws/mock"
)

// Client wraps the AWS AppConfig client
type Client struct {
	AppConfig       mock.AppConfigAPI
	Region          string
	PollingInterval time.Duration // Interval for polling deployment status (default: 5s)
}

// NewClient creates a new AWS client with the specified region
func NewClient(ctx context.Context, region string) (*Client, error) {
	var cfg aws.Config
	var err error

	// Load AWS config
	if region != "" {
		// If region is explicitly provided, use it
		cfg, err = config.LoadDefaultConfig(ctx,
			config.WithRegion(region),
		)
	} else {
		// Otherwise, let AWS SDK resolve the default region from AWS config
		cfg, err = config.LoadDefaultConfig(ctx)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create AppConfig client
	appconfigClient := appconfig.NewFromConfig(cfg)

	return &Client{
		AppConfig:       appconfigClient,
		Region:          cfg.Region,
		PollingInterval: 5 * time.Second, // Default polling interval
	}, nil
}
