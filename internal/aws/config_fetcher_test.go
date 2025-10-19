package aws

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/appconfig"
	"github.com/aws/aws-sdk-go-v2/service/appconfig/types"
	"github.com/koh-sh/apcdeploy/internal/aws/mock"
)

func TestGetLatestDeployedConfiguration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		mockSetup func(*mock.MockAppConfigClient)
		wantNil   bool
		wantErr   bool
		validate  func(*testing.T, *DeployedConfigInfo)
	}{
		{
			name: "successfully retrieves deployed configuration",
			mockSetup: func(m *mock.MockAppConfigClient) {
				m.ListDeploymentsFunc = func(ctx context.Context, params *appconfig.ListDeploymentsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListDeploymentsOutput, error) {
					return &appconfig.ListDeploymentsOutput{
						Items: []types.DeploymentSummary{
							{
								DeploymentNumber:     1,
								ConfigurationVersion: aws.String("2"),
								State:                types.DeploymentStateComplete,
							},
						},
					}, nil
				}
				m.GetDeploymentFunc = func(ctx context.Context, params *appconfig.GetDeploymentInput, optFns ...func(*appconfig.Options)) (*appconfig.GetDeploymentOutput, error) {
					return &appconfig.GetDeploymentOutput{
						DeploymentNumber:       1,
						ConfigurationProfileId: aws.String("prof-789"),
						ConfigurationVersion:   aws.String("2"),
						State:                  types.DeploymentStateComplete,
					}, nil
				}
				m.GetHostedConfigurationVersionFunc = func(ctx context.Context, params *appconfig.GetHostedConfigurationVersionInput, optFns ...func(*appconfig.Options)) (*appconfig.GetHostedConfigurationVersionOutput, error) {
					return &appconfig.GetHostedConfigurationVersionOutput{
						Content:     []byte(`{"key":"value"}`),
						ContentType: aws.String("application/json"),
					}, nil
				}
			},
			wantNil: false,
			wantErr: false,
			validate: func(t *testing.T, info *DeployedConfigInfo) {
				if info.DeploymentNumber != 1 {
					t.Errorf("expected DeploymentNumber 1, got %d", info.DeploymentNumber)
				}
				if info.VersionNumber != 2 {
					t.Errorf("expected VersionNumber 2, got %d", info.VersionNumber)
				}
				if info.ContentType != "application/json" {
					t.Errorf("expected ContentType 'application/json', got %q", info.ContentType)
				}
				if string(info.Content) != `{"key":"value"}` {
					t.Errorf("expected Content %q, got %q", `{"key":"value"}`, string(info.Content))
				}
				if info.State != types.DeploymentStateComplete {
					t.Errorf("expected State COMPLETE, got %v", info.State)
				}
			},
		},
		{
			name: "returns nil when no deployment found",
			mockSetup: func(m *mock.MockAppConfigClient) {
				m.ListDeploymentsFunc = func(ctx context.Context, params *appconfig.ListDeploymentsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListDeploymentsOutput, error) {
					return &appconfig.ListDeploymentsOutput{
						Items: []types.DeploymentSummary{},
					}, nil
				}
			},
			wantNil: true,
			wantErr: false,
		},
		{
			name: "returns error when getting deployment details fails",
			mockSetup: func(m *mock.MockAppConfigClient) {
				m.ListDeploymentsFunc = func(ctx context.Context, params *appconfig.ListDeploymentsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListDeploymentsOutput, error) {
					return nil, fmt.Errorf("API error")
				}
			},
			wantNil: false,
			wantErr: true,
		},
		{
			name: "returns error when getting configuration version fails",
			mockSetup: func(m *mock.MockAppConfigClient) {
				m.ListDeploymentsFunc = func(ctx context.Context, params *appconfig.ListDeploymentsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListDeploymentsOutput, error) {
					return &appconfig.ListDeploymentsOutput{
						Items: []types.DeploymentSummary{
							{
								DeploymentNumber:     1,
								ConfigurationVersion: aws.String("2"),
								State:                types.DeploymentStateComplete,
							},
						},
					}, nil
				}
				m.GetDeploymentFunc = func(ctx context.Context, params *appconfig.GetDeploymentInput, optFns ...func(*appconfig.Options)) (*appconfig.GetDeploymentOutput, error) {
					return &appconfig.GetDeploymentOutput{
						DeploymentNumber:       1,
						ConfigurationProfileId: aws.String("prof-789"),
						ConfigurationVersion:   aws.String("2"),
						State:                  types.DeploymentStateComplete,
					}, nil
				}
				m.GetHostedConfigurationVersionFunc = func(ctx context.Context, params *appconfig.GetHostedConfigurationVersionInput, optFns ...func(*appconfig.Options)) (*appconfig.GetHostedConfigurationVersionOutput, error) {
					return nil, fmt.Errorf("version not found")
				}
			},
			wantNil: false,
			wantErr: true,
		},
		{
			name: "returns error when version number format is invalid",
			mockSetup: func(m *mock.MockAppConfigClient) {
				m.ListDeploymentsFunc = func(ctx context.Context, params *appconfig.ListDeploymentsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListDeploymentsOutput, error) {
					return &appconfig.ListDeploymentsOutput{
						Items: []types.DeploymentSummary{
							{
								DeploymentNumber:     1,
								ConfigurationVersion: aws.String("invalid"),
								State:                types.DeploymentStateComplete,
							},
						},
					}, nil
				}
				m.GetDeploymentFunc = func(ctx context.Context, params *appconfig.GetDeploymentInput, optFns ...func(*appconfig.Options)) (*appconfig.GetDeploymentOutput, error) {
					return &appconfig.GetDeploymentOutput{
						DeploymentNumber:       1,
						ConfigurationProfileId: aws.String("prof-789"),
						ConfigurationVersion:   aws.String("invalid"),
						State:                  types.DeploymentStateComplete,
					}, nil
				}
			},
			wantNil: false,
			wantErr: true,
		},
		{
			name: "defaults to application/json when content type is nil",
			mockSetup: func(m *mock.MockAppConfigClient) {
				m.ListDeploymentsFunc = func(ctx context.Context, params *appconfig.ListDeploymentsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListDeploymentsOutput, error) {
					return &appconfig.ListDeploymentsOutput{
						Items: []types.DeploymentSummary{
							{
								DeploymentNumber:     1,
								ConfigurationVersion: aws.String("1"),
								State:                types.DeploymentStateComplete,
							},
						},
					}, nil
				}
				m.GetDeploymentFunc = func(ctx context.Context, params *appconfig.GetDeploymentInput, optFns ...func(*appconfig.Options)) (*appconfig.GetDeploymentOutput, error) {
					return &appconfig.GetDeploymentOutput{
						DeploymentNumber:       1,
						ConfigurationProfileId: aws.String("prof-789"),
						ConfigurationVersion:   aws.String("1"),
						State:                  types.DeploymentStateComplete,
					}, nil
				}
				m.GetHostedConfigurationVersionFunc = func(ctx context.Context, params *appconfig.GetHostedConfigurationVersionInput, optFns ...func(*appconfig.Options)) (*appconfig.GetHostedConfigurationVersionOutput, error) {
					// Return nil ContentType
					return &appconfig.GetHostedConfigurationVersionOutput{
						Content:     []byte(`{"key":"value"}`),
						ContentType: nil,
					}, nil
				}
			},
			wantNil: false,
			wantErr: false,
			validate: func(t *testing.T, info *DeployedConfigInfo) {
				if info.ContentType != "application/json" {
					t.Errorf("expected default ContentType 'application/json', got %q", info.ContentType)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockClient := &mock.MockAppConfigClient{}
			tt.mockSetup(mockClient)

			client := NewTestClient(mockClient)

			result, err := GetLatestDeployedConfiguration(context.Background(), client, "app-123", "env-456", "prof-789")

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.wantNil {
				if result != nil {
					t.Errorf("expected nil result, got %+v", result)
				}
				return
			}

			if result == nil {
				t.Fatal("expected non-nil result")
			}

			if tt.validate != nil {
				tt.validate(t, result)
			}
		})
	}
}
