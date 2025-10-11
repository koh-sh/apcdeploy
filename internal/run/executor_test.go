package run

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/appconfig"
	"github.com/aws/aws-sdk-go-v2/service/appconfig/types"
	awsInternal "github.com/koh-sh/apcdeploy/internal/aws"
	"github.com/koh-sh/apcdeploy/internal/aws/mock"
	"github.com/koh-sh/apcdeploy/internal/config"
	reportertest "github.com/koh-sh/apcdeploy/internal/reporter/testing"
)

func TestNewExecutor(t *testing.T) {
	reporter := &reportertest.MockReporter{}
	executor := NewExecutor(reporter)

	if executor == nil {
		t.Fatal("expected executor to be non-nil")
		return
	}

	if executor.reporter != reporter {
		t.Error("expected executor to have the provided reporter")
	}
}

func TestExecutorValidateTimeout(t *testing.T) {
	tests := []struct {
		name        string
		timeout     int
		wantErr     bool
		expectedMsg string
	}{
		{
			name:        "negative timeout is invalid",
			timeout:     -1,
			wantErr:     true,
			expectedMsg: "timeout must be a positive value",
		},
		{
			name:    "zero timeout is valid",
			timeout: 0,
			wantErr: false,
		},
		{
			name:    "positive timeout is valid",
			timeout: 300,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reporter := &reportertest.MockReporter{}
			executor := NewExecutor(reporter)

			opts := &Options{
				ConfigFile: "nonexistent.yml",
				WaitDeploy: false,
				WaitBake:   false,
				Timeout:    tt.timeout,
			}

			err := executor.Execute(context.Background(), opts)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error for negative timeout, got nil")
				} else if !strings.Contains(err.Error(), tt.expectedMsg) {
					t.Errorf("expected error containing %q, got %q", tt.expectedMsg, err.Error())
				}
			} else {
				// We expect an error here because the config file doesn't exist
				// but it should not be a timeout validation error
				if err != nil && strings.Contains(err.Error(), "timeout must be a positive value") {
					t.Errorf("unexpected timeout validation error: %v", err)
				}
			}
		})
	}
}

func TestExecutorValidateWaitFlags(t *testing.T) {
	reporter := &reportertest.MockReporter{}
	executor := NewExecutor(reporter)

	opts := &Options{
		ConfigFile: "nonexistent.yml",
		WaitDeploy: true,
		WaitBake:   true,
		Timeout:    300,
	}

	err := executor.Execute(context.Background(), opts)

	if err == nil {
		t.Error("expected error when both --wait-deploy and --wait-bake are specified")
		return
	}

	if !strings.Contains(err.Error(), "--wait-deploy and --wait-bake cannot be used together") {
		t.Errorf("expected error about mutually exclusive flags, got: %v", err)
	}
}

func TestExecutorLoadConfigurationError(t *testing.T) {
	reporter := &reportertest.MockReporter{}
	executor := NewExecutor(reporter)

	opts := &Options{
		ConfigFile: "nonexistent.yml",
		WaitDeploy: false,
		WaitBake:   false,
		Timeout:    300,
	}

	err := executor.Execute(context.Background(), opts)

	if err == nil {
		t.Error("expected error when loading non-existent config file")
	}

	if !strings.Contains(err.Error(), "failed to load configuration") {
		t.Errorf("expected 'failed to load configuration' error, got: %v", err)
	}

	// Verify reporter was called for progress
	if len(reporter.Messages) == 0 {
		t.Error("expected reporter to have received messages")
	}

	if !strings.Contains(reporter.Messages[0], "Loading configuration") {
		t.Errorf("expected first message to be about loading configuration, got: %v", reporter.Messages[0])
	}
}

