package aws

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/appconfig"
	"github.com/aws/aws-sdk-go-v2/service/appconfig/types"
	"github.com/koh-sh/apcdeploy/internal/aws/mock"
)

func TestCheckOngoingDeployment(t *testing.T) {
	tests := []struct {
		name            string
		deployments     []types.DeploymentSummary
		expectedOngoing bool
		expectedErr     bool
	}{
		{
			name:            "no deployments",
			deployments:     []types.DeploymentSummary{},
			expectedOngoing: false,
			expectedErr:     false,
		},
		{
			name: "only completed deployments",
			deployments: []types.DeploymentSummary{
				{
					DeploymentNumber: 1,
					State:            types.DeploymentStateComplete,
				},
			},
			expectedOngoing: false,
			expectedErr:     false,
		},
		{
			name: "has deploying deployment",
			deployments: []types.DeploymentSummary{
				{
					DeploymentNumber: 1,
					State:            types.DeploymentStateDeploying,
				},
			},
			expectedOngoing: true,
			expectedErr:     false,
		},
		{
			name: "has baking deployment",
			deployments: []types.DeploymentSummary{
				{
					DeploymentNumber: 1,
					State:            types.DeploymentStateBaking,
				},
			},
			expectedOngoing: true,
			expectedErr:     false,
		},
		{
			name: "mixed states",
			deployments: []types.DeploymentSummary{
				{
					DeploymentNumber: 2,
					State:            types.DeploymentStateDeploying,
				},
				{
					DeploymentNumber: 1,
					State:            types.DeploymentStateComplete,
				},
			},
			expectedOngoing: true,
			expectedErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &mock.MockAppConfigClient{
				ListDeploymentsFunc: func(ctx context.Context, params *appconfig.ListDeploymentsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListDeploymentsOutput, error) {
					return &appconfig.ListDeploymentsOutput{
						Items: tt.deployments,
					}, nil
				},
			}

			client := &Client{AppConfig: mockClient}
			hasOngoing, deployment, err := client.CheckOngoingDeployment(context.Background(), "app-123", "env-123")

			if (err != nil) != tt.expectedErr {
				t.Errorf("CheckOngoingDeployment() error = %v, expectedErr %v", err, tt.expectedErr)
				return
			}

			if hasOngoing != tt.expectedOngoing {
				t.Errorf("CheckOngoingDeployment() hasOngoing = %v, want %v", hasOngoing, tt.expectedOngoing)
			}

			if tt.expectedOngoing && deployment == nil {
				t.Error("Expected deployment to be returned when ongoing deployment exists")
			}

			if !tt.expectedOngoing && deployment != nil {
				t.Error("Expected no deployment to be returned when no ongoing deployment")
			}
		})
	}
}

