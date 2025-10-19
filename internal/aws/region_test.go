package aws_test

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/account"
	accountTypes "github.com/aws/aws-sdk-go-v2/service/account/types"
	awsInternal "github.com/koh-sh/apcdeploy/internal/aws"
	"github.com/koh-sh/apcdeploy/internal/aws/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListEnabledRegions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		setupMock      func() *mock.MockAccountClient
		expectedResult []string
		expectError    bool
		checkResult    func(*testing.T, []string, error)
	}{
		{
			name: "success returns sorted regions",
			setupMock: func() *mock.MockAccountClient {
				return &mock.MockAccountClient{
					ListRegionsFunc: func(ctx context.Context, params *account.ListRegionsInput, optFns ...func(*account.Options)) (*account.ListRegionsOutput, error) {
						return &account.ListRegionsOutput{
							Regions: []accountTypes.Region{
								{RegionName: aws.String("us-west-2"), RegionOptStatus: accountTypes.RegionOptStatusEnabled},
								{RegionName: aws.String("ap-northeast-1"), RegionOptStatus: accountTypes.RegionOptStatusEnabledByDefault},
								{RegionName: aws.String("us-east-1"), RegionOptStatus: accountTypes.RegionOptStatusEnabled},
							},
						}, nil
					},
				}
			},
			expectedResult: []string{"ap-northeast-1", "us-east-1", "us-west-2"},
			expectError:    false,
			checkResult: func(t *testing.T, regions []string, err error) {
				require.NoError(t, err)
				assert.Equal(t, []string{"ap-northeast-1", "us-east-1", "us-west-2"}, regions)
			},
		},
		{
			name: "filters disabled regions",
			setupMock: func() *mock.MockAccountClient {
				return &mock.MockAccountClient{
					ListRegionsFunc: func(ctx context.Context, params *account.ListRegionsInput, optFns ...func(*account.Options)) (*account.ListRegionsOutput, error) {
						return &account.ListRegionsOutput{
							Regions: []accountTypes.Region{
								{RegionName: aws.String("us-east-1"), RegionOptStatus: accountTypes.RegionOptStatusEnabled},
								{RegionName: aws.String("ap-east-1"), RegionOptStatus: accountTypes.RegionOptStatusDisabled},
								{RegionName: aws.String("us-west-2"), RegionOptStatus: accountTypes.RegionOptStatusEnabledByDefault},
							},
						}, nil
					},
				}
			},
			expectedResult: []string{"us-east-1", "us-west-2"},
			expectError:    false,
			checkResult: func(t *testing.T, regions []string, err error) {
				require.NoError(t, err)
				assert.Equal(t, []string{"us-east-1", "us-west-2"}, regions)
				assert.NotContains(t, regions, "ap-east-1")
			},
		},
		{
			name: "empty list returns empty",
			setupMock: func() *mock.MockAccountClient {
				return &mock.MockAccountClient{
					ListRegionsFunc: func(ctx context.Context, params *account.ListRegionsInput, optFns ...func(*account.Options)) (*account.ListRegionsOutput, error) {
						return &account.ListRegionsOutput{
							Regions: []accountTypes.Region{},
						}, nil
					},
				}
			},
			expectedResult: []string{},
			expectError:    false,
			checkResult: func(t *testing.T, regions []string, err error) {
				require.NoError(t, err)
				assert.Empty(t, regions)
			},
		},
		{
			name: "API error returns error",
			setupMock: func() *mock.MockAccountClient {
				return &mock.MockAccountClient{
					ListRegionsFunc: func(ctx context.Context, params *account.ListRegionsInput, optFns ...func(*account.Options)) (*account.ListRegionsOutput, error) {
						return nil, errors.New("API error")
					},
				}
			},
			expectedResult: nil,
			expectError:    true,
			checkResult: func(t *testing.T, regions []string, err error) {
				require.Error(t, err)
				assert.EqualError(t, err, "API error")
				assert.Nil(t, regions)
			},
		},
		{
			name: "handles nil region name",
			setupMock: func() *mock.MockAccountClient {
				return &mock.MockAccountClient{
					ListRegionsFunc: func(ctx context.Context, params *account.ListRegionsInput, optFns ...func(*account.Options)) (*account.ListRegionsOutput, error) {
						return &account.ListRegionsOutput{
							Regions: []accountTypes.Region{
								{RegionName: aws.String("us-east-1"), RegionOptStatus: accountTypes.RegionOptStatusEnabled},
								{RegionName: nil, RegionOptStatus: accountTypes.RegionOptStatusEnabled},
								{RegionName: aws.String("us-west-2"), RegionOptStatus: accountTypes.RegionOptStatusEnabled},
							},
						}, nil
					},
				}
			},
			expectedResult: []string{"us-east-1", "us-west-2"},
			expectError:    false,
			checkResult: func(t *testing.T, regions []string, err error) {
				require.NoError(t, err)
				assert.Equal(t, []string{"us-east-1", "us-west-2"}, regions)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			mockClient := tt.setupMock()

			regions, err := awsInternal.ListEnabledRegions(ctx, mockClient)

			tt.checkResult(t, regions, err)
		})
	}
}

func TestListEnabledRegions_RequestsCorrectFilter(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	var capturedInput *account.ListRegionsInput

	mockClient := &mock.MockAccountClient{
		ListRegionsFunc: func(ctx context.Context, params *account.ListRegionsInput, optFns ...func(*account.Options)) (*account.ListRegionsOutput, error) {
			capturedInput = params
			return &account.ListRegionsOutput{
				Regions: []accountTypes.Region{},
			}, nil
		},
	}

	_, err := awsInternal.ListEnabledRegions(ctx, mockClient)
	require.NoError(t, err)

	// Verify correct filter was requested
	require.NotNil(t, capturedInput)
	assert.Contains(t, capturedInput.RegionOptStatusContains, accountTypes.RegionOptStatusEnabled)
	assert.Contains(t, capturedInput.RegionOptStatusContains, accountTypes.RegionOptStatusEnabledByDefault)
}
