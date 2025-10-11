package mock

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/appconfigdata"
)

// MockAppConfigDataClient is a mock implementation of aws.AppConfigDataAPI.
type MockAppConfigDataClient struct {
	StartConfigurationSessionFunc func(ctx context.Context, params *appconfigdata.StartConfigurationSessionInput, optFns ...func(*appconfigdata.Options)) (*appconfigdata.StartConfigurationSessionOutput, error)
	GetLatestConfigurationFunc    func(ctx context.Context, params *appconfigdata.GetLatestConfigurationInput, optFns ...func(*appconfigdata.Options)) (*appconfigdata.GetLatestConfigurationOutput, error)
}

func (m *MockAppConfigDataClient) StartConfigurationSession(ctx context.Context, params *appconfigdata.StartConfigurationSessionInput, optFns ...func(*appconfigdata.Options)) (*appconfigdata.StartConfigurationSessionOutput, error) {
	return m.StartConfigurationSessionFunc(ctx, params, optFns...)
}

func (m *MockAppConfigDataClient) GetLatestConfiguration(ctx context.Context, params *appconfigdata.GetLatestConfigurationInput, optFns ...func(*appconfigdata.Options)) (*appconfigdata.GetLatestConfigurationOutput, error) {
	return m.GetLatestConfigurationFunc(ctx, params, optFns...)
}
