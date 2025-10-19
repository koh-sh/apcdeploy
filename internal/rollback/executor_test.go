package rollback

import (
	"context"
	"errors"
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
	prompttest "github.com/koh-sh/apcdeploy/internal/prompt/testing"
	reportertest "github.com/koh-sh/apcdeploy/internal/reporter/testing"
)

// Helper function to create standard mock client with common AWS API responses
func createStandardMockClient(
	listDeploymentsFunc func(ctx context.Context, params *appconfig.ListDeploymentsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListDeploymentsOutput, error),
	getDeploymentFunc func(ctx context.Context, params *appconfig.GetDeploymentInput, optFns ...func(*appconfig.Options)) (*appconfig.GetDeploymentOutput, error),
	stopDeploymentFunc func(ctx context.Context, params *appconfig.StopDeploymentInput, optFns ...func(*appconfig.Options)) (*appconfig.StopDeploymentOutput, error),
) *mock.MockAppConfigClient {
	return &mock.MockAppConfigClient{
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
		ListDeploymentsFunc: listDeploymentsFunc,
		GetDeploymentFunc:   getDeploymentFunc,
		StopDeploymentFunc:  stopDeploymentFunc,
	}
}

// Helper function to create test configuration files
func createTestConfig(t *testing.T) (configPath string, cleanup func()) {
	t.Helper()

	tempDir, err := os.MkdirTemp("", "rollback-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	configPath = filepath.Join(tempDir, "apcdeploy.yml")
	configContent := `application: test-app
configuration_profile: test-profile
environment: test-env
deployment_strategy: AppConfig.AllAtOnce
data_file: data.json
region: us-east-1
`
	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to write config: %v", err)
	}

	dataPath := filepath.Join(tempDir, "data.json")
	if err := os.WriteFile(dataPath, []byte(`{"key": "value"}`), 0o644); err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("Failed to write data: %v", err)
	}

	return configPath, func() { os.RemoveAll(tempDir) }
}

func TestNewExecutor(t *testing.T) {
	t.Parallel()

	reporter := &reportertest.MockReporter{}
	prompter := &prompttest.MockPrompter{}
	executor := NewExecutor(reporter, prompter)

	if executor == nil {
		t.Fatal("expected executor to be non-nil")
		return
	}

	if executor.reporter != reporter {
		t.Error("expected executor to have the provided reporter")
	}

	if executor.prompter != prompter {
		t.Error("expected executor to have the provided prompter")
	}
}

func TestExecutorLoadConfigurationError(t *testing.T) {
	t.Parallel()

	reporter := &reportertest.MockReporter{}
	prompter := &prompttest.MockPrompter{}
	executor := NewExecutor(reporter, prompter)

	opts := &Options{
		ConfigFile: "nonexistent.yml",
	}

	err := executor.Execute(context.Background(), opts)

	if err == nil {
		t.Error("expected error when loading non-existent config file")
	}

	if !strings.Contains(err.Error(), "failed to load configuration") {
		t.Errorf("expected 'failed to load configuration' error, got: %v", err)
	}
}

func TestExecutorNoOngoingDeployment(t *testing.T) {
	t.Parallel()

	configPath, cleanup := createTestConfig(t)
	defer cleanup()

	// Create mock AWS client that returns no ongoing deployments
	mockClient := createStandardMockClient(
		func(ctx context.Context, params *appconfig.ListDeploymentsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListDeploymentsOutput, error) {
			// No ongoing deployments
			return &appconfig.ListDeploymentsOutput{
				Items: []types.DeploymentSummary{
					{
						DeploymentNumber: 1,
						State:            types.DeploymentStateComplete,
					},
				},
			}, nil
		},
		nil, // GetDeployment not called in this test
		nil, // StopDeployment not called in this test
	)

	reporter := &reportertest.MockReporter{}
	prompter := &prompttest.MockPrompter{}
	executor := NewExecutorWithFactory(reporter, prompter, func(ctx context.Context, region string) (*awsInternal.Client, error) {
		return awsInternal.NewTestClient(mockClient), nil
	})

	opts := &Options{
		ConfigFile: configPath,
	}

	err := executor.Execute(context.Background(), opts)
	if err == nil {
		t.Error("expected error for no ongoing deployment case")
	}

	if !errors.Is(err, ErrNoOngoingDeployment) {
		t.Errorf("expected ErrNoOngoingDeployment, got: %v", err)
	}
}

