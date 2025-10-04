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
		mockEventLog  []types.DeploymentEvent
		timeout       time.Duration
		wantErr       bool
		wantComplete  bool
		wantErrMsg    string
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
			wantErrMsg:    "deployment was rolled back",
		},
		{
			name:          "deployment is rolled back with CloudWatch alarm reason",
			deploymentNum: 5,
			mockStates:    []types.DeploymentState{types.DeploymentStateRolledBack},
			mockEventLog: []types.DeploymentEvent{
				{
					EventType:   types.DeploymentEventTypeRollbackStarted,
					Description: aws_stringPtr("Rollback initiated by CloudWatch Alarm: arn:aws:cloudwatch:us-east-1:123456789012:alarm:HighErrorRate"),
				},
			},
			timeout:      10 * time.Second,
			wantErr:      true,
			wantComplete: false,
			wantErrMsg:   "deployment was rolled back: Rollback initiated by CloudWatch Alarm: arn:aws:cloudwatch:us-east-1:123456789012:alarm:HighErrorRate",
		},
		{
			name:          "deployment is rolled back with custom reason",
			deploymentNum: 6,
			mockStates:    []types.DeploymentState{types.DeploymentStateRolledBack},
			mockEventLog: []types.DeploymentEvent{
				{
					EventType:   types.DeploymentEventTypeRollbackStarted,
					Description: aws_stringPtr("Rollback initiated manually"),
				},
			},
			timeout:      10 * time.Second,
			wantErr:      true,
			wantComplete: false,
			wantErrMsg:   "deployment was rolled back: Rollback initiated manually",
		},
		{
			name:          "deployment is rolled back by user request (RollbackCompleted event)",
			deploymentNum: 7,
			mockStates:    []types.DeploymentState{types.DeploymentStateRolledBack},
			mockEventLog: []types.DeploymentEvent{
				{
					EventType:   types.DeploymentEventTypeRollbackCompleted,
					Description: aws_stringPtr("Deployment rolled back by user request"),
				},
			},
			timeout:      10 * time.Second,
			wantErr:      true,
			wantComplete: false,
			wantErrMsg:   "deployment was rolled back: Deployment rolled back by user request",
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
						EventLog:           tt.mockEventLog,
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

			if tt.wantErrMsg != "" && err != nil {
				if err.Error() != tt.wantErrMsg {
					t.Errorf("WaitForDeployment() error message = %q, want %q", err.Error(), tt.wantErrMsg)
				}
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

func TestGetLatestDeployment(t *testing.T) {
	tests := []struct {
		name              string
		deployments       []types.DeploymentSummary
		getDeploymentFunc func(ctx context.Context, params *appconfig.GetDeploymentInput, optFns ...func(*appconfig.Options)) (*appconfig.GetDeploymentOutput, error)
		profileID         string
		wantDeployment    *DeploymentInfo
		wantErr           bool
	}{
		{
			name:        "no deployments",
			deployments: []types.DeploymentSummary{},
			getDeploymentFunc: func(ctx context.Context, params *appconfig.GetDeploymentInput, optFns ...func(*appconfig.Options)) (*appconfig.GetDeploymentOutput, error) {
				return nil, nil
			},
			profileID:      "profile-123",
			wantDeployment: nil,
			wantErr:        false,
		},
		{
			name: "single matching deployment",
			deployments: []types.DeploymentSummary{
				{DeploymentNumber: 1},
			},
			getDeploymentFunc: func(ctx context.Context, params *appconfig.GetDeploymentInput, optFns ...func(*appconfig.Options)) (*appconfig.GetDeploymentOutput, error) {
				return &appconfig.GetDeploymentOutput{
					DeploymentNumber:       1,
					ConfigurationProfileId: aws_stringPtr("profile-123"),
					ConfigurationVersion:   aws_stringPtr("5"),
					State:                  types.DeploymentStateComplete,
					Description:            aws_stringPtr("test deployment"),
				}, nil
			},
			profileID: "profile-123",
			wantDeployment: &DeploymentInfo{
				DeploymentNumber:     1,
				ConfigurationVersion: "5",
				State:                types.DeploymentStateComplete,
				Description:          "test deployment",
			},
			wantErr: false,
		},
		{
			name: "multiple deployments returns latest",
			deployments: []types.DeploymentSummary{
				{DeploymentNumber: 1},
				{DeploymentNumber: 3},
				{DeploymentNumber: 2},
			},
			getDeploymentFunc: func(ctx context.Context, params *appconfig.GetDeploymentInput, optFns ...func(*appconfig.Options)) (*appconfig.GetDeploymentOutput, error) {
				deployNum := *params.DeploymentNumber
				return &appconfig.GetDeploymentOutput{
					DeploymentNumber:       deployNum,
					ConfigurationProfileId: aws_stringPtr("profile-123"),
					ConfigurationVersion:   aws_stringPtr("5"),
					State:                  types.DeploymentStateComplete,
				}, nil
			},
			profileID: "profile-123",
			wantDeployment: &DeploymentInfo{
				DeploymentNumber:     3,
				ConfigurationVersion: "5",
				State:                types.DeploymentStateComplete,
				Description:          "",
			},
			wantErr: false,
		},
		{
			name: "ignores non-matching profile",
			deployments: []types.DeploymentSummary{
				{DeploymentNumber: 1},
				{DeploymentNumber: 2},
			},
			getDeploymentFunc: func(ctx context.Context, params *appconfig.GetDeploymentInput, optFns ...func(*appconfig.Options)) (*appconfig.GetDeploymentOutput, error) {
				deployNum := *params.DeploymentNumber
				profileID := "other-profile"
				if deployNum == 2 {
					profileID = "profile-123"
				}
				return &appconfig.GetDeploymentOutput{
					DeploymentNumber:       deployNum,
					ConfigurationProfileId: aws_stringPtr(profileID),
					ConfigurationVersion:   aws_stringPtr("5"),
					State:                  types.DeploymentStateComplete,
				}, nil
			},
			profileID: "profile-123",
			wantDeployment: &DeploymentInfo{
				DeploymentNumber:     2,
				ConfigurationVersion: "5",
				State:                types.DeploymentStateComplete,
				Description:          "",
			},
			wantErr: false,
		},
		{
			name: "ignores ROLLED_BACK deployment and returns last successful",
			deployments: []types.DeploymentSummary{
				{DeploymentNumber: 1},
				{DeploymentNumber: 2},
				{DeploymentNumber: 3},
			},
			getDeploymentFunc: func(ctx context.Context, params *appconfig.GetDeploymentInput, optFns ...func(*appconfig.Options)) (*appconfig.GetDeploymentOutput, error) {
				deployNum := *params.DeploymentNumber
				state := types.DeploymentStateComplete
				configVersion := "5"
				// Deployment 3 is ROLLED_BACK
				if deployNum == 3 {
					state = types.DeploymentStateRolledBack
					configVersion = "7"
				}
				// Deployment 2 is the last successful (COMPLETE)
				if deployNum == 2 {
					configVersion = "6"
				}
				return &appconfig.GetDeploymentOutput{
					DeploymentNumber:       deployNum,
					ConfigurationProfileId: aws_stringPtr("profile-123"),
					ConfigurationVersion:   aws_stringPtr(configVersion),
					State:                  state,
				}, nil
			},
			profileID: "profile-123",
			wantDeployment: &DeploymentInfo{
				DeploymentNumber:     2,
				ConfigurationVersion: "6",
				State:                types.DeploymentStateComplete,
				Description:          "",
			},
			wantErr: false,
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
				GetDeploymentFunc: tt.getDeploymentFunc,
			}

			client := &Client{AppConfig: mockClient}
			deployment, err := GetLatestDeployment(context.Background(), client, "app-123", "env-123", tt.profileID)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetLatestDeployment() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantDeployment == nil {
				if deployment != nil {
					t.Errorf("GetLatestDeployment() = %v, want nil", deployment)
				}
				return
			}

			if deployment == nil {
				t.Error("GetLatestDeployment() = nil, want deployment")
				return
			}

			if deployment.DeploymentNumber != tt.wantDeployment.DeploymentNumber {
				t.Errorf("DeploymentNumber = %v, want %v", deployment.DeploymentNumber, tt.wantDeployment.DeploymentNumber)
			}
			if deployment.ConfigurationVersion != tt.wantDeployment.ConfigurationVersion {
				t.Errorf("ConfigurationVersion = %v, want %v", deployment.ConfigurationVersion, tt.wantDeployment.ConfigurationVersion)
			}
			if deployment.State != tt.wantDeployment.State {
				t.Errorf("State = %v, want %v", deployment.State, tt.wantDeployment.State)
			}
		})
	}
}

