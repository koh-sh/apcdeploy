package mock

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/account"
)

// MockAccountClient is a mock implementation of aws.AccountAPI
type MockAccountClient struct {
	ListRegionsFunc func(ctx context.Context, params *account.ListRegionsInput, optFns ...func(*account.Options)) (*account.ListRegionsOutput, error)
}

func (m *MockAccountClient) ListRegions(ctx context.Context, params *account.ListRegionsInput, optFns ...func(*account.Options)) (*account.ListRegionsOutput, error) {
	return m.ListRegionsFunc(ctx, params, optFns...)
}
