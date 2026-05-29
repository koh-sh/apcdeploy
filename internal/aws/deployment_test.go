package aws

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/appconfig"
	"github.com/aws/aws-sdk-go-v2/service/appconfig/types"
	"github.com/koh-sh/apcdeploy/internal/aws/mock"
)

func TestCheckOngoingDeployment(t *testing.T) {
	t.Parallel()
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
			name: "has validating deployment",
			deployments: []types.DeploymentSummary{
				{
					DeploymentNumber: 1,
					State:            types.DeploymentStateValidating,
				},
			},
			expectedOngoing: true,
			expectedErr:     false,
		},
		{
			name: "has rolling back deployment",
			deployments: []types.DeploymentSummary{
				{
					DeploymentNumber: 1,
					State:            types.DeploymentStateRollingBack,
				},
			},
			expectedOngoing: true,
			expectedErr:     false,
		},
		{
			name: "has reverted deployment",
			deployments: []types.DeploymentSummary{
				{
					DeploymentNumber: 1,
					State:            types.DeploymentStateReverted,
				},
			},
			expectedOngoing: true,
			expectedErr:     false,
		},
		{
			name: "rolled back deployment is not ongoing",
			deployments: []types.DeploymentSummary{
				{
					DeploymentNumber: 1,
					State:            types.DeploymentStateRolledBack,
				},
			},
			expectedOngoing: false,
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

			client := &Client{appConfig: mockClient}
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
	t.Parallel()
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
					ContentType:   new("application/json"),
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
					ContentType:   new("application/x-yaml"),
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

			client := &Client{appConfig: mockClient}
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
	t.Parallel()
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

			client := &Client{appConfig: mockClient}
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

// TestWaitForDeploymentPhase_FullCompletion exercises waitForBaking=true,
// covering rollback-reason extraction paths that the parameterized
// TestWaitForDeploymentPhase does not.
func TestWaitForDeploymentPhase_FullCompletion(t *testing.T) {
	t.Parallel()
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
					Description: new("Rollback initiated by CloudWatch Alarm: arn:aws:cloudwatch:us-east-1:123456789012:alarm:HighErrorRate"),
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
					Description: new("Rollback initiated manually"),
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
					Description: new("Deployment rolled back by user request"),
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
						PercentageComplete: new(float32(callCount) * 50.0),
						EventLog:           tt.mockEventLog,
					}, nil
				},
			}

			client := &Client{
				appConfig:       mockClient,
				PollingInterval: 100 * time.Millisecond, // Fast polling for tests
			}
			err := client.WaitForDeploymentPhase(
				context.Background(),
				"app-123",
				"env-123",
				tt.deploymentNum,
				true, // waitForBaking=true matches the legacy WaitForDeployment behavior
				tt.timeout,
				nil,
			)

			if (err != nil) != tt.wantErr {
				t.Errorf("WaitForDeploymentPhase() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErrMsg != "" && err != nil {
				if err.Error() != tt.wantErrMsg {
					t.Errorf("WaitForDeploymentPhase() error message = %q, want %q", err.Error(), tt.wantErrMsg)
				}
			}
		})
	}
}