// TestExecutorFullWorkflowWithMock tests the complete deployment workflow with mocked AWS
func TestExecutorFullWorkflowWithMock(t *testing.T) {
	// Create temporary test files
	tempDir, err := os.MkdirTemp("", "executor-full-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create config file
	configPath := filepath.Join(tempDir, "apcdeploy.yml")
	configContent := `application: test-app
configuration_profile: test-profile
environment: test-env
deployment_strategy: AppConfig.AllAtOnce
data_file: data.json
region: us-east-1
`
	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	// Create data file
	dataPath := filepath.Join(tempDir, "data.json")
	if err := os.WriteFile(dataPath, []byte(`{"key": "value"}`), 0o644); err != nil {
		t.Fatalf("Failed to write data: %v", err)
	}

	// Create mock AWS client
	mockClient := &mock.MockAppConfigClient{
		ListApplicationsFunc: func(ctx context.Context, params *appconfig.ListApplicationsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListApplicationsOutput, error) {
			return &appconfig.ListApplicationsOutput{
				Items: []types.Application{
					{
						Id:   aws.String("app-123"),
						Name: aws.String("test-app"),
					},
				},
			}, nil
		},
		ListConfigurationProfilesFunc: func(ctx context.Context, params *appconfig.ListConfigurationProfilesInput, optFns ...func(*appconfig.Options)) (*appconfig.ListConfigurationProfilesOutput, error) {
			return &appconfig.ListConfigurationProfilesOutput{
				Items: []types.ConfigurationProfileSummary{
					{
						Id:   aws.String("profile-123"),
						Name: aws.String("test-profile"),
						Type: aws.String("AWS.Freeform"),
					},
				},
			}, nil
		},
		GetConfigurationProfileFunc: func(ctx context.Context, params *appconfig.GetConfigurationProfileInput, optFns ...func(*appconfig.Options)) (*appconfig.GetConfigurationProfileOutput, error) {
			return &appconfig.GetConfigurationProfileOutput{
				Id:   aws.String("profile-123"),
				Name: aws.String("test-profile"),
				Type: aws.String("AWS.Freeform"),
			}, nil
		},
		ListEnvironmentsFunc: func(ctx context.Context, params *appconfig.ListEnvironmentsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListEnvironmentsOutput, error) {
			return &appconfig.ListEnvironmentsOutput{
				Items: []types.Environment{
					{
						Id:   aws.String("env-123"),
						Name: aws.String("test-env"),
					},
				},
			}, nil
		},
		ListDeploymentStrategiesFunc: func(ctx context.Context, params *appconfig.ListDeploymentStrategiesInput, optFns ...func(*appconfig.Options)) (*appconfig.ListDeploymentStrategiesOutput, error) {
			return &appconfig.ListDeploymentStrategiesOutput{
				Items: []types.DeploymentStrategy{
					{
						Id:   aws.String("strategy-123"),
						Name: aws.String("AppConfig.AllAtOnce"),
					},
				},
			}, nil
		},
		ListDeploymentsFunc: func(ctx context.Context, params *appconfig.ListDeploymentsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListDeploymentsOutput, error) {
			return &appconfig.ListDeploymentsOutput{
				Items: []types.DeploymentSummary{},
			}, nil
		},
		CreateHostedConfigurationVersionFunc: func(ctx context.Context, params *appconfig.CreateHostedConfigurationVersionInput, optFns ...func(*appconfig.Options)) (*appconfig.CreateHostedConfigurationVersionOutput, error) {
			return &appconfig.CreateHostedConfigurationVersionOutput{
				VersionNumber: 1,
			}, nil
		},
		StartDeploymentFunc: func(ctx context.Context, params *appconfig.StartDeploymentInput, optFns ...func(*appconfig.Options)) (*appconfig.StartDeploymentOutput, error) {
			return &appconfig.StartDeploymentOutput{
				DeploymentNumber: 1,
			}, nil
		},
		GetDeploymentFunc: func(ctx context.Context, params *appconfig.GetDeploymentInput, optFns ...func(*appconfig.Options)) (*appconfig.GetDeploymentOutput, error) {
			return &appconfig.GetDeploymentOutput{
				State: types.DeploymentStateComplete,
			}, nil
		},
	}

	// Create deployer factory that uses the mock client
	deployerFactory := func(ctx context.Context, cfg *config.Config) (*Deployer, error) {
		awsClient := &awsInternal.Client{
			AppConfig: mockClient,
		}
		return NewWithClient(cfg, awsClient), nil
	}

	reporter := &reportertest.MockReporter{}
	executor := NewExecutorWithFactory(reporter, deployerFactory)

	opts := &Options{
		ConfigFile: configPath,
		WaitDeploy: false,
		WaitBake:   false,
		Timeout:    300,
	}

	err = executor.Execute(context.Background(), opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify all expected messages were reported
	expectedMessages := []string{
		"Loading configuration",
		"Configuration loaded",
		"Resolving AWS resources",
		"Resolved resources",
		"Checking for ongoing deployments",
		"No ongoing deployments",
		"Validating configuration data",
		"Configuration data validated",
		"Creating configuration version",
		"Created configuration version 1",
		"Starting deployment",
		"Deployment #1 started",
		"Deployment #1 is in progress",
	}

	for _, expected := range expectedMessages {
		found := false
		for _, msg := range reporter.Messages {
			if strings.Contains(msg, expected) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected message containing %q not found in: %v", expected, reporter.Messages)
		}
	}
}

// TestExecutorFullWorkflowWithWait tests deployment with wait options
func TestExecutorFullWorkflowWithWait(t *testing.T) {
	tests := []struct {
		name       string
		waitDeploy bool
		waitBake   bool
		mockStates []types.DeploymentState
		wantMsg    string
	}{
		{
			name:       "wait for bake: immediate completion",
			waitDeploy: false,
			waitBake:   true,
			mockStates: []types.DeploymentState{types.DeploymentStateComplete},
			wantMsg:    "Deployment completed successfully",
		},
		{
			name:       "wait for bake: completion after polling",
			waitDeploy: false,
			waitBake:   true,
			mockStates: []types.DeploymentState{types.DeploymentStateDeploying, types.DeploymentStateBaking, types.DeploymentStateComplete},
			wantMsg:    "Deployment completed successfully",
		},
		{
			name:       "wait for deploy: stops at baking",
			waitDeploy: true,
			waitBake:   false,
			mockStates: []types.DeploymentState{types.DeploymentStateDeploying, types.DeploymentStateBaking},
			wantMsg:    "Deployment phase completed (now baking)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary test files
			tempDir, err := os.MkdirTemp("", "executor-wait-*")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tempDir)

			configPath := filepath.Join(tempDir, "apcdeploy.yml")
			configContent := `application: test-app
configuration_profile: test-profile
environment: test-env
deployment_strategy: AppConfig.AllAtOnce
data_file: data.json
region: us-east-1
`
			if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
				t.Fatalf("Failed to write config: %v", err)
			}

			dataPath := filepath.Join(tempDir, "data.json")
			if err := os.WriteFile(dataPath, []byte(`{"key": "value"}`), 0o644); err != nil {
				t.Fatalf("Failed to write data: %v", err)
			}

			callCount := 0
			mockClient := &mock.MockAppConfigClient{
				ListApplicationsFunc: func(ctx context.Context, params *appconfig.ListApplicationsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListApplicationsOutput, error) {
					return &appconfig.ListApplicationsOutput{
						Items: []types.Application{{Id: aws.String("app-123"), Name: aws.String("test-app")}},
					}, nil
				},
				ListConfigurationProfilesFunc: func(ctx context.Context, params *appconfig.ListConfigurationProfilesInput, optFns ...func(*appconfig.Options)) (*appconfig.ListConfigurationProfilesOutput, error) {
					return &appconfig.ListConfigurationProfilesOutput{
						Items: []types.ConfigurationProfileSummary{{Id: aws.String("profile-123"), Name: aws.String("test-profile"), Type: aws.String("AWS.Freeform")}},
					}, nil
				},
				GetConfigurationProfileFunc: func(ctx context.Context, params *appconfig.GetConfigurationProfileInput, optFns ...func(*appconfig.Options)) (*appconfig.GetConfigurationProfileOutput, error) {
					return &appconfig.GetConfigurationProfileOutput{Id: aws.String("profile-123"), Type: aws.String("AWS.Freeform")}, nil
				},
				ListEnvironmentsFunc: func(ctx context.Context, params *appconfig.ListEnvironmentsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListEnvironmentsOutput, error) {
					return &appconfig.ListEnvironmentsOutput{
						Items: []types.Environment{{Id: aws.String("env-123"), Name: aws.String("test-env")}},
					}, nil
				},
				ListDeploymentStrategiesFunc: func(ctx context.Context, params *appconfig.ListDeploymentStrategiesInput, optFns ...func(*appconfig.Options)) (*appconfig.ListDeploymentStrategiesOutput, error) {
					return &appconfig.ListDeploymentStrategiesOutput{
						Items: []types.DeploymentStrategy{{Id: aws.String("strategy-123"), Name: aws.String("AppConfig.AllAtOnce")}},
					}, nil
				},
				ListDeploymentsFunc: func(ctx context.Context, params *appconfig.ListDeploymentsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListDeploymentsOutput, error) {
					return &appconfig.ListDeploymentsOutput{Items: []types.DeploymentSummary{}}, nil
				},
				CreateHostedConfigurationVersionFunc: func(ctx context.Context, params *appconfig.CreateHostedConfigurationVersionInput, optFns ...func(*appconfig.Options)) (*appconfig.CreateHostedConfigurationVersionOutput, error) {
					return &appconfig.CreateHostedConfigurationVersionOutput{VersionNumber: 1}, nil
				},
				StartDeploymentFunc: func(ctx context.Context, params *appconfig.StartDeploymentInput, optFns ...func(*appconfig.Options)) (*appconfig.StartDeploymentOutput, error) {
					return &appconfig.StartDeploymentOutput{DeploymentNumber: 1}, nil
				},
				GetDeploymentFunc: func(ctx context.Context, params *appconfig.GetDeploymentInput, optFns ...func(*appconfig.Options)) (*appconfig.GetDeploymentOutput, error) {
					var state types.DeploymentState
					if callCount < len(tt.mockStates) {
						state = tt.mockStates[callCount]
					} else {
						state = tt.mockStates[len(tt.mockStates)-1]
					}
					callCount++
					return &appconfig.GetDeploymentOutput{State: state}, nil
				},
			}

			deployerFactory := func(ctx context.Context, cfg *config.Config) (*Deployer, error) {
				return NewWithClient(cfg, &awsInternal.Client{
					AppConfig:       mockClient,
					PollingInterval: 100 * time.Millisecond, // Fast polling for tests
				}), nil
			}

			reporter := &reportertest.MockReporter{}
			executor := NewExecutorWithFactory(reporter, deployerFactory)

			opts := &Options{
				ConfigFile: configPath,
				WaitDeploy: tt.waitDeploy,
				WaitBake:   tt.waitBake,
				Timeout:    30,
			}

			err = executor.Execute(context.Background(), opts)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Verify expected message
			hasExpectedMsg := false
			for _, msg := range reporter.Messages {
				if strings.Contains(msg, tt.wantMsg) {
					hasExpectedMsg = true
					break
				}
			}

			if !hasExpectedMsg {
				t.Errorf("expected message containing %q, got messages: %v", tt.wantMsg, reporter.Messages)
			}
		})
	}
}

