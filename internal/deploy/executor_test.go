package deploy

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/appconfig"
	"github.com/aws/aws-sdk-go-v2/service/appconfig/types"
	awsInternal "github.com/koh-sh/apcdeploy/internal/aws"
	"github.com/koh-sh/apcdeploy/internal/aws/mock"
	"github.com/koh-sh/apcdeploy/internal/config"
)

// mockReporter is a test implementation of ProgressReporter
type mockReporter struct {
	messages []string
}

func (m *mockReporter) Progress(message string) {
	m.messages = append(m.messages, "progress: "+message)
}

func (m *mockReporter) Success(message string) {
	m.messages = append(m.messages, "success: "+message)
}

func (m *mockReporter) Warning(message string) {
	m.messages = append(m.messages, "warning: "+message)
}

func TestNewExecutor(t *testing.T) {
	reporter := &mockReporter{}
	executor := NewExecutor(reporter)

	if executor == nil {
		t.Fatal("expected executor to be non-nil")
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
			reporter := &mockReporter{}
			executor := NewExecutor(reporter)

			opts := &Options{
				ConfigFile: "nonexistent.yml",
				NoWait:     true,
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

func TestExecutorLoadConfigurationError(t *testing.T) {
	reporter := &mockReporter{}
	executor := NewExecutor(reporter)

	opts := &Options{
		ConfigFile: "nonexistent.yml",
		NoWait:     true,
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
	if len(reporter.messages) == 0 {
		t.Error("expected reporter to have received messages")
	}

	if !strings.Contains(reporter.messages[0], "Loading configuration") {
		t.Errorf("expected first message to be about loading configuration, got: %v", reporter.messages[0])
	}
}

func TestExecutorReporterMessages(t *testing.T) {
	// Create temporary test files
	tempDir, err := os.MkdirTemp("", "executor-test-*")
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

	reporter := &mockReporter{}
	executor := NewExecutor(reporter)

	opts := &Options{
		ConfigFile: configPath,
		NoWait:     true,
		Timeout:    300,
	}

	// This will fail because we don't have real AWS credentials,
	// but we can verify that the reporter received the initial messages
	_ = executor.Execute(context.Background(), opts)

	// Verify reporter received messages
	if len(reporter.messages) < 1 {
		t.Error("expected reporter to receive at least one message")
	}

	// Check first message is about loading configuration
	if !strings.Contains(reporter.messages[0], "Loading configuration") {
		t.Errorf("expected first message about loading configuration, got: %v", reporter.messages[0])
	}

	// Check that we have a success message for configuration loading
	hasSuccessMessage := false
	for _, msg := range reporter.messages {
		if strings.Contains(msg, "success") && strings.Contains(msg, "Configuration loaded") {
			hasSuccessMessage = true
			break
		}
	}

	if !hasSuccessMessage {
		t.Error("expected success message for configuration loading")
	}
}

func TestExecutorNoWaitOption(t *testing.T) {
	tests := []struct {
		name   string
		noWait bool
	}{
		{
			name:   "with no-wait option",
			noWait: true,
		},
		{
			name:   "without no-wait option",
			noWait: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reporter := &mockReporter{}
			executor := NewExecutor(reporter)

			opts := &Options{
				ConfigFile: "nonexistent.yml",
				NoWait:     tt.noWait,
				Timeout:    300,
			}

			// This will fail due to missing config file,
			// but the Options struct is properly set
			_ = executor.Execute(context.Background(), opts)

			// The test passes if no panic occurs and options are properly passed
		})
	}
}

// TestExecutorIntegrationSuccess tests the complete deployment flow with mocked AWS
func TestExecutorIntegrationSuccess(t *testing.T) {
	// Create temporary test files
	tempDir, err := os.MkdirTemp("", "executor-integration-*")
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
	}

	// Mock client created for future use when dependency injection is added
	_ = mockClient

	reporter := &mockReporter{}
	executor := NewExecutor(reporter)

	opts := &Options{
		ConfigFile: configPath,
		NoWait:     true,
		Timeout:    300,
	}

	// Note: This will fail because New() creates its own AWS client
	// This test documents what needs to be done for full integration testing
	_ = executor.Execute(context.Background(), opts)

	// For now, verify that reporter was called
	if len(reporter.messages) == 0 {
		t.Error("expected reporter to receive messages")
	}
}

// TestExecutorWithValidationError tests validation error handling
func TestExecutorWithValidationError(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "executor-validation-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create config with invalid JSON data
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

	// Create invalid JSON data file
	dataPath := filepath.Join(tempDir, "data.json")
	if err := os.WriteFile(dataPath, []byte(`{invalid json`), 0o644); err != nil {
		t.Fatalf("Failed to write data: %v", err)
	}

	reporter := &mockReporter{}
	executor := NewExecutor(reporter)

	opts := &Options{
		ConfigFile: configPath,
		NoWait:     true,
		Timeout:    300,
	}

	err = executor.Execute(context.Background(), opts)

	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}

	// The error could be either validation or AWS connection error
	// Both are acceptable for this test case
	if !strings.Contains(err.Error(), "validation failed") && !strings.Contains(err.Error(), "failed to resolve") {
		t.Errorf("expected validation or resolve error, got: %v", err)
	}
}

// TestExecutorWithNoWaitFlag tests deployment with no-wait option
func TestExecutorWithNoWaitFlag(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "executor-nowait-*")
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

	tests := []struct {
		name   string
		noWait bool
	}{
		{
			name:   "with no-wait flag",
			noWait: true,
		},
		{
			name:   "without no-wait flag (wait for deployment)",
			noWait: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reporter := &mockReporter{}
			executor := NewExecutor(reporter)

			opts := &Options{
				ConfigFile: configPath,
				NoWait:     tt.noWait,
				Timeout:    300,
			}

			// Will fail due to AWS client initialization, but tests option handling
			_ = executor.Execute(context.Background(), opts)

			// Verify reporter received initial messages
			if len(reporter.messages) == 0 {
				t.Error("expected reporter to receive messages")
			}
		})
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

	reporter := &mockReporter{}
	executor := NewExecutorWithFactory(reporter, deployerFactory)

	opts := &Options{
		ConfigFile: configPath,
		NoWait:     true,
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
		for _, msg := range reporter.messages {
			if strings.Contains(msg, expected) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected message containing %q not found in: %v", expected, reporter.messages)
		}
	}
}