func TestExecutorSuccessWithOngoingDeployment(t *testing.T) {
	t.Parallel()

	configPath, cleanup := createTestConfig(t)
	defer cleanup()

	// Track if StopDeployment was called
	stopDeploymentCalled := false

	// Create mock AWS client with ongoing deployment
	mockClient := createStandardMockClient(
		func(ctx context.Context, params *appconfig.ListDeploymentsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListDeploymentsOutput, error) {
			return &appconfig.ListDeploymentsOutput{
				Items: []types.DeploymentSummary{
					{
						DeploymentNumber: 2,
						State:            types.DeploymentStateDeploying,
					},
					{
						DeploymentNumber: 1,
						State:            types.DeploymentStateComplete,
					},
				},
			}, nil
		},
		func(ctx context.Context, params *appconfig.GetDeploymentInput, optFns ...func(*appconfig.Options)) (*appconfig.GetDeploymentOutput, error) {
			deploymentNum := int32(2)
			return &appconfig.GetDeploymentOutput{
				DeploymentNumber:       deploymentNum,
				ConfigurationProfileId: aws.String("profile-123"),
				ConfigurationVersion:   aws.String("1"),
				DeploymentStrategyId:   aws.String("strategy-123"),
				State:                  types.DeploymentStateDeploying,
				EventLog:               []types.DeploymentEvent{},
				StartedAt:              aws.Time(time.Now()),
				PercentageComplete:     aws.Float32(50.0),
				GrowthFactor:           aws.Float32(100.0),
			}, nil
		},
		func(ctx context.Context, params *appconfig.StopDeploymentInput, optFns ...func(*appconfig.Options)) (*appconfig.StopDeploymentOutput, error) {
			stopDeploymentCalled = true
			return &appconfig.StopDeploymentOutput{}, nil
		},
	)

	reporter := &reportertest.MockReporter{}
	prompter := &prompttest.MockPrompter{
		InputFunc: func(message string, placeholder string) (string, error) {
			return "yes", nil
		},
	}
	executor := NewExecutorWithFactory(reporter, prompter, func(ctx context.Context, region string) (*awsInternal.Client, error) {
		return awsInternal.NewTestClient(mockClient), nil
	})

	opts := &Options{
		ConfigFile: configPath,
	}

	err := executor.Execute(context.Background(), opts)
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}

	if !stopDeploymentCalled {
		t.Error("expected StopDeployment to be called")
	}
}

func TestExecutorUserDeclined(t *testing.T) {
	t.Parallel()

	configPath, cleanup := createTestConfig(t)
	defer cleanup()

	// Create mock AWS client with ongoing deployment
	mockClient := createStandardMockClient(
		func(ctx context.Context, params *appconfig.ListDeploymentsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListDeploymentsOutput, error) {
			return &appconfig.ListDeploymentsOutput{
				Items: []types.DeploymentSummary{
					{
						DeploymentNumber: 2,
						State:            types.DeploymentStateDeploying,
					},
				},
			}, nil
		},
		func(ctx context.Context, params *appconfig.GetDeploymentInput, optFns ...func(*appconfig.Options)) (*appconfig.GetDeploymentOutput, error) {
			deploymentNum := int32(2)
			return &appconfig.GetDeploymentOutput{
				DeploymentNumber:       deploymentNum,
				ConfigurationProfileId: aws.String("profile-123"),
				ConfigurationVersion:   aws.String("1"),
				DeploymentStrategyId:   aws.String("strategy-123"),
				State:                  types.DeploymentStateDeploying,
				EventLog:               []types.DeploymentEvent{},
				StartedAt:              aws.Time(time.Now()),
			}, nil
		},
		nil, // StopDeployment not called in this test
	)

	reporter := &reportertest.MockReporter{}
	prompter := &prompttest.MockPrompter{
		InputFunc: func(message string, placeholder string) (string, error) {
			return "no", nil // User declines
		},
	}
	executor := NewExecutorWithFactory(reporter, prompter, func(ctx context.Context, region string) (*awsInternal.Client, error) {
		return awsInternal.NewTestClient(mockClient), nil
	})

	opts := &Options{
		ConfigFile: configPath,
	}

	err := executor.Execute(context.Background(), opts)
	if err == nil {
		t.Error("expected error when user declines")
	}

	if !errors.Is(err, ErrUserDeclined) {
		t.Errorf("expected ErrUserDeclined, got: %v", err)
	}
}