// TestExecutorSkipsDeploymentWhenNoDiff tests that deployment is skipped when there are no changes
func TestExecutorSkipsDeploymentWhenNoDiff(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "executor-nodiff-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	configPath := filepath.Join(tempDir, "apcdeploy.yml")
	configContent := `application: test-app
configuration_profile: test-profile
environment: test-env
deployment_strategy: AppConfig.AllAtOnce
data_file: data.json
region: us-east-1
`
	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	// Create data file with same content as deployed version
	dataPath := filepath.Join(tempDir, "data.json")
	dataContent := []byte(`{"key": "value"}`)
	if err := os.WriteFile(dataPath, dataContent, 0o644); err != nil {
		t.Fatalf("Failed to write data: %v", err)
	}

	createVersionCalled := false
	startDeploymentCalled := false

	mockClient := &mock.MockAppConfigClient{
		ListApplicationsFunc: func(ctx context.Context, params *appconfig.ListApplicationsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListApplicationsOutput, error) {
			return &appconfig.ListApplicationsOutput{
				Items: []types.Application{{Id: aws.String("app-123"), Name: aws.String("test-app")}},
			}, nil
		},
		ListConfigurationProfilesFunc: func(ctx context.Context, params *appconfig.ListConfigurationProfilesInput, optFns ...func(*appconfig.Options)) (*appconfig.ListConfigurationProfilesOutput, error) {
			return &appconfig.ListConfigurationProfilesOutput{
				Items: []types.ConfigurationProfileSummary{{Id: aws.String("profile-123"), Name: aws.String("test-profile"), Type: aws.String("AWS.Freeform")}},
			}, nil
		},
		GetConfigurationProfileFunc: func(ctx context.Context, params *appconfig.GetConfigurationProfileInput, optFns ...func(*appconfig.Options)) (*appconfig.GetConfigurationProfileOutput, error) {
			return &appconfig.GetConfigurationProfileOutput{Id: aws.String("profile-123"), Type: aws.String("AWS.Freeform")}, nil
		},
		ListEnvironmentsFunc: func(ctx context.Context, params *appconfig.ListEnvironmentsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListEnvironmentsOutput, error) {
			return &appconfig.ListEnvironmentsOutput{
				Items: []types.Environment{{Id: aws.String("env-123"), Name: aws.String("test-env")}},
			}, nil
		},
		ListDeploymentStrategiesFunc: func(ctx context.Context, params *appconfig.ListDeploymentStrategiesInput, optFns ...func(*appconfig.Options)) (*appconfig.ListDeploymentStrategiesOutput, error) {
			return &appconfig.ListDeploymentStrategiesOutput{
				Items: []types.DeploymentStrategy{{Id: aws.String("strategy-123"), Name: aws.String("AppConfig.AllAtOnce")}},
			}, nil
		},
		ListDeploymentsFunc: func(ctx context.Context, params *appconfig.ListDeploymentsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListDeploymentsOutput, error) {
			// Return a completed deployment
			return &appconfig.ListDeploymentsOutput{
				Items: []types.DeploymentSummary{
					{
						DeploymentNumber:     1,
						State:                types.DeploymentStateComplete,
						ConfigurationVersion: aws.String("1"),
					},
				},
			}, nil
		},
		GetDeploymentFunc: func(ctx context.Context, params *appconfig.GetDeploymentInput, optFns ...func(*appconfig.Options)) (*appconfig.GetDeploymentOutput, error) {
			return &appconfig.GetDeploymentOutput{
				State:                  types.DeploymentStateComplete,
				ConfigurationProfileId: aws.String("profile-123"),
				ConfigurationVersion:   aws.String("1"),
			}, nil
		},
		GetHostedConfigurationVersionFunc: func(ctx context.Context, params *appconfig.GetHostedConfigurationVersionInput, optFns ...func(*appconfig.Options)) (*appconfig.GetHostedConfigurationVersionOutput, error) {
			// Return same content as local file (no diff)
			return &appconfig.GetHostedConfigurationVersionOutput{
				Content: dataContent,
			}, nil
		},
		CreateHostedConfigurationVersionFunc: func(ctx context.Context, params *appconfig.CreateHostedConfigurationVersionInput, optFns ...func(*appconfig.Options)) (*appconfig.CreateHostedConfigurationVersionOutput, error) {
			createVersionCalled = true
			return &appconfig.CreateHostedConfigurationVersionOutput{VersionNumber: 2}, nil
		},
		StartDeploymentFunc: func(ctx context.Context, params *appconfig.StartDeploymentInput, optFns ...func(*appconfig.Options)) (*appconfig.StartDeploymentOutput, error) {
			startDeploymentCalled = true
			return &appconfig.StartDeploymentOutput{DeploymentNumber: 2}, nil
		},
	}

	deployerFactory := func(ctx context.Context, cfg *config.Config) (*Deployer, error) {
		return NewWithClient(cfg, &awsInternal.Client{AppConfig: mockClient}), nil
	}

	reporter := &reportertest.MockReporter{}
	executor := NewExecutorWithFactory(reporter, deployerFactory)

	opts := &Options{
		ConfigFile: configPath,
		WaitDeploy: false,
		WaitBake:   false,
		Timeout:    300,
		Force:      false,
	}

	err = executor.Execute(context.Background(), opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify that deployment was NOT started
	if createVersionCalled {
		t.Error("CreateHostedConfigurationVersion should not have been called when there are no changes")
	}
	if startDeploymentCalled {
		t.Error("StartDeployment should not have been called when there are no changes")
	}

	// Verify success message about no changes
	found := false
	for _, msg := range reporter.Messages {
		if strings.Contains(msg, "No changes detected") || strings.Contains(msg, "no diff") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected message about no changes, got messages: %v", reporter.Messages)
	}
}

// TestExecutorForceDeploymentWithNoDiff tests that --force flag bypasses diff check
func TestExecutorForceDeploymentWithNoDiff(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "executor-force-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	configPath := filepath.Join(tempDir, "apcdeploy.yml")
	configContent := `application: test-app
configuration_profile: test-profile
environment: test-env
deployment_strategy: AppConfig.AllAtOnce
data_file: data.json
region: us-east-1
`
	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	// Create data file with same content as deployed version
	dataPath := filepath.Join(tempDir, "data.json")
	dataContent := []byte(`{"key": "value"}`)
	if err := os.WriteFile(dataPath, dataContent, 0o644); err != nil {
		t.Fatalf("Failed to write data: %v", err)
	}

	createVersionCalled := false
	startDeploymentCalled := false

	mockClient := &mock.MockAppConfigClient{
		ListApplicationsFunc: func(ctx context.Context, params *appconfig.ListApplicationsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListApplicationsOutput, error) {
			return &appconfig.ListApplicationsOutput{
				Items: []types.Application{{Id: aws.String("app-123"), Name: aws.String("test-app")}},
			}, nil
		},
		ListConfigurationProfilesFunc: func(ctx context.Context, params *appconfig.ListConfigurationProfilesInput, optFns ...func(*appconfig.Options)) (*appconfig.ListConfigurationProfilesOutput, error) {
			return &appconfig.ListConfigurationProfilesOutput{
				Items: []types.ConfigurationProfileSummary{{Id: aws.String("profile-123"), Name: aws.String("test-profile"), Type: aws.String("AWS.Freeform")}},
			}, nil
		},
		GetConfigurationProfileFunc: func(ctx context.Context, params *appconfig.GetConfigurationProfileInput, optFns ...func(*appconfig.Options)) (*appconfig.GetConfigurationProfileOutput, error) {
			return &appconfig.GetConfigurationProfileOutput{Id: aws.String("profile-123"), Type: aws.String("AWS.Freeform")}, nil
		},
		ListEnvironmentsFunc: func(ctx context.Context, params *appconfig.ListEnvironmentsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListEnvironmentsOutput, error) {
			return &appconfig.ListEnvironmentsOutput{
				Items: []types.Environment{{Id: aws.String("env-123"), Name: aws.String("test-env")}},
			}, nil
		},
		ListDeploymentStrategiesFunc: func(ctx context.Context, params *appconfig.ListDeploymentStrategiesInput, optFns ...func(*appconfig.Options)) (*appconfig.ListDeploymentStrategiesOutput, error) {
			return &appconfig.ListDeploymentStrategiesOutput{
				Items: []types.DeploymentStrategy{{Id: aws.String("strategy-123"), Name: aws.String("AppConfig.AllAtOnce")}},
			}, nil
		},
		ListDeploymentsFunc: func(ctx context.Context, params *appconfig.ListDeploymentsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListDeploymentsOutput, error) {
			// Return a completed deployment
			return &appconfig.ListDeploymentsOutput{
				Items: []types.DeploymentSummary{
					{
						DeploymentNumber:     1,
						State:                types.DeploymentStateComplete,
						ConfigurationVersion: aws.String("1"),
					},
				},
			}, nil
		},
		GetDeploymentFunc: func(ctx context.Context, params *appconfig.GetDeploymentInput, optFns ...func(*appconfig.Options)) (*appconfig.GetDeploymentOutput, error) {
			return &appconfig.GetDeploymentOutput{
				State:                  types.DeploymentStateComplete,
				ConfigurationProfileId: aws.String("profile-123"),
				ConfigurationVersion:   aws.String("1"),
			}, nil
		},
		GetHostedConfigurationVersionFunc: func(ctx context.Context, params *appconfig.GetHostedConfigurationVersionInput, optFns ...func(*appconfig.Options)) (*appconfig.GetHostedConfigurationVersionOutput, error) {
			// Return same content as local file (no diff)
			return &appconfig.GetHostedConfigurationVersionOutput{
				Content: dataContent,
			}, nil
		},
		CreateHostedConfigurationVersionFunc: func(ctx context.Context, params *appconfig.CreateHostedConfigurationVersionInput, optFns ...func(*appconfig.Options)) (*appconfig.CreateHostedConfigurationVersionOutput, error) {
			createVersionCalled = true
			return &appconfig.CreateHostedConfigurationVersionOutput{VersionNumber: 2}, nil
		},
		StartDeploymentFunc: func(ctx context.Context, params *appconfig.StartDeploymentInput, optFns ...func(*appconfig.Options)) (*appconfig.StartDeploymentOutput, error) {
			startDeploymentCalled = true
			return &appconfig.StartDeploymentOutput{DeploymentNumber: 2}, nil
		},
	}

	deployerFactory := func(ctx context.Context, cfg *config.Config) (*Deployer, error) {
		return NewWithClient(cfg, &awsInternal.Client{AppConfig: mockClient}), nil
	}

	reporter := &reportertest.MockReporter{}
	executor := NewExecutorWithFactory(reporter, deployerFactory)

	opts := &Options{
		ConfigFile: configPath,
		WaitDeploy: false,
		WaitBake:   false,
		Timeout:    300,
		Force:      true, // Force deployment even without changes
	}

	err = executor.Execute(context.Background(), opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify that deployment WAS started even without changes
	if !createVersionCalled {
		t.Error("CreateHostedConfigurationVersion should have been called with --force flag")
	}
	if !startDeploymentCalled {
		t.Error("StartDeployment should have been called with --force flag")
	}

	// Verify deployment started message
	found := false
	for _, msg := range reporter.Messages {
		if strings.Contains(msg, "Deployment #2 started") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected deployment started message, got messages: %v", reporter.Messages)
	}
}

// TestExecutorWithOngoingDeployment tests error when deployment is in progress
func TestExecutorWithOngoingDeployment(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "executor-ongoing-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	configPath := filepath.Join(tempDir, "apcdeploy.yml")
	if err := os.WriteFile(configPath, []byte(`application: test-app
configuration_profile: test-profile
environment: test-env
data_file: data.json
region: us-east-1
`), 0o644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	dataPath := filepath.Join(tempDir, "data.json")
	if err := os.WriteFile(dataPath, []byte(`{"key": "value"}`), 0o644); err != nil {
		t.Fatalf("Failed to write data: %v", err)
	}

	mockClient := &mock.MockAppConfigClient{
		ListApplicationsFunc: func(ctx context.Context, params *appconfig.ListApplicationsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListApplicationsOutput, error) {
			return &appconfig.ListApplicationsOutput{
				Items: []types.Application{{Id: aws.String("app-123"), Name: aws.String("test-app")}},
			}, nil
		},
		ListConfigurationProfilesFunc: func(ctx context.Context, params *appconfig.ListConfigurationProfilesInput, optFns ...func(*appconfig.Options)) (*appconfig.ListConfigurationProfilesOutput, error) {
			return &appconfig.ListConfigurationProfilesOutput{
				Items: []types.ConfigurationProfileSummary{{Id: aws.String("profile-123"), Name: aws.String("test-profile"), Type: aws.String("AWS.Freeform")}},
			}, nil
		},
		GetConfigurationProfileFunc: func(ctx context.Context, params *appconfig.GetConfigurationProfileInput, optFns ...func(*appconfig.Options)) (*appconfig.GetConfigurationProfileOutput, error) {
			return &appconfig.GetConfigurationProfileOutput{Id: aws.String("profile-123"), Type: aws.String("AWS.Freeform")}, nil
		},
		ListEnvironmentsFunc: func(ctx context.Context, params *appconfig.ListEnvironmentsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListEnvironmentsOutput, error) {
			return &appconfig.ListEnvironmentsOutput{
				Items: []types.Environment{{Id: aws.String("env-123"), Name: aws.String("test-env")}},
			}, nil
		},
		ListDeploymentStrategiesFunc: func(ctx context.Context, params *appconfig.ListDeploymentStrategiesInput, optFns ...func(*appconfig.Options)) (*appconfig.ListDeploymentStrategiesOutput, error) {
			return &appconfig.ListDeploymentStrategiesOutput{
				Items: []types.DeploymentStrategy{{Id: aws.String("strategy-123"), Name: aws.String("AppConfig.AllAtOnce")}},
			}, nil
		},
		ListDeploymentsFunc: func(ctx context.Context, params *appconfig.ListDeploymentsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListDeploymentsOutput, error) {
			// Return an ongoing deployment
			return &appconfig.ListDeploymentsOutput{
				Items: []types.DeploymentSummary{
					{
						DeploymentNumber: 1,
						State:            types.DeploymentStateDeploying,
					},
				},
			}, nil
		},
	}

	deployerFactory := func(ctx context.Context, cfg *config.Config) (*Deployer, error) {
		return NewWithClient(cfg, &awsInternal.Client{
			AppConfig:       mockClient,
			PollingInterval: 100 * time.Millisecond, // Fast polling for tests
		}), nil
	}

	reporter := &reportertest.MockReporter{}
	executor := NewExecutorWithFactory(reporter, deployerFactory)

	opts := &Options{
		ConfigFile: configPath,
		WaitDeploy: false,
		WaitBake:   false,
		Timeout:    300,
	}

	err = executor.Execute(context.Background(), opts)

	if err == nil {
		t.Fatal("expected error for ongoing deployment")
	}

	if !strings.Contains(err.Error(), "deployment already in progress") {
		t.Errorf("expected 'deployment already in progress' error, got: %v", err)
	}
}
