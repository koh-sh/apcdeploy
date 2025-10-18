package aws

import "time"

// NewTestClient creates a new Client with a mock AppConfigSDKAPI for testing.
// This function is intended for use in tests only.
func NewTestClient(mockClient AppConfigSDKAPI) *Client {
	return &Client{
		appConfig: mockClient,
		Region:    "us-east-1", // Default test region
	}
}

// NewTestClientWithData creates a new Client with mock clients for both AppConfig and AppConfigData.
// This function is intended for use in tests only.
func NewTestClientWithData(mockAppConfig AppConfigSDKAPI, mockAppConfigData AppConfigDataAPI) *Client {
	return &Client{
		appConfig:     mockAppConfig,
		AppConfigData: mockAppConfigData,
		Region:        "us-east-1", // Default test region
	}
}

// NewTestClientFull creates a new Client with all fields for testing.
// This function is intended for use in tests only.
func NewTestClientFull(mockAppConfig AppConfigSDKAPI, mockAppConfigData AppConfigDataAPI, region string, pollingInterval time.Duration) *Client {
	return &Client{
		appConfig:       mockAppConfig,
		AppConfigData:   mockAppConfigData,
		Region:          region,
		PollingInterval: pollingInterval,
	}
}