func TestExecutorSkipConfirmation(t *testing.T) {
	t.Parallel()

	configPath, cleanup := createTestConfig(t)
	defer cleanup()

	stopDeploymentCalled := false

	// Create mock AWS client with ongoing deployment
	mockClient := createStandardMockClient(
		func(ctx context.Context, params *appconfig.ListDeploymentsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListDeploymentsOutput, error) {
			return &appconfig.ListDeploymentsOutput{
				Items: []types.DeploymentSummary{
					{
						DeploymentNumber: 2,
						State:            types.DeploymentStateDeploying,
					},
				},
			}, nil
		},
		func(ctx context.Context, params *appconfig.GetDeploymentInput, optFns ...func(*appconfig.Options)) (*appconfig.GetDeploymentOutput, error) {
			deploymentNum := int32(2)
			return &appconfig.GetDeploymentOutput{
				DeploymentNumber:       deploymentNum,
				ConfigurationProfileId: aws.String("profile-123"),
				ConfigurationVersion:   aws.String("1"),
				DeploymentStrategyId:   aws.String("strategy-123"),
				State:                  types.DeploymentStateDeploying,
				EventLog:               []types.DeploymentEvent{},
				StartedAt:              aws.Time(time.Now()),
			}, nil
		},
		func(ctx context.Context, params *appconfig.StopDeploymentInput, optFns ...func(*appconfig.Options)) (*appconfig.StopDeploymentOutput, error) {
			stopDeploymentCalled = true
			return &appconfig.StopDeploymentOutput{}, nil
		},
	)

	reporter := &reportertest.MockReporter{}
	prompter := &prompttest.MockPrompter{}
	executor := NewExecutorWithFactory(reporter, prompter, func(ctx context.Context, region string) (*awsInternal.Client, error) {
		return awsInternal.NewTestClient(mockClient), nil
	})

	opts := &Options{
		ConfigFile:       configPath,
		SkipConfirmation: true, // Skip confirmation prompt
	}

	err := executor.Execute(context.Background(), opts)
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}

	if !stopDeploymentCalled {
		t.Error("expected StopDeployment to be called")
	}
}

func TestExecutorTTYCheckError(t *testing.T) {
	t.Parallel()

	configPath, cleanup := createTestConfig(t)
	defer cleanup()

	// Create mock AWS client with ongoing deployment
	mockClient := createStandardMockClient(
		func(ctx context.Context, params *appconfig.ListDeploymentsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListDeploymentsOutput, error) {
			return &appconfig.ListDeploymentsOutput{
				Items: []types.DeploymentSummary{
					{
						DeploymentNumber: 2,
						State:            types.DeploymentStateDeploying,
					},
				},
			}, nil
		},
		func(ctx context.Context, params *appconfig.GetDeploymentInput, optFns ...func(*appconfig.Options)) (*appconfig.GetDeploymentOutput, error) {
			deploymentNum := int32(2)
			return &appconfig.GetDeploymentOutput{
				DeploymentNumber:       deploymentNum,
				ConfigurationProfileId: aws.String("profile-123"),
				ConfigurationVersion:   aws.String("1"),
				DeploymentStrategyId:   aws.String("strategy-123"),
				State:                  types.DeploymentStateDeploying,
				EventLog:               []types.DeploymentEvent{},
				StartedAt:              aws.Time(time.Now()),
			}, nil
		},
		nil, // StopDeployment not called in this test
	)

	reporter := &reportertest.MockReporter{}
	prompter := &prompttest.MockPrompter{
		CheckTTYFunc: func() error {
			return errors.New("not a tty")
		},
	}
	executor := NewExecutorWithFactory(reporter, prompter, func(ctx context.Context, region string) (*awsInternal.Client, error) {
		return awsInternal.NewTestClient(mockClient), nil
	})

	opts := &Options{
		ConfigFile:       configPath,
		SkipConfirmation: false, // Require confirmation (TTY check will fail)
	}

	err := executor.Execute(context.Background(), opts)
	if err == nil {
		t.Error("expected error when TTY check fails")
	}

	if !strings.Contains(err.Error(), "use --yes to skip confirmation") {
		t.Errorf("expected error message to suggest --yes flag, got: %v", err)
	}
}