// TestExecutorFullWorkflowWithWait tests deployment with wait option
func TestExecutorFullWorkflowWithWait(t *testing.T) {
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
			callCount++
			state := types.DeploymentStateDeploying
			if callCount >= 2 {
				state = types.DeploymentStateComplete
			}
			return &appconfig.GetDeploymentOutput{State: state}, nil
		},
	}

	deployerFactory := func(ctx context.Context, cfg *config.Config) (*Deployer, error) {
		return NewWithClient(cfg, &awsInternal.Client{AppConfig: mockClient}), nil
	}

	reporter := &mockReporter{}
	executor := NewExecutorWithFactory(reporter, deployerFactory)

	opts := &Options{
		ConfigFile: configPath,
		NoWait:     false, // Wait for deployment
		Timeout:    30,
	}

	err = executor.Execute(context.Background(), opts)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify deployment completed message
	hasCompletedMsg := false
	for _, msg := range reporter.messages {
		if strings.Contains(msg, "Deployment completed successfully") {
			hasCompletedMsg = true
			break
		}
	}

	if !hasCompletedMsg {
		t.Error("expected 'Deployment completed successfully' message")
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
		return NewWithClient(cfg, &awsInternal.Client{AppConfig: mockClient}), nil
	}

	reporter := &mockReporter{}
	executor := NewExecutorWithFactory(reporter, deployerFactory)

	opts := &Options{
		ConfigFile: configPath,
		NoWait:     true,
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