func TestGetLatestDeployment(t *testing.T) {
	t.Parallel()
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
					ConfigurationProfileId: new("profile-123"),
					ConfigurationVersion:   new("5"),
					State:                  types.DeploymentStateComplete,
					Description:            new("test deployment"),
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
					ConfigurationProfileId: new("profile-123"),
					ConfigurationVersion:   new("5"),
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
					ConfigurationProfileId: new(profileID),
					ConfigurationVersion:   new("5"),
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
					ConfigurationProfileId: new("profile-123"),
					ConfigurationVersion:   new(configVersion),
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

			client := &Client{appConfig: mockClient}
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

// TestGetLatestDeployment_AllGetDeploymentFail covers the failure-mode added
// when GetDeployment returns errors for every deployment in the list. Before
// the fix the loop silently swallowed every error and returned (nil, nil),
// which callers (pull / edit / run skip-unchanged) would then misread as
// "no prior deployment". The new behavior surfaces the underlying AWS error
// so the user sees a concrete failure instead.
func TestGetLatestDeployment_AllGetDeploymentFail(t *testing.T) {
	t.Parallel()
	mockClient := &mock.MockAppConfigClient{
		ListDeploymentsFunc: func(ctx context.Context, params *appconfig.ListDeploymentsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListDeploymentsOutput, error) {
			return &appconfig.ListDeploymentsOutput{
				Items: []types.DeploymentSummary{
					{DeploymentNumber: 1},
					{DeploymentNumber: 2},
				},
			}, nil
		},
		GetDeploymentFunc: func(ctx context.Context, params *appconfig.GetDeploymentInput, optFns ...func(*appconfig.Options)) (*appconfig.GetDeploymentOutput, error) {
			return nil, errors.New("AccessDeniedException: caller not allowed to GetDeployment")
		},
	}

	client := &Client{appConfig: mockClient}
	deployment, err := GetLatestDeployment(context.Background(), client, "app-123", "env-123", "profile-123")
	if err == nil {
		t.Fatal("expected error when every GetDeployment fails, got nil")
	}
	if deployment != nil {
		t.Errorf("expected nil deployment on full failure, got %+v", deployment)
	}
	if !strings.Contains(err.Error(), "AccessDeniedException") {
		t.Errorf("expected wrapped underlying AWS error, got %q", err.Error())
	}
}

// TestGetLatestDeployment_PartialGetDeploymentFail covers the case where some
// GetDeployment calls fail but others succeed: the function should still find
// and return the latest deployment from the successful subset, not error out.
func TestGetLatestDeployment_PartialGetDeploymentFail(t *testing.T) {
	t.Parallel()
	mockClient := &mock.MockAppConfigClient{
		ListDeploymentsFunc: func(ctx context.Context, params *appconfig.ListDeploymentsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListDeploymentsOutput, error) {
			return &appconfig.ListDeploymentsOutput{
				Items: []types.DeploymentSummary{
					{DeploymentNumber: 1},
					{DeploymentNumber: 2},
				},
			}, nil
		},
		GetDeploymentFunc: func(ctx context.Context, params *appconfig.GetDeploymentInput, optFns ...func(*appconfig.Options)) (*appconfig.GetDeploymentOutput, error) {
			if *params.DeploymentNumber == 1 {
				return nil, errors.New("transient throttle on deployment 1")
			}
			return &appconfig.GetDeploymentOutput{
				DeploymentNumber:       *params.DeploymentNumber,
				ConfigurationProfileId: new("profile-123"),
				ConfigurationVersion:   new("9"),
				State:                  types.DeploymentStateComplete,
			}, nil
		},
	}

	client := &Client{appConfig: mockClient}
	deployment, err := GetLatestDeployment(context.Background(), client, "app-123", "env-123", "profile-123")
	if err != nil {
		t.Fatalf("expected partial failure to be tolerated, got %v", err)
	}
	if deployment == nil {
		t.Fatal("expected deployment from successful subset, got nil")
	}
	if deployment.DeploymentNumber != 2 {
		t.Errorf("DeploymentNumber = %d, want 2", deployment.DeploymentNumber)
	}
}

func TestGetLatestDeploymentIncludingRollback(t *testing.T) {
	t.Parallel()
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
					ConfigurationProfileId: new("profile-123"),
					ConfigurationVersion:   new(configVersion),
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
					ConfigurationProfileId: new("profile-123"),
					ConfigurationVersion:   new("10"),
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

			client := &Client{appConfig: mockClient}
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
	t.Parallel()
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
					ContentType: new("application/json"),
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

			client := &Client{appConfig: mockClient}
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

func TestWaitForDeploymentPhase(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		deploymentNum int32
		waitForBaking bool
		mockStates    []types.DeploymentState
		mockEventLog  []types.DeploymentEvent
		timeout       time.Duration
		wantErr       bool
		wantErrMsg    string
	}{
		{
			name:          "wait for deploy: completes when entering BAKING",
			deploymentNum: 1,
			waitForBaking: false,
			mockStates:    []types.DeploymentState{types.DeploymentStateDeploying, types.DeploymentStateBaking},
			timeout:       10 * time.Second,
			wantErr:       false,
		},
		{
			name:          "wait for deploy: already in BAKING",
			deploymentNum: 2,
			waitForBaking: false,
			mockStates:    []types.DeploymentState{types.DeploymentStateBaking},
			timeout:       10 * time.Second,
			wantErr:       false,
		},
		{
			name:          "wait for deploy: already COMPLETE",
			deploymentNum: 3,
			waitForBaking: false,
			mockStates:    []types.DeploymentState{types.DeploymentStateComplete},
			timeout:       10 * time.Second,
			wantErr:       false,
		},
		{
			name:          "wait for deploy: rolled back during DEPLOYING",
			deploymentNum: 4,
			waitForBaking: false,
			mockStates:    []types.DeploymentState{types.DeploymentStateDeploying, types.DeploymentStateRolledBack},
			timeout:       10 * time.Second,
			wantErr:       true,
			wantErrMsg:    "deployment was rolled back",
		},
		{
			name:          "wait for bake: completes when COMPLETE",
			deploymentNum: 5,
			waitForBaking: true,
			mockStates:    []types.DeploymentState{types.DeploymentStateDeploying, types.DeploymentStateBaking, types.DeploymentStateComplete},
			timeout:       10 * time.Second,
			wantErr:       false,
		},
		{
			name:          "wait for bake: already COMPLETE",
			deploymentNum: 6,
			waitForBaking: true,
			mockStates:    []types.DeploymentState{types.DeploymentStateComplete},
			timeout:       10 * time.Second,
			wantErr:       false,
		},
		{
			name:          "wait for bake: rolled back during BAKING",
			deploymentNum: 7,
			waitForBaking: true,
			mockStates:    []types.DeploymentState{types.DeploymentStateBaking, types.DeploymentStateRolledBack},
			mockEventLog: []types.DeploymentEvent{
				{
					EventType:   types.DeploymentEventTypeRollbackStarted,
					Description: new("CloudWatch alarm triggered"),
				},
			},
			timeout:    10 * time.Second,
			wantErr:    true,
			wantErrMsg: "deployment was rolled back: CloudWatch alarm triggered",
		},
		{
			name:          "wait for deploy: timeout",
			deploymentNum: 8,
			waitForBaking: false,
			mockStates:    []types.DeploymentState{types.DeploymentStateDeploying, types.DeploymentStateDeploying},
			timeout:       1 * time.Second,
			wantErr:       true,
			wantErrMsg:    "deployment timed out",
		},
		{
			name:          "wait for bake: timeout",
			deploymentNum: 9,
			waitForBaking: true,
			mockStates:    []types.DeploymentState{types.DeploymentStateBaking, types.DeploymentStateBaking},
			timeout:       1 * time.Second,
			wantErr:       true,
			wantErrMsg:    "deployment timed out",
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
						PercentageComplete: new(float32(callCount) * 30.0),
						EventLog:           tt.mockEventLog,
					}, nil
				},
			}

			client := &Client{
				appConfig:       mockClient,
				PollingInterval: 100 * time.Millisecond, // Fast polling for tests
			}
			err := client.WaitForDeploymentPhase(
				context.Background(),
				"app-123",
				"env-123",
				tt.deploymentNum,
				tt.waitForBaking,
				tt.timeout,
				nil,
			)

			if (err != nil) != tt.wantErr {
				t.Errorf("WaitForDeploymentPhase() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErrMsg != "" && err != nil {
				if !strings.Contains(err.Error(), tt.wantErrMsg) {
					t.Errorf("WaitForDeploymentPhase() error message = %q, want to contain %q", err.Error(), tt.wantErrMsg)
				}
			}
		})
	}
}

