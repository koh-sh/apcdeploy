package init

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/appconfig"
	appconfigTypes "github.com/aws/aws-sdk-go-v2/service/appconfig/types"
	awsInternal "github.com/koh-sh/apcdeploy/internal/aws"
	awsMock "github.com/koh-sh/apcdeploy/internal/aws/mock"
	"github.com/koh-sh/apcdeploy/internal/prompt"
	promptTesting "github.com/koh-sh/apcdeploy/internal/prompt/testing"
	"github.com/koh-sh/apcdeploy/internal/reporter"
	reporterTesting "github.com/koh-sh/apcdeploy/internal/reporter/testing"
)

func TestNewExecutor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		validateFunc func(*testing.T, *Executor)
	}{
		{
			name: "creates executor with reporter and prompter",
			validateFunc: func(t *testing.T, executor *Executor) {
				if executor == nil {
					t.Fatal("expected non-nil Executor")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mockReporter := &reporterTesting.MockReporter{}
			mockPrompter := &promptTesting.MockPrompter{}

			executor := NewExecutor(mockReporter, mockPrompter)

			if executor.reporter != mockReporter {
				t.Error("expected reporter to be set")
			}
			if executor.prompter != mockPrompter {
				t.Error("expected prompter to be set")
			}

			if tt.validateFunc != nil {
				tt.validateFunc(t, executor)
			}
		})
	}
}

