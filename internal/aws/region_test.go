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

func TestListEnabledRegions_Success_ReturnsSorted(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	mockClient := &mock.MockAccountClient{
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

	regions, err := awsInternal.ListEnabledRegions(ctx, mockClient)
	require.NoError(t, err)
	assert.Equal(t, []string{"ap-northeast-1", "us-east-1", "us-west-2"}, regions)
}

func TestListEnabledRegions_FiltersDisabled(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	mockClient := &mock.MockAccountClient{
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

	regions, err := awsInternal.ListEnabledRegions(ctx, mockClient)
	require.NoError(t, err)
	// Should only return enabled regions, not disabled
	assert.Equal(t, []string{"us-east-1", "us-west-2"}, regions)
	assert.NotContains(t, regions, "ap-east-1")
}

func TestListEnabledRegions_EmptyList_ReturnsEmpty(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	mockClient := &mock.MockAccountClient{
		ListRegionsFunc: func(ctx context.Context, params *account.ListRegionsInput, optFns ...func(*account.Options)) (*account.ListRegionsOutput, error) {
			return &account.ListRegionsOutput{
				Regions: []accountTypes.Region{},
			}, nil
		},
	}

	regions, err := awsInternal.ListEnabledRegions(ctx, mockClient)
	require.NoError(t, err)
	assert.Empty(t, regions)
}

func TestListEnabledRegions_APIError_ReturnsError(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	expectedErr := errors.New("API error")

	mockClient := &mock.MockAccountClient{
		ListRegionsFunc: func(ctx context.Context, params *account.ListRegionsInput, optFns ...func(*account.Options)) (*account.ListRegionsOutput, error) {
			return nil, expectedErr
		},
	}

	regions, err := awsInternal.ListEnabledRegions(ctx, mockClient)
	require.Error(t, err)
	assert.Equal(t, expectedErr, err)
	assert.Nil(t, regions)
}

func TestListEnabledRegions_HandlesNilRegionName(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	mockClient := &mock.MockAccountClient{
		ListRegionsFunc: func(ctx context.Context, params *account.ListRegionsInput, optFns ...func(*account.Options)) (*account.ListRegionsOutput, error) {
			return &account.ListRegionsOutput{
				Regions: []accountTypes.Region{
					{RegionName: aws.String("us-east-1"), RegionOptStatus: accountTypes.RegionOptStatusEnabled},
					{RegionName: nil, RegionOptStatus: accountTypes.RegionOptStatusEnabled}, // Nil region name
					{RegionName: aws.String("us-west-2"), RegionOptStatus: accountTypes.RegionOptStatusEnabled},
				},
			}, nil
		},
	}

	regions, err := awsInternal.ListEnabledRegions(ctx, mockClient)
	require.NoError(t, err)
	// Should skip nil region names
	assert.Equal(t, []string{"us-east-1", "us-west-2"}, regions)
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