func TestGetLatestDeploymentIncludingRollback(t *testing.T) {
	tests := []struct {
		name              string
		deployments       []types.DeploymentSummary
		getDeploymentFunc func(ctx context.Context, params *appconfig.GetDeploymentInput, optFns ...func(*appconfig.Options)) (*appconfig.GetDeploymentOutput, error)
		profileID         string
		wantDeployment    *DeploymentInfo
		wantErr           bool
	}{
		{
			name: "returns ROLLED_BACK deployment when it's the latest",
			deployments: []types.DeploymentSummary{
				{DeploymentNumber: 1},
				{DeploymentNumber: 2},
				{DeploymentNumber: 3},
			},
			getDeploymentFunc: func(ctx context.Context, params *appconfig.GetDeploymentInput, optFns ...func(*appconfig.Options)) (*appconfig.GetDeploymentOutput, error) {
				deployNum := *params.DeploymentNumber
				state := types.DeploymentStateComplete
				configVersion := "5"
				// Deployment 3 is ROLLED_BACK (latest)
				if deployNum == 3 {
					state = types.DeploymentStateRolledBack
					configVersion = "7"
				}
				// Deployment 2 is successful
				if deployNum == 2 {
					configVersion = "6"
				}
				return &appconfig.GetDeploymentOutput{
					DeploymentNumber:       deployNum,
					ConfigurationProfileId: aws_stringPtr("profile-123"),
					ConfigurationVersion:   aws_stringPtr(configVersion),
					State:                  state,
				}, nil
			},
			profileID: "profile-123",
			wantDeployment: &DeploymentInfo{
				DeploymentNumber:     3,
				ConfigurationVersion: "7",
				State:                types.DeploymentStateRolledBack,
				Description:          "",
			},
			wantErr: false,
		},
		{
			name: "returns latest deployment regardless of state",
			deployments: []types.DeploymentSummary{
				{DeploymentNumber: 5},
				{DeploymentNumber: 7},
			},
			getDeploymentFunc: func(ctx context.Context, params *appconfig.GetDeploymentInput, optFns ...func(*appconfig.Options)) (*appconfig.GetDeploymentOutput, error) {
				deployNum := *params.DeploymentNumber
				return &appconfig.GetDeploymentOutput{
					DeploymentNumber:       deployNum,
					ConfigurationProfileId: aws_stringPtr("profile-123"),
					ConfigurationVersion:   aws_stringPtr("10"),
					State:                  types.DeploymentStateComplete,
				}, nil
			},
			profileID: "profile-123",
			wantDeployment: &DeploymentInfo{
				DeploymentNumber:     7,
				ConfigurationVersion: "10",
				State:                types.DeploymentStateComplete,
				Description:          "",
			},
			wantErr: false,
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
				GetDeploymentFunc: tt.getDeploymentFunc,
			}

			client := &Client{AppConfig: mockClient}
			deployment, err := GetLatestDeploymentIncludingRollback(context.Background(), client, "app-123", "env-123", tt.profileID)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetLatestDeploymentIncludingRollback() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantDeployment == nil {
				if deployment != nil {
					t.Errorf("GetLatestDeploymentIncludingRollback() = %v, want nil", deployment)
				}
				return
			}

			if deployment == nil {
				t.Error("GetLatestDeploymentIncludingRollback() = nil, want deployment")
				return
			}

			if deployment.DeploymentNumber != tt.wantDeployment.DeploymentNumber {
				t.Errorf("DeploymentNumber = %v, want %v", deployment.DeploymentNumber, tt.wantDeployment.DeploymentNumber)
			}
			if deployment.ConfigurationVersion != tt.wantDeployment.ConfigurationVersion {
				t.Errorf("ConfigurationVersion = %v, want %v", deployment.ConfigurationVersion, tt.wantDeployment.ConfigurationVersion)
			}
			if deployment.State != tt.wantDeployment.State {
				t.Errorf("State = %v, want %v", deployment.State, tt.wantDeployment.State)
			}
		})
	}
}

