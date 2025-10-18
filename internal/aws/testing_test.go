package aws

import (
	"testing"
	"time"

	"github.com/koh-sh/apcdeploy/internal/aws/mock"
)

func TestNewTestClient(t *testing.T) {
	mockClient := &mock.MockAppConfigClient{}
	client := NewTestClient(mockClient)

	if client == nil {
		t.Fatal("NewTestClient() returned nil")
	}

	if client.appConfig != mockClient {
		t.Error("NewTestClient() did not set appConfig correctly")
	}

	if client.Region != "us-east-1" {
		t.Errorf("NewTestClient() Region = %s, want us-east-1", client.Region)
	}
}

func TestNewTestClientWithData(t *testing.T) {
	mockAppConfig := &mock.MockAppConfigClient{}
	mockAppConfigData := &mock.MockAppConfigDataClient{}

	client := NewTestClientWithData(mockAppConfig, mockAppConfigData)

	if client == nil {
		t.Fatal("NewTestClientWithData() returned nil")
	}

	if client.appConfig != mockAppConfig {
		t.Error("NewTestClientWithData() did not set appConfig correctly")
	}

	if client.AppConfigData != mockAppConfigData {
		t.Error("NewTestClientWithData() did not set AppConfigData correctly")
	}

	if client.Region != "us-east-1" {
		t.Errorf("NewTestClientWithData() Region = %s, want us-east-1", client.Region)
	}
}

func TestNewTestClientFull(t *testing.T) {
	mockAppConfig := &mock.MockAppConfigClient{}
	mockAppConfigData := &mock.MockAppConfigDataClient{}
	testRegion := "ap-northeast-1"
	testInterval := 10 * time.Second

	client := NewTestClientFull(mockAppConfig, mockAppConfigData, testRegion, testInterval)

	if client == nil {
		t.Fatal("NewTestClientFull() returned nil")
	}

	if client.appConfig != mockAppConfig {
		t.Error("NewTestClientFull() did not set appConfig correctly")
	}

	if client.AppConfigData != mockAppConfigData {
		t.Error("NewTestClientFull() did not set AppConfigData correctly")
	}

	if client.Region != testRegion {
		t.Errorf("NewTestClientFull() Region = %s, want %s", client.Region, testRegion)
	}

	if client.PollingInterval != testInterval {
		t.Errorf("NewTestClientFull() PollingInterval = %v, want %v", client.PollingInterval, testInterval)
	}
}
