package aws

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/appconfig"
	"github.com/aws/aws-sdk-go-v2/service/appconfig/types"
	"github.com/koh-sh/apcdeploy/internal/aws/mock"
)

func TestNewConfigVersionFetcher(t *testing.T) {
	// Use the actual client since we just need to verify the constructor
	// The client's internal state doesn't matter for this test
	ctx := context.Background()

	// Set AWS region via environment to avoid errors
	t.Setenv("AWS_REGION", "us-east-1")

	client, err := NewClient(ctx, "us-east-1")
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	fetcher := NewConfigVersionFetcher(client)

	if fetcher.client == nil {
		t.Error("fetcher client should not be nil")
	}
}

func TestGetLatestVersion(t *testing.T) {
	tests := []struct {
		name              string
		appID             string
		profileID         string
		mockVersions      []types.HostedConfigurationVersionSummary
		mockVersion       *appconfig.GetHostedConfigurationVersionOutput
		mockListErr       error
		mockGetErr        error
		wantVersionNumber int32
		wantContentType   string
		wantErr           bool
		errContains       string
	}{
		{
			name:      "successful latest version retrieval",
			appID:     "app-123",
			profileID: "prof-456",
			mockVersions: []types.HostedConfigurationVersionSummary{
				{
					VersionNumber: 1,
				},
				{
					VersionNumber: 3,
				},
				{
					VersionNumber: 2,
				},
			},
			mockVersion: &appconfig.GetHostedConfigurationVersionOutput{
				VersionNumber: 3,
				Content:       []byte(`{"key": "value"}`),
				ContentType:   aws.String("application/json"),
			},
			wantVersionNumber: 3,
			wantContentType:   "application/json",
			wantErr:           false,
		},
		{
			name:      "successful latest version retrieval without content type",
			appID:     "app-123",
			profileID: "prof-456",
			mockVersions: []types.HostedConfigurationVersionSummary{
				{
					VersionNumber: 1,
				},
			},
			mockVersion: &appconfig.GetHostedConfigurationVersionOutput{
				VersionNumber: 1,
				Content:       []byte(`{"key": "value"}`),
				ContentType:   nil,
			},
			wantVersionNumber: 1,
			wantContentType:   "",
			wantErr:           false,
		},
		{
			name:         "no versions found",
			appID:        "app-123",
			profileID:    "prof-456",
			mockVersions: []types.HostedConfigurationVersionSummary{},
			wantErr:      true,
			errContains:  "no configuration versions found",
		},
		{
			name:        "list API error",
			appID:       "app-123",
			profileID:   "prof-456",
			mockListErr: errors.New("API error"),
			wantErr:     true,
			errContains: "failed to list configuration versions",
		},
		{
			name:      "get version API error",
			appID:     "app-123",
			profileID: "prof-456",
			mockVersions: []types.HostedConfigurationVersionSummary{
				{
					VersionNumber: 1,
				},
			},
			mockGetErr:  errors.New("API error"),
			wantErr:     true,
			errContains: "failed to get configuration version",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &mock.MockAppConfigClient{
				ListHostedConfigurationVersionsFunc: func(ctx context.Context, params *appconfig.ListHostedConfigurationVersionsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListHostedConfigurationVersionsOutput, error) {
					if tt.mockListErr != nil {
						return nil, tt.mockListErr
					}
					return &appconfig.ListHostedConfigurationVersionsOutput{
						Items: tt.mockVersions,
					}, nil
				},
				GetHostedConfigurationVersionFunc: func(ctx context.Context, params *appconfig.GetHostedConfigurationVersionInput, optFns ...func(*appconfig.Options)) (*appconfig.GetHostedConfigurationVersionOutput, error) {
					if tt.mockGetErr != nil {
						return nil, tt.mockGetErr
					}
					return tt.mockVersion, nil
				},
			}

			fetcher := &ConfigVersionFetcher{
				client: mockClient,
			}

			ctx := context.Background()
			versionInfo, err := fetcher.GetLatestVersion(ctx, tt.appID, tt.profileID)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error = %v, want to contain %v", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if versionInfo.VersionNumber != tt.wantVersionNumber {
				t.Errorf("VersionNumber = %v, want %v", versionInfo.VersionNumber, tt.wantVersionNumber)
			}

			if versionInfo.ContentType != tt.wantContentType {
				t.Errorf("ContentType = %v, want %v", versionInfo.ContentType, tt.wantContentType)
			}
		})
	}
}