// Test that WaitForDeploymentPhase with waitForBaking=false stops at BAKING
func TestWaitForDeploymentPhase_StopsAtBaking(t *testing.T) {
	t.Parallel()
	callCount := 0
	mockClient := &mock.MockAppConfigClient{
		GetDeploymentFunc: func(ctx context.Context, params *appconfig.GetDeploymentInput, optFns ...func(*appconfig.Options)) (*appconfig.GetDeploymentOutput, error) {
			callCount++
			// Deployment goes: DEPLOYING -> BAKING -> COMPLETE
			// But we should stop at BAKING when waitForBaking=false
			states := []types.DeploymentState{
				types.DeploymentStateDeploying,
				types.DeploymentStateBaking,
				types.DeploymentStateComplete,
			}
			var state types.DeploymentState
			if callCount-1 < len(states) {
				state = states[callCount-1]
			} else {
				state = types.DeploymentStateComplete
			}

			return &appconfig.GetDeploymentOutput{
				DeploymentNumber: 1,
				State:            state,
			}, nil
		},
	}

	client := &Client{
		appConfig:       mockClient,
		PollingInterval: 50 * time.Millisecond,
	}

	err := client.WaitForDeploymentPhase(
		context.Background(),
		"app-123",
		"env-123",
		1,
		false, // waitForBaking=false
		5*time.Second,
		nil,
	)
	if err != nil {
		t.Errorf("WaitForDeploymentPhase() unexpected error: %v", err)
	}

	// Should have called GetDeployment exactly 2 times (DEPLOYING, then BAKING)
	// and stopped at BAKING without checking COMPLETE
	if callCount > 2 {
		t.Errorf("Expected to stop at BAKING (2 calls), but made %d calls", callCount)
	}
}

