package aws

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/appconfig"
	"github.com/aws/aws-sdk-go-v2/service/appconfigdata"
	"github.com/koh-sh/apcdeploy/internal/config"
)

// Client wraps the AWS AppConfig client and implements AppConfigAPI interface.
// It holds the raw SDK client internally and delegates/enhances its methods.
type Client struct {
	// appConfig is the underlying AWS SDK client (private field implementing AppConfigSDKAPI)
	// This can be either *appconfig.Client in production or mock.MockAppConfigClient in tests
	appConfig       AppConfigSDKAPI
	AppConfigData   AppConfigDataAPI
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
		cfg, err = awsConfig.LoadDefaultConfig(ctx,
			awsConfig.WithRegion(region),
		)
	} else {
		// Otherwise, let AWS SDK resolve the default region from AWS config
		cfg, err = awsConfig.LoadDefaultConfig(ctx)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create AppConfig client
	appconfigClient := appconfig.NewFromConfig(cfg)
	appconfigdataClient := appconfigdata.NewFromConfig(cfg)

	return &Client{
		appConfig:       appconfigClient,
		AppConfigData:   appconfigdataClient,
		Region:          cfg.Region,
		PollingInterval: config.DefaultPollingInterval,
	}, nil
}