func TestGetHostedConfigurationVersion(t *testing.T) {
	tests := []struct {
		name          string
		versionNumber string
		mockFunc      func(ctx context.Context, params *appconfig.GetHostedConfigurationVersionInput, optFns ...func(*appconfig.Options)) (*appconfig.GetHostedConfigurationVersionOutput, error)
		wantContent   []byte
		wantErr       bool
	}{
		{
			name:          "successful retrieval",
			versionNumber: "5",
			mockFunc: func(ctx context.Context, params *appconfig.GetHostedConfigurationVersionInput, optFns ...func(*appconfig.Options)) (*appconfig.GetHostedConfigurationVersionOutput, error) {
				return &appconfig.GetHostedConfigurationVersionOutput{
					Content:     []byte(`{"key": "value"}`),
					ContentType: aws_stringPtr("application/json"),
				}, nil
			},
			wantContent: []byte(`{"key": "value"}`),
			wantErr:     false,
		},
		{
			name:          "invalid version number format",
			versionNumber: "invalid",
			mockFunc: func(ctx context.Context, params *appconfig.GetHostedConfigurationVersionInput, optFns ...func(*appconfig.Options)) (*appconfig.GetHostedConfigurationVersionOutput, error) {
				return nil, errors.New("should not be called")
			},
			wantContent: nil,
			wantErr:     true,
		},
		{
			name:          "API error",
			versionNumber: "5",
			mockFunc: func(ctx context.Context, params *appconfig.GetHostedConfigurationVersionInput, optFns ...func(*appconfig.Options)) (*appconfig.GetHostedConfigurationVersionOutput, error) {
				return nil, errors.New("API error")
			},
			wantContent: nil,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &mock.MockAppConfigClient{
				GetHostedConfigurationVersionFunc: tt.mockFunc,
			}

			client := &Client{AppConfig: mockClient}
			content, err := GetHostedConfigurationVersion(context.Background(), client, "app-123", "profile-123", tt.versionNumber)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetHostedConfigurationVersion() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && string(content) != string(tt.wantContent) {
				t.Errorf("GetHostedConfigurationVersion() content = %s, want %s", content, tt.wantContent)
			}
		})
	}
}