func TestCreateHostedConfigurationVersion(t *testing.T) {
	tests := []struct {
		name        string
		content     []byte
		contentType string
		description string
		mockFunc    func(ctx context.Context, params *appconfig.CreateHostedConfigurationVersionInput, optFns ...func(*appconfig.Options)) (*appconfig.CreateHostedConfigurationVersionOutput, error)
		wantErr     bool
	}{
		{
			name:        "successful creation with JSON",
			content:     []byte(`{"key": "value"}`),
			contentType: "application/json",
			description: "test version",
			mockFunc: func(ctx context.Context, params *appconfig.CreateHostedConfigurationVersionInput, optFns ...func(*appconfig.Options)) (*appconfig.CreateHostedConfigurationVersionOutput, error) {
				return &appconfig.CreateHostedConfigurationVersionOutput{
					VersionNumber: 1,
					ContentType:   aws_stringPtr("application/json"),
				}, nil
			},
			wantErr: false,
		},
		{
			name:        "successful creation with YAML",
			content:     []byte("key: value"),
			contentType: "application/x-yaml",
			description: "",
			mockFunc: func(ctx context.Context, params *appconfig.CreateHostedConfigurationVersionInput, optFns ...func(*appconfig.Options)) (*appconfig.CreateHostedConfigurationVersionOutput, error) {
				return &appconfig.CreateHostedConfigurationVersionOutput{
					VersionNumber: 2,
					ContentType:   aws_stringPtr("application/x-yaml"),
				}, nil
			},
			wantErr: false,
		},
		{
			name:        "API error",
			content:     []byte("content"),
			contentType: "text/plain",
			description: "test",
			mockFunc: func(ctx context.Context, params *appconfig.CreateHostedConfigurationVersionInput, optFns ...func(*appconfig.Options)) (*appconfig.CreateHostedConfigurationVersionOutput, error) {
				return nil, errors.New("API error")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &mock.MockAppConfigClient{
				CreateHostedConfigurationVersionFunc: tt.mockFunc,
			}

			client := &Client{AppConfig: mockClient}
			versionNum, err := client.CreateHostedConfigurationVersion(
				context.Background(),
				"app-123",
				"profile-123",
				tt.content,
				tt.contentType,
				tt.description,
			)

			if (err != nil) != tt.wantErr {
				t.Errorf("CreateHostedConfigurationVersion() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && versionNum == 0 {
				t.Error("Expected non-zero version number")
			}
		})
	}
}

func TestStartDeployment(t *testing.T) {
	tests := []struct {
		name        string
		strategyID  string
		versionNum  int32
		description string
		mockFunc    func(ctx context.Context, params *appconfig.StartDeploymentInput, optFns ...func(*appconfig.Options)) (*appconfig.StartDeploymentOutput, error)
		wantErr     bool
	}{
		{
			name:        "successful deployment start",
			strategyID:  "strategy-123",
			versionNum:  1,
			description: "test deployment",
			mockFunc: func(ctx context.Context, params *appconfig.StartDeploymentInput, optFns ...func(*appconfig.Options)) (*appconfig.StartDeploymentOutput, error) {
				return &appconfig.StartDeploymentOutput{
					DeploymentNumber: 10,
					State:            types.DeploymentStateDeploying,
				}, nil
			},
			wantErr: false,
		},
		{
			name:        "API error",
			strategyID:  "strategy-123",
			versionNum:  1,
			description: "",
			mockFunc: func(ctx context.Context, params *appconfig.StartDeploymentInput, optFns ...func(*appconfig.Options)) (*appconfig.StartDeploymentOutput, error) {
				return nil, errors.New("deployment failed")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &mock.MockAppConfigClient{
				StartDeploymentFunc: tt.mockFunc,
			}

			client := &Client{AppConfig: mockClient}
			deployNum, err := client.StartDeployment(
				context.Background(),
				"app-123",
				"env-123",
				"profile-123",
				tt.strategyID,
				tt.versionNum,
				tt.description,
			)

			if (err != nil) != tt.wantErr {
				t.Errorf("StartDeployment() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && deployNum == 0 {
				t.Error("Expected non-zero deployment number")
			}
		})
	}
}

func TestWaitForDeployment(t *testing.T) {
	tests := []struct {
		name          string
		deploymentNum int32
		mockStates    []types.DeploymentState
		timeout       time.Duration
		wantErr       bool
		wantComplete  bool
	}{
		{
			name:          "deployment completes immediately",
			deploymentNum: 1,
			mockStates:    []types.DeploymentState{types.DeploymentStateComplete},
			timeout:       10 * time.Second,
			wantErr:       false,
			wantComplete:  true,
		},
		{
			name:          "deployment is rolled back immediately",
			deploymentNum: 2,
			mockStates:    []types.DeploymentState{types.DeploymentStateRolledBack},
			timeout:       10 * time.Second,
			wantErr:       true,
			wantComplete:  false,
		},
		{
			name:          "deployment times out",
			deploymentNum: 3,
			mockStates:    []types.DeploymentState{types.DeploymentStateDeploying, types.DeploymentStateDeploying},
			timeout:       1 * time.Second,
			wantErr:       true,
			wantComplete:  false,
		},
		{
			name:          "deployment completes after one poll (5s wait)",
			deploymentNum: 4,
			mockStates:    []types.DeploymentState{types.DeploymentStateDeploying, types.DeploymentStateComplete},
			timeout:       10 * time.Second,
			wantErr:       false,
			wantComplete:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			callCount := 0
			mockClient := &mock.MockAppConfigClient{
				GetDeploymentFunc: func(ctx context.Context, params *appconfig.GetDeploymentInput, optFns ...func(*appconfig.Options)) (*appconfig.GetDeploymentOutput, error) {
					var state types.DeploymentState
					if callCount < len(tt.mockStates) {
						state = tt.mockStates[callCount]
					} else {
						state = tt.mockStates[len(tt.mockStates)-1]
					}
					callCount++

					return &appconfig.GetDeploymentOutput{
						DeploymentNumber:   tt.deploymentNum,
						State:              state,
						PercentageComplete: aws_floatPtr(float32(callCount) * 50.0),
					}, nil
				},
			}

			client := &Client{
				AppConfig:       mockClient,
				PollingInterval: 100 * time.Millisecond, // Fast polling for tests
			}
			err := client.WaitForDeployment(
				context.Background(),
				"app-123",
				"env-123",
				tt.deploymentNum,
				tt.timeout,
			)

			if (err != nil) != tt.wantErr {
				t.Errorf("WaitForDeployment() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Helper functions for tests
func aws_stringPtr(s string) *string {
	return &s
}

func aws_floatPtr(f float32) *float32 {
	return &f
}