// TestExecutorFullWorkflowWithMock tests the complete init workflow with mocked AWS
func TestExecutorFullWorkflowWithMock(t *testing.T) {
	t.Parallel()

	// Create temporary test files
	tempDir, err := os.MkdirTemp("", "executor-full-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	configPath := filepath.Join(tempDir, "apcdeploy.yml")
	dataFilename := "data.json"

	// Create mock AWS client
	mockClient := &awsMock.MockAppConfigClient{
		ListHostedConfigurationVersionsFunc: func(ctx context.Context, params *appconfig.ListHostedConfigurationVersionsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListHostedConfigurationVersionsOutput, error) {
			return &appconfig.ListHostedConfigurationVersionsOutput{
				Items: []appconfigTypes.HostedConfigurationVersionSummary{
					{VersionNumber: 1},
				},
			}, nil
		},
		ListApplicationsFunc: func(ctx context.Context, params *appconfig.ListApplicationsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListApplicationsOutput, error) {
			return &appconfig.ListApplicationsOutput{
				Items: []appconfigTypes.Application{
					{
						Id:   aws.String("app-123"),
						Name: aws.String("test-app"),
					},
				},
			}, nil
		},
		ListConfigurationProfilesFunc: func(ctx context.Context, params *appconfig.ListConfigurationProfilesInput, optFns ...func(*appconfig.Options)) (*appconfig.ListConfigurationProfilesOutput, error) {
			return &appconfig.ListConfigurationProfilesOutput{
				Items: []appconfigTypes.ConfigurationProfileSummary{
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
				Items: []appconfigTypes.Environment{
					{
						Id:   aws.String("env-123"),
						Name: aws.String("test-env"),
					},
				},
			}, nil
		},
		ListDeploymentStrategiesFunc: func(ctx context.Context, params *appconfig.ListDeploymentStrategiesInput, optFns ...func(*appconfig.Options)) (*appconfig.ListDeploymentStrategiesOutput, error) {
			return &appconfig.ListDeploymentStrategiesOutput{
				Items: []appconfigTypes.DeploymentStrategy{
					{Id: aws.String("strategy-123"), Name: aws.String("AppConfig.AllAtOnce")},
				},
			}, nil
		},
		ListDeploymentsFunc: func(ctx context.Context, params *appconfig.ListDeploymentsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListDeploymentsOutput, error) {
			return &appconfig.ListDeploymentsOutput{
				Items: []appconfigTypes.DeploymentSummary{
					{
						DeploymentNumber:     1,
						State:                appconfigTypes.DeploymentStateComplete,
						ConfigurationVersion: aws.String("1"),
					},
				},
			}, nil
		},
		GetDeploymentFunc: func(ctx context.Context, params *appconfig.GetDeploymentInput, optFns ...func(*appconfig.Options)) (*appconfig.GetDeploymentOutput, error) {
			return &appconfig.GetDeploymentOutput{
				State:                  appconfigTypes.DeploymentStateComplete,
				ConfigurationProfileId: aws.String("profile-123"),
				ConfigurationVersion:   aws.String("1"),
			}, nil
		},
		GetHostedConfigurationVersionFunc: func(ctx context.Context, params *appconfig.GetHostedConfigurationVersionInput, optFns ...func(*appconfig.Options)) (*appconfig.GetHostedConfigurationVersionOutput, error) {
			return &appconfig.GetHostedConfigurationVersionOutput{
				ApplicationId:          aws.String("app-123"),
				ConfigurationProfileId: aws.String("profile-123"),
				VersionNumber:          1,
				Content:                []byte(`{"key": "value"}`),
				ContentType:            aws.String("application/json"),
			}, nil
		},
	}

	// Create workflow factory that uses the mock client
	workflowFactory := func(ctx context.Context, opts *Options, prompter prompt.Prompter, reporter reporter.ProgressReporter) (*InitWorkflow, error) {
		awsClient := awsInternal.NewTestClient(mockClient)
		return NewInitWorkflowWithClient(awsClient, prompter, reporter), nil
	}

	mockReporter := &reporterTesting.MockReporter{}
	mockPrompter := &promptTesting.MockPrompter{}

	executor := NewExecutorWithFactory(mockReporter, mockPrompter, workflowFactory)

	opts := &Options{
		Application: "test-app",
		Profile:     "test-profile",
		Environment: "test-env",
		Region:      "us-east-1",
		ConfigFile:  configPath,
		OutputData:  dataFilename,
		Force:       false,
	}

	err = executor.Execute(context.Background(), opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify files were created
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Errorf("config file was not created: %s", configPath)
	}
	dataPath := filepath.Join(tempDir, dataFilename)
	if _, err := os.Stat(dataPath); os.IsNotExist(err) {
		t.Errorf("data file was not created: %s", dataPath)
	}

	// Verify all expected messages were reported
	expectedMessages := []string{
		"Initializing apcdeploy configuration",
		"Resolving AWS resources",
		"Fetching",
		"Generating configuration file",
		"Initialization complete",
	}

	for _, expected := range expectedMessages {
		found := false
		for _, msg := range mockReporter.Messages {
			if strings.Contains(msg, expected) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected message containing %q not found in: %v", expected, mockReporter.Messages)
		}
	}
}

// TestExecutorWithInteractiveSelection tests interactive resource selection
func TestExecutorWithInteractiveSelection(t *testing.T) {
	t.Parallel()

	// Create temporary test files
	tempDir, err := os.MkdirTemp("", "executor-interactive-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	configPath := filepath.Join(tempDir, "apcdeploy.yml")
	dataFilename := "data.json"

	// Create mock AWS client
	mockClient := &awsMock.MockAppConfigClient{
		ListHostedConfigurationVersionsFunc: func(ctx context.Context, params *appconfig.ListHostedConfigurationVersionsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListHostedConfigurationVersionsOutput, error) {
			return &appconfig.ListHostedConfigurationVersionsOutput{
				Items: []appconfigTypes.HostedConfigurationVersionSummary{
					{VersionNumber: 1},
				},
			}, nil
		},
		ListApplicationsFunc: func(ctx context.Context, params *appconfig.ListApplicationsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListApplicationsOutput, error) {
			return &appconfig.ListApplicationsOutput{
				Items: []appconfigTypes.Application{
					{Id: aws.String("app-123"), Name: aws.String("test-app")},
					{Id: aws.String("app-456"), Name: aws.String("other-app")},
				},
			}, nil
		},
		ListConfigurationProfilesFunc: func(ctx context.Context, params *appconfig.ListConfigurationProfilesInput, optFns ...func(*appconfig.Options)) (*appconfig.ListConfigurationProfilesOutput, error) {
			return &appconfig.ListConfigurationProfilesOutput{
				Items: []appconfigTypes.ConfigurationProfileSummary{
					{Id: aws.String("profile-123"), Name: aws.String("test-profile"), Type: aws.String("AWS.Freeform")},
					{Id: aws.String("profile-456"), Name: aws.String("other-profile"), Type: aws.String("AWS.Freeform")},
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
				Items: []appconfigTypes.Environment{
					{Id: aws.String("env-123"), Name: aws.String("test-env")},
					{Id: aws.String("env-456"), Name: aws.String("other-env")},
				},
			}, nil
		},
		ListDeploymentStrategiesFunc: func(ctx context.Context, params *appconfig.ListDeploymentStrategiesInput, optFns ...func(*appconfig.Options)) (*appconfig.ListDeploymentStrategiesOutput, error) {
			return &appconfig.ListDeploymentStrategiesOutput{
				Items: []appconfigTypes.DeploymentStrategy{
					{Id: aws.String("strategy-123"), Name: aws.String("AppConfig.AllAtOnce")},
				},
			}, nil
		},
		ListDeploymentsFunc: func(ctx context.Context, params *appconfig.ListDeploymentsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListDeploymentsOutput, error) {
			return &appconfig.ListDeploymentsOutput{
				Items: []appconfigTypes.DeploymentSummary{
					{
						DeploymentNumber:     1,
						State:                appconfigTypes.DeploymentStateComplete,
						ConfigurationVersion: aws.String("1"),
					},
				},
			}, nil
		},
		GetDeploymentFunc: func(ctx context.Context, params *appconfig.GetDeploymentInput, optFns ...func(*appconfig.Options)) (*appconfig.GetDeploymentOutput, error) {
			return &appconfig.GetDeploymentOutput{
				State:                  appconfigTypes.DeploymentStateComplete,
				ConfigurationProfileId: aws.String("profile-123"),
				ConfigurationVersion:   aws.String("1"),
			}, nil
		},
		GetHostedConfigurationVersionFunc: func(ctx context.Context, params *appconfig.GetHostedConfigurationVersionInput, optFns ...func(*appconfig.Options)) (*appconfig.GetHostedConfigurationVersionOutput, error) {
			return &appconfig.GetHostedConfigurationVersionOutput{
				ApplicationId:          aws.String("app-123"),
				ConfigurationProfileId: aws.String("profile-123"),
				VersionNumber:          1,
				Content:                []byte(`{"key": "value"}`),
				ContentType:            aws.String("application/json"),
			}, nil
		},
	}

	// Configure mock prompter to return selections
	selectCallCount := 0
	mockPrompter := &promptTesting.MockPrompter{
		SelectFunc: func(message string, options []string) (string, error) {
			selectCallCount++
			// Return the first option for each selection
			if len(options) > 0 {
				return options[0], nil
			}
			return "", nil
		},
	}

	// Create workflow factory that uses the mock client
	workflowFactory := func(ctx context.Context, opts *Options, prompter prompt.Prompter, reporter reporter.ProgressReporter) (*InitWorkflow, error) {
		awsClient := awsInternal.NewTestClient(mockClient)
		return NewInitWorkflowWithClient(awsClient, prompter, reporter), nil
	}

	mockReporter := &reporterTesting.MockReporter{}

	executor := NewExecutorWithFactory(mockReporter, mockPrompter, workflowFactory)

	// Test with empty flags to trigger interactive selection
	opts := &Options{
		Application: "", // Empty - should trigger interactive selection
		Profile:     "", // Empty - should trigger interactive selection
		Environment: "", // Empty - should trigger interactive selection
		Region:      "us-east-1",
		ConfigFile:  configPath,
		OutputData:  dataFilename,
		Force:       false,
	}

	err = executor.Execute(context.Background(), opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify interactive selection was triggered
	if selectCallCount != 3 {
		t.Errorf("expected 3 interactive selections (app, profile, env), got %d", selectCallCount)
	}

	// Verify success messages
	foundSelection := false
	for _, msg := range mockReporter.Messages {
		if strings.Contains(msg, "Selected") {
			foundSelection = true
			break
		}
	}
	if !foundSelection {
		t.Error("expected selection success messages from interactive prompts")
	}
}

// TestExecutorFactoryError tests error handling when factory fails
func TestExecutorFactoryError(t *testing.T) {
	t.Parallel()

	mockReporter := &reporterTesting.MockReporter{}
	mockPrompter := &promptTesting.MockPrompter{}

	// Create factory that returns an error
	workflowFactory := func(ctx context.Context, opts *Options, prompter prompt.Prompter, reporter reporter.ProgressReporter) (*InitWorkflow, error) {
		return nil, fmt.Errorf("factory error")
	}

	executor := NewExecutorWithFactory(mockReporter, mockPrompter, workflowFactory)

	opts := &Options{
		Application: "test-app",
		Profile:     "test-profile",
		Environment: "test-env",
		Region:      "us-east-1",
		ConfigFile:  "apcdeploy.yml",
	}

	err := executor.Execute(context.Background(), opts)

	if err == nil {
		t.Fatal("expected error from factory")
	}

	if !strings.Contains(err.Error(), "failed to create init workflow") {
		t.Errorf("expected 'failed to create init workflow' error, got: %v", err)
	}
}

// TestExecutorTTYErrorFromFactory tests that TTY errors from factory are returned as-is
func TestExecutorTTYErrorFromFactory(t *testing.T) {
	t.Parallel()

	mockReporter := &reporterTesting.MockReporter{}
	mockPrompter := &promptTesting.MockPrompter{}

	// Create factory that returns a TTY error
	workflowFactory := func(ctx context.Context, opts *Options, prompter prompt.Prompter, reporter reporter.ProgressReporter) (*InitWorkflow, error) {
		return nil, fmt.Errorf("%w: please provide --region, --app, --profile, and --env flags", prompt.ErrNoTTY)
	}

	executor := NewExecutorWithFactory(mockReporter, mockPrompter, workflowFactory)

	opts := &Options{
		Application: "test-app",
		Profile:     "test-profile",
		Environment: "test-env",
		Region:      "",
		ConfigFile:  "apcdeploy.yml",
	}

	err := executor.Execute(context.Background(), opts)

	if err == nil {
		t.Fatal("expected error from factory")
	}

	// TTY error should NOT be wrapped with "failed to create init workflow"
	if strings.Contains(err.Error(), "failed to create init workflow") {
		t.Errorf("TTY error should not be wrapped, got: %v", err)
	}

	if !strings.Contains(err.Error(), "interactive mode requires a TTY") {
		t.Errorf("expected 'interactive mode requires a TTY' error, got: %v", err)
	}
}

// TestExecutorTTYErrorFromWorkflowRun tests that TTY errors from workflow.Run are returned as-is
func TestExecutorTTYErrorFromWorkflowRun(t *testing.T) {
	t.Parallel()

	mockReporter := &reporterTesting.MockReporter{}
	mockPrompter := &promptTesting.MockPrompter{
		CheckTTYFunc: func() error {
			return prompt.ErrNoTTY
		},
	}

	// Create mock AWS client
	mockClient := &awsMock.MockAppConfigClient{}

	// Create workflow factory that uses the mock client
	workflowFactory := func(ctx context.Context, opts *Options, prompter prompt.Prompter, reporter reporter.ProgressReporter) (*InitWorkflow, error) {
		awsClient := awsInternal.NewTestClient(mockClient)
		return NewInitWorkflowWithClient(awsClient, prompter, reporter), nil
	}

	executor := NewExecutorWithFactory(mockReporter, mockPrompter, workflowFactory)

	// Empty flags to trigger TTY check in workflow.Run
	opts := &Options{
		Application: "", // Empty to trigger interactive mode check
		Profile:     "test-profile",
		Environment: "test-env",
		Region:      "us-east-1",
		ConfigFile:  "apcdeploy.yml",
	}

	err := executor.Execute(context.Background(), opts)

	if err == nil {
		t.Fatal("expected error from workflow.Run")
	}

	// TTY error should be returned as-is without wrapping
	if !strings.Contains(err.Error(), "interactive mode requires a TTY") {
		t.Errorf("expected 'interactive mode requires a TTY' error, got: %v", err)
	}

	if !strings.Contains(err.Error(), "please provide --region, --app, --profile, and --env flags") {
		t.Errorf("expected helpful message about all required flags, got: %v", err)
	}
}