func TestStopDeployment(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		applicationID    string
		environmentID    string
		deploymentNumber int32
		mockFunc         func(ctx context.Context, params *appconfig.StopDeploymentInput, optFns ...func(*appconfig.Options)) (*appconfig.StopDeploymentOutput, error)
		wantErr          bool
		errContains      string
	}{
		{
			name:             "successful stop deployment",
			applicationID:    "app-123",
			environmentID:    "env-123",
			deploymentNumber: 1,
			mockFunc: func(ctx context.Context, params *appconfig.StopDeploymentInput, optFns ...func(*appconfig.Options)) (*appconfig.StopDeploymentOutput, error) {
				return &appconfig.StopDeploymentOutput{}, nil
			},
			wantErr: false,
		},
		{
			name:             "API error",
			applicationID:    "app-123",
			environmentID:    "env-123",
			deploymentNumber: 1,
			mockFunc: func(ctx context.Context, params *appconfig.StopDeploymentInput, optFns ...func(*appconfig.Options)) (*appconfig.StopDeploymentOutput, error) {
				return nil, errors.New("API error")
			},
			wantErr:     true,
			errContains: "failed to stop deployment",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockClient := &mock.MockAppConfigClient{
				StopDeploymentFunc: tt.mockFunc,
			}

			client := &Client{appConfig: mockClient}
			err := client.StopDeployment(
				context.Background(),
				tt.applicationID,
				tt.environmentID,
				tt.deploymentNumber,
			)

			if (err != nil) != tt.wantErr {
				t.Errorf("StopDeployment() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && err != nil && tt.errContains != "" {
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("StopDeployment() error = %v, should contain %v", err, tt.errContains)
				}
			}
		})
	}
}

