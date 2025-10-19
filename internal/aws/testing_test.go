package aws

import (
	"testing"
	"time"

	"github.com/koh-sh/apcdeploy/internal/aws/mock"
)

func TestNewTestClient(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                    string
		createClient            func() *Client
		expectedRegion          string
		expectedPollingInterval time.Duration
		checkAppConfig          bool
		checkAppConfigData      bool
	}{
		{
			name: "NewTestClient sets defaults",
			createClient: func() *Client {
				mockClient := &mock.MockAppConfigClient{}
				return NewTestClient(mockClient)
			},
			expectedRegion:          "us-east-1",
			expectedPollingInterval: 0, // NewTestClient does not set PollingInterval
			checkAppConfig:          true,
			checkAppConfigData:      false,
		},
		{
			name: "NewTestClientWithData sets both clients",
			createClient: func() *Client {
				mockAppConfig := &mock.MockAppConfigClient{}
				mockAppConfigData := &mock.MockAppConfigDataClient{}
				return NewTestClientWithData(mockAppConfig, mockAppConfigData)
			},
			expectedRegion:          "us-east-1",
			expectedPollingInterval: 0, // NewTestClientWithData does not set PollingInterval
			checkAppConfig:          true,
			checkAppConfigData:      true,
		},
		{
			name: "NewTestClientFull sets custom values",
			createClient: func() *Client {
				mockAppConfig := &mock.MockAppConfigClient{}
				mockAppConfigData := &mock.MockAppConfigDataClient{}
				testRegion := "ap-northeast-1"
				testInterval := 10 * time.Second
				return NewTestClientFull(mockAppConfig, mockAppConfigData, testRegion, testInterval)
			},
			expectedRegion:          "ap-northeast-1",
			expectedPollingInterval: 10 * time.Second,
			checkAppConfig:          true,
			checkAppConfigData:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			client := tt.createClient()

			if client == nil {
				t.Fatal("client creation returned nil")
			}

			if tt.checkAppConfig && client.appConfig == nil {
				t.Error("appConfig not set correctly")
			}

			if tt.checkAppConfigData && client.AppConfigData == nil {
				t.Error("AppConfigData not set correctly")
			}

			if client.Region != tt.expectedRegion {
				t.Errorf("Region = %s, want %s", client.Region, tt.expectedRegion)
			}

			if client.PollingInterval != tt.expectedPollingInterval {
				t.Errorf("PollingInterval = %v, want %v", client.PollingInterval, tt.expectedPollingInterval)
			}
		})
	}
}