func TestExecutorStopDeploymentError(t *testing.T) {
	t.Parallel()

	configPath, cleanup := createTestConfig(t)
	defer cleanup()

	// Create mock AWS client with ongoing deployment and StopDeployment error
	mockClient := createStandardMockClient(
		func(ctx context.Context, params *appconfig.ListDeploymentsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListDeploymentsOutput, error) {
			return &appconfig.ListDeploymentsOutput{
				Items: []types.DeploymentSummary{
					{
						DeploymentNumber: 2,
						State:            types.DeploymentStateDeploying,
					},
				},
			}, nil
		},
		func(ctx context.Context, params *appconfig.GetDeploymentInput, optFns ...func(*appconfig.Options)) (*appconfig.GetDeploymentOutput, error) {
			deploymentNum := int32(2)
			return &appconfig.GetDeploymentOutput{
				DeploymentNumber:       deploymentNum,
				ConfigurationProfileId: aws.String("profile-123"),
				ConfigurationVersion:   aws.String("1"),
				DeploymentStrategyId:   aws.String("strategy-123"),
				State:                  types.DeploymentStateDeploying,
				EventLog:               []types.DeploymentEvent{},
				StartedAt:              aws.Time(time.Now()),
			}, nil
		},
		func(ctx context.Context, params *appconfig.StopDeploymentInput, optFns ...func(*appconfig.Options)) (*appconfig.StopDeploymentOutput, error) {
			return nil, errors.New("API error")
		},
	)

	reporter := &reportertest.MockReporter{}
	prompter := &prompttest.MockPrompter{}
	executor := NewExecutorWithFactory(reporter, prompter, func(ctx context.Context, region string) (*awsInternal.Client, error) {
		return awsInternal.NewTestClient(mockClient), nil
	})

	opts := &Options{
		ConfigFile:       configPath,
		SkipConfirmation: true, // Skip confirmation to reach StopDeployment call
	}

	err := executor.Execute(context.Background(), opts)
	if err == nil {
		t.Error("expected error when StopDeployment fails")
	}

	if !strings.Contains(err.Error(), "failed to stop deployment") {
		t.Errorf("expected 'failed to stop deployment' error, got: %v", err)
	}
}

func TestExecutorGetDeploymentDetailsError(t *testing.T) {
	t.Parallel()

	configPath, cleanup := createTestConfig(t)
	defer cleanup()

	// Create mock AWS client with ongoing deployment but GetDeployment error
	mockClient := createStandardMockClient(
		func(ctx context.Context, params *appconfig.ListDeploymentsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListDeploymentsOutput, error) {
			return &appconfig.ListDeploymentsOutput{
				Items: []types.DeploymentSummary{
					{
						DeploymentNumber: 2,
						State:            types.DeploymentStateDeploying,
					},
				},
			}, nil
		},
		func(ctx context.Context, params *appconfig.GetDeploymentInput, optFns ...func(*appconfig.Options)) (*appconfig.GetDeploymentOutput, error) {
			return nil, errors.New("API error getting deployment details")
		},
		nil, // StopDeployment not called in this test
	)

	reporter := &reportertest.MockReporter{}
	prompter := &prompttest.MockPrompter{}
	executor := NewExecutorWithFactory(reporter, prompter, func(ctx context.Context, region string) (*awsInternal.Client, error) {
		return awsInternal.NewTestClient(mockClient), nil
	})

	opts := &Options{
		ConfigFile:       configPath,
		SkipConfirmation: true,
	}

	err := executor.Execute(context.Background(), opts)
	if err == nil {
		t.Error("expected error when GetDeployment fails")
	}

	if !strings.Contains(err.Error(), "failed to get deployment details") {
		t.Errorf("expected 'failed to get deployment details' error, got: %v", err)
	}
}

func TestExecutorCheckOngoingDeploymentError(t *testing.T) {
	t.Parallel()

	configPath, cleanup := createTestConfig(t)
	defer cleanup()

	// Create mock AWS client that returns error when checking deployments
	mockClient := createStandardMockClient(
		func(ctx context.Context, params *appconfig.ListDeploymentsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListDeploymentsOutput, error) {
			return nil, errors.New("API error listing deployments")
		},
		nil, // GetDeployment not called in this test
		nil, // StopDeployment not called in this test
	)

	reporter := &reportertest.MockReporter{}
	prompter := &prompttest.MockPrompter{}
	executor := NewExecutorWithFactory(reporter, prompter, func(ctx context.Context, region string) (*awsInternal.Client, error) {
		return awsInternal.NewTestClient(mockClient), nil
	})

	opts := &Options{
		ConfigFile: configPath,
	}

	err := executor.Execute(context.Background(), opts)
	if err == nil {
		t.Error("expected error when checking ongoing deployment fails")
	}

	if !strings.Contains(err.Error(), "failed to check ongoing deployment") {
		t.Errorf("expected 'failed to check ongoing deployment' error, got: %v", err)
	}
}