func TestExtractRollbackReason(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		eventLog []types.DeploymentEvent
		want     string
	}{
		{
			name: "rollback with CloudWatch alarm",
			eventLog: []types.DeploymentEvent{
				{EventType: types.DeploymentEventTypeRollbackStarted, Description: new("Rollback initiated by CloudWatch Alarm")},
			},
			want: "Rollback initiated by CloudWatch Alarm",
		},
		{
			name: "rollback completed event",
			eventLog: []types.DeploymentEvent{
				{EventType: types.DeploymentEventTypeRollbackCompleted, Description: new("Rollback completed successfully")},
			},
			want: "Rollback completed successfully",
		},
		{
			name: "no rollback events",
			eventLog: []types.DeploymentEvent{
				{EventType: types.DeploymentEventTypeDeploymentStarted, Description: new("Deployment started")},
			},
			want: "",
		},
		{
			name:     "empty event log",
			eventLog: []types.DeploymentEvent{},
			want:     "",
		},
		{
			name: "rollback event without description",
			eventLog: []types.DeploymentEvent{
				{EventType: types.DeploymentEventTypeRollbackStarted, Description: nil},
			},
			want: "",
		},
		{
			name: "multiple events, get most recent rollback",
			eventLog: []types.DeploymentEvent{
				{EventType: types.DeploymentEventTypeDeploymentStarted, Description: new("Deployment started")},
				{EventType: types.DeploymentEventTypeRollbackStarted, Description: new("First rollback")},
				{EventType: types.DeploymentEventTypeRollbackStarted, Description: new("Second rollback")},
			},
			want: "Second rollback",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := ExtractRollbackReason(tt.eventLog); got != tt.want {
				t.Errorf("ExtractRollbackReason() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestWaitForBakingComplete covers the bake-only wait helper used by run/edit
// when --wait-bake is set. It verifies state-machine handling (BAKING →
// COMPLETE / rollback / unexpected state / timeout) and that the bake tick
// callback receives the configured FinalBakeTimeInMinutes as the total
// duration, with elapsed advancing across ticks.
func TestWaitForBakingComplete(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string
		mockStates     []types.DeploymentState
		mockEventLog   []types.DeploymentEvent
		bakeMinutes    int32
		timeout        time.Duration
		wantErr        bool
		wantErrMsg     string
		wantTickCalled bool
	}{
		{
			name:           "completes immediately",
			mockStates:     []types.DeploymentState{types.DeploymentStateComplete},
			bakeMinutes:    1,
			timeout:        2 * time.Second,
			wantTickCalled: true,
		},
		{
			name:           "baking then complete",
			mockStates:     []types.DeploymentState{types.DeploymentStateBaking, types.DeploymentStateComplete},
			bakeMinutes:    1,
			timeout:        2 * time.Second,
			wantTickCalled: true,
		},
		{
			name:           "zero bake duration still completes",
			mockStates:     []types.DeploymentState{types.DeploymentStateBaking, types.DeploymentStateComplete},
			bakeMinutes:    0,
			timeout:        2 * time.Second,
			wantTickCalled: true,
		},
		{
			name:         "rolled back during bake surfaces reason",
			mockStates:   []types.DeploymentState{types.DeploymentStateBaking, types.DeploymentStateRolledBack},
			mockEventLog: []types.DeploymentEvent{{EventType: types.DeploymentEventTypeRollbackStarted, Description: new("alarm")}},
			bakeMinutes:  1,
			timeout:      2 * time.Second,
			wantErr:      true,
			wantErrMsg:   "deployment was rolled back: alarm",
		},
		{
			name:        "rolled back without description",
			mockStates:  []types.DeploymentState{types.DeploymentStateRolledBack},
			bakeMinutes: 1,
			timeout:     2 * time.Second,
			wantErr:     true,
			wantErrMsg:  "deployment was rolled back",
		},
		{
			name:        "unexpected DEPLOYING state errors out",
			mockStates:  []types.DeploymentState{types.DeploymentStateDeploying},
			bakeMinutes: 1,
			timeout:     2 * time.Second,
			wantErr:     true,
			wantErrMsg:  "unexpected deployment state during bake wait",
		},
		{
			name:        "timeout while still baking",
			mockStates:  []types.DeploymentState{types.DeploymentStateBaking, types.DeploymentStateBaking},
			bakeMinutes: 1,
			timeout:     500 * time.Millisecond,
			wantErr:     true,
			wantErrMsg:  "bake phase timed out",
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
						DeploymentNumber:       1,
						State:                  state,
						FinalBakeTimeInMinutes: tt.bakeMinutes,
						EventLog:               tt.mockEventLog,
					}, nil
				},
			}

			client := &Client{
				appConfig:       mockClient,
				PollingInterval: 50 * time.Millisecond,
			}

			var lastElapsed time.Duration
			var lastTotal time.Duration
			var ticked bool
			tick := func(elapsed, total time.Duration) {
				lastElapsed = elapsed
				lastTotal = total
				ticked = true
			}

			err := client.WaitForBakingComplete(
				context.Background(),
				"app-123",
				"env-123",
				1,
				tt.timeout,
				tick,
			)

			if (err != nil) != tt.wantErr {
				t.Fatalf("WaitForBakingComplete() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErrMsg != "" && err != nil {
				if !strings.Contains(err.Error(), tt.wantErrMsg) {
					t.Errorf("WaitForBakingComplete() error = %q, want contains %q", err.Error(), tt.wantErrMsg)
				}
			}
			if tt.wantTickCalled && !ticked {
				t.Errorf("expected tick to be invoked at least once")
			}
			if tt.wantTickCalled {
				wantTotal := time.Duration(tt.bakeMinutes) * time.Minute
				if lastTotal != wantTotal {
					t.Errorf("final tick total = %v, want %v", lastTotal, wantTotal)
				}
				if lastElapsed < 0 {
					t.Errorf("final tick elapsed = %v, want >= 0", lastElapsed)
				}
			}
		})
	}
}

// TestWaitForBakingComplete_TickElapsed verifies that elapsed time
// monotonically increases across ticks while BAKING, and the final
// COMPLETE tick reports the full bake duration so callers can render a
// definitive "done" state.
func TestWaitForBakingComplete_TickElapsed(t *testing.T) {
	t.Parallel()
	calls := 0
	mockClient := &mock.MockAppConfigClient{
		GetDeploymentFunc: func(ctx context.Context, params *appconfig.GetDeploymentInput, optFns ...func(*appconfig.Options)) (*appconfig.GetDeploymentOutput, error) {
			calls++
			state := types.DeploymentStateBaking
			if calls >= 3 {
				state = types.DeploymentStateComplete
			}
			return &appconfig.GetDeploymentOutput{
				DeploymentNumber:       1,
				State:                  state,
				FinalBakeTimeInMinutes: 60,
			}, nil
		},
	}

	client := &Client{
		appConfig:       mockClient,
		PollingInterval: 30 * time.Millisecond,
	}

	type tickRecord struct {
		elapsed time.Duration
		total   time.Duration
	}
	var ticks []tickRecord
	err := client.WaitForBakingComplete(
		context.Background(),
		"app-123",
		"env-123",
		1,
		2*time.Second,
		func(elapsed, total time.Duration) {
			ticks = append(ticks, tickRecord{elapsed: elapsed, total: total})
		},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ticks) < 3 {
		t.Fatalf("expected at least 3 ticks, got %d", len(ticks))
	}
	wantTotal := 60 * time.Minute
	for i, tk := range ticks {
		if tk.total != wantTotal {
			t.Errorf("tick %d total = %v, want %v", i, tk.total, wantTotal)
		}
	}
	for i := 1; i < len(ticks); i++ {
		if ticks[i].elapsed < ticks[i-1].elapsed {
			t.Errorf("tick %d elapsed = %v not monotonic with previous %v", i, ticks[i].elapsed, ticks[i-1].elapsed)
		}
	}
	if ticks[len(ticks)-1].elapsed != wantTotal {
		t.Errorf("final tick elapsed = %v, want %v (full bake duration)", ticks[len(ticks)-1].elapsed, wantTotal)
	}
}

func TestGetDeploymentDetails(t *testing.T) {
	t.Parallel()

	startedAt := time.Date(2024, 1, 2, 10, 0, 0, 0, time.UTC)
	completedAt := time.Date(2024, 1, 2, 10, 12, 0, 0, time.UTC)

	tests := []struct {
		name        string
		mockFunc    func(ctx context.Context, params *appconfig.GetDeploymentInput, optFns ...func(*appconfig.Options)) (*appconfig.GetDeploymentOutput, error)
		wantErr     bool
		errContains string
		check       func(t *testing.T, d *DeploymentDetails)
	}{
		{
			name: "returns details with all fields populated",
			mockFunc: func(_ context.Context, params *appconfig.GetDeploymentInput, _ ...func(*appconfig.Options)) (*appconfig.GetDeploymentOutput, error) {
				if aws.ToString(params.ApplicationId) != "app-123" {
					t.Errorf("ApplicationId = %q, want %q", aws.ToString(params.ApplicationId), "app-123")
				}
				if aws.ToString(params.EnvironmentId) != "env-123" {
					t.Errorf("EnvironmentId = %q, want %q", aws.ToString(params.EnvironmentId), "env-123")
				}
				if params.DeploymentNumber == nil || *params.DeploymentNumber != 7 {
					t.Errorf("DeploymentNumber = %v, want 7", params.DeploymentNumber)
				}
				return &appconfig.GetDeploymentOutput{
					DeploymentNumber:            7,
					ConfigurationProfileId:      aws.String("profile-123"),
					ConfigurationVersion:        aws.String("3"),
					DeploymentStrategyId:        aws.String("strategy-123"),
					State:                       types.DeploymentStateComplete,
					Description:                 aws.String("hotfix"),
					StartedAt:                   &startedAt,
					CompletedAt:                 &completedAt,
					PercentageComplete:          aws.Float32(100),
					GrowthFactor:                aws.Float32(20),
					DeploymentDurationInMinutes: 30,
					FinalBakeTimeInMinutes:      10,
				}, nil
			},
			check: func(t *testing.T, d *DeploymentDetails) {
				if d.DeploymentNumber != 7 {
					t.Errorf("DeploymentNumber = %d, want 7", d.DeploymentNumber)
				}
				if d.ConfigurationProfileID != "profile-123" {
					t.Errorf("ConfigurationProfileID = %q, want %q", d.ConfigurationProfileID, "profile-123")
				}
				if d.State != types.DeploymentStateComplete {
					t.Errorf("State = %q, want %q", d.State, types.DeploymentStateComplete)
				}
				if d.Description != "hotfix" {
					t.Errorf("Description = %q, want %q", d.Description, "hotfix")
				}
				if d.StartedAt == nil || !d.StartedAt.Equal(startedAt) {
					t.Errorf("StartedAt = %v, want %v", d.StartedAt, startedAt)
				}
				if d.CompletedAt == nil || !d.CompletedAt.Equal(completedAt) {
					t.Errorf("CompletedAt = %v, want %v", d.CompletedAt, completedAt)
				}
				if d.PercentageComplete != 100 {
					t.Errorf("PercentageComplete = %v, want 100", d.PercentageComplete)
				}
				if d.GrowthFactor != 20 {
					t.Errorf("GrowthFactor = %v, want 20", d.GrowthFactor)
				}
				if d.DeploymentDurationInMinutes != 30 {
					t.Errorf("DeploymentDurationInMinutes = %d, want 30", d.DeploymentDurationInMinutes)
				}
				if d.FinalBakeTimeInMinutes != 10 {
					t.Errorf("FinalBakeTimeInMinutes = %d, want 10", d.FinalBakeTimeInMinutes)
				}
			},
		},
		{
			name: "nil pointer fields default to zero values",
			mockFunc: func(_ context.Context, _ *appconfig.GetDeploymentInput, _ ...func(*appconfig.Options)) (*appconfig.GetDeploymentOutput, error) {
				return &appconfig.GetDeploymentOutput{
					DeploymentNumber: 1,
					State:            types.DeploymentStateDeploying,
				}, nil
			},
			check: func(t *testing.T, d *DeploymentDetails) {
				if d.PercentageComplete != 0 {
					t.Errorf("PercentageComplete = %v, want 0", d.PercentageComplete)
				}
				if d.GrowthFactor != 0 {
					t.Errorf("GrowthFactor = %v, want 0", d.GrowthFactor)
				}
				if d.ConfigurationProfileID != "" {
					t.Errorf("ConfigurationProfileID = %q, want empty", d.ConfigurationProfileID)
				}
			},
		},
		{
			name: "GetDeployment error is wrapped",
			mockFunc: func(_ context.Context, _ *appconfig.GetDeploymentInput, _ ...func(*appconfig.Options)) (*appconfig.GetDeploymentOutput, error) {
				return nil, errors.New("api boom")
			},
			wantErr:     true,
			errContains: "failed to get deployment details",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := &mock.MockAppConfigClient{GetDeploymentFunc: tt.mockFunc}
			client := NewTestClient(m)
			got, err := GetDeploymentDetails(context.Background(), client, "app-123", "env-123", 7)
			if (err != nil) != tt.wantErr {
				t.Fatalf("GetDeploymentDetails() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error = %v, want to contain %q", err, tt.errContains)
				}
				return
			}
			if tt.check != nil {
				tt.check(t, got)
			}
		})
	}
}
