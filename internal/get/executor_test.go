package get

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/appconfig"
	"github.com/aws/aws-sdk-go-v2/service/appconfig/types"
	"github.com/aws/aws-sdk-go-v2/service/appconfigdata"
	awsInternal "github.com/koh-sh/apcdeploy/internal/aws"
	"github.com/koh-sh/apcdeploy/internal/aws/mock"
	"github.com/koh-sh/apcdeploy/internal/config"
	"github.com/koh-sh/apcdeploy/internal/prompt"
	prompttest "github.com/koh-sh/apcdeploy/internal/prompt/testing"
	reportertest "github.com/koh-sh/apcdeploy/internal/reporter/testing"
)

func TestNewExecutor(t *testing.T) {
	t.Parallel()

	reporter := &reportertest.MockReporter{}
	prompter := &prompttest.MockPrompter{}
	executor := NewExecutor(reporter, prompter)

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

	// Verify reporter was called for progress
	if len(reporter.Messages) == 0 {
		t.Error("expected reporter to have received messages")
	}

	if !strings.Contains(reporter.Messages[0], "Loading configuration") {
		t.Errorf("expected first message to be about loading configuration, got: %v", reporter.Messages[0])
	}
}

// TestExecutorFullWorkflowWithMock tests the complete get workflow with mocked AWS
func TestExecutorFullWorkflowWithMock(t *testing.T) {
	t.Parallel()

	// Create temporary test files
	tempDir, err := os.MkdirTemp("", "get-executor-full-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create config file
	configPath := filepath.Join(tempDir, "apcdeploy.yml")
	configContent := `application: test-app
configuration_profile: test-profile
environment: test-env
data_file: data.json
region: us-east-1
`
	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	// Create data file (not needed for get command, but config loader expects it)
	dataPath := filepath.Join(tempDir, "data.json")
	if err := os.WriteFile(dataPath, []byte(`{"key": "value"}`), 0o644); err != nil {
		t.Fatalf("Failed to write data: %v", err)
	}

	// Configuration content to be returned by appconfigdata
	configData := []byte(`{"feature": "enabled", "value": 123}`)

	// Create mock AWS clients
	mockAppConfigClient := &mock.MockAppConfigClient{
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
			// Return empty list since deployment strategy is not needed for get command
			return &appconfig.ListDeploymentStrategiesOutput{
				Items: []types.DeploymentStrategy{},
			}, nil
		},
	}

	mockAppConfigDataClient := &mock.MockAppConfigDataClient{
		StartConfigurationSessionFunc: func(ctx context.Context, params *appconfigdata.StartConfigurationSessionInput, optFns ...func(*appconfigdata.Options)) (*appconfigdata.StartConfigurationSessionOutput, error) {
			return &appconfigdata.StartConfigurationSessionOutput{
				InitialConfigurationToken: aws.String("initial-token"),
			}, nil
		},
		GetLatestConfigurationFunc: func(ctx context.Context, params *appconfigdata.GetLatestConfigurationInput, optFns ...func(*appconfigdata.Options)) (*appconfigdata.GetLatestConfigurationOutput, error) {
			return &appconfigdata.GetLatestConfigurationOutput{
				Configuration: configData,
			}, nil
		},
	}

	// Create getter factory that uses the mock client
	getterFactory := func(ctx context.Context, cfg *config.Config) (*Getter, error) {
		awsClient := &awsInternal.Client{
			AppConfig:     mockAppConfigClient,
			AppConfigData: mockAppConfigDataClient,
		}
		return NewWithClient(cfg, awsClient), nil
	}

	reporter := &reportertest.MockReporter{}
	prompter := &prompttest.MockPrompter{}
	executor := NewExecutorWithFactory(reporter, prompter, getterFactory)

	opts := &Options{
		ConfigFile:       configPath,
		SkipConfirmation: true, // Skip confirmation for this test
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
		"Fetching latest configuration",
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

// TestExecutorResolveResourcesError tests error during resource resolution
func TestExecutorResolveResourcesError(t *testing.T) {
	t.Parallel()

	tempDir, err := os.MkdirTemp("", "get-executor-resolve-*")
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
	if err := os.WriteFile(dataPath, []byte(`{}`), 0o644); err != nil {
		t.Fatalf("Failed to write data: %v", err)
	}

	// Mock that returns empty lists (resource not found)
	mockAppConfigClient := &mock.MockAppConfigClient{
		ListApplicationsFunc: func(ctx context.Context, params *appconfig.ListApplicationsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListApplicationsOutput, error) {
			return &appconfig.ListApplicationsOutput{
				Items: []types.Application{},
			}, nil
		},
	}

	getterFactory := func(ctx context.Context, cfg *config.Config) (*Getter, error) {
		awsClient := &awsInternal.Client{
			AppConfig: mockAppConfigClient,
		}
		return NewWithClient(cfg, awsClient), nil
	}

	reporter := &reportertest.MockReporter{}
	prompter := &prompttest.MockPrompter{}
	executor := NewExecutorWithFactory(reporter, prompter, getterFactory)

	opts := &Options{
		ConfigFile: configPath,
	}

	err = executor.Execute(context.Background(), opts)

	if err == nil {
		t.Fatal("expected error when application not found")
	}

	if !strings.Contains(err.Error(), "failed to resolve resources") {
		t.Errorf("expected 'failed to resolve resources' error, got: %v", err)
	}
}

// TestExecutorGetterFactoryError tests error during getter creation
func TestExecutorGetterFactoryError(t *testing.T) {
	t.Parallel()

	tempDir, err := os.MkdirTemp("", "get-executor-factory-*")
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
	if err := os.WriteFile(dataPath, []byte(`{}`), 0o644); err != nil {
		t.Fatalf("Failed to write data: %v", err)
	}

	getterFactory := func(ctx context.Context, cfg *config.Config) (*Getter, error) {
		return nil, errors.New("factory error")
	}

	reporter := &reportertest.MockReporter{}
	prompter := &prompttest.MockPrompter{}
	executor := NewExecutorWithFactory(reporter, prompter, getterFactory)

	opts := &Options{
		ConfigFile: configPath,
	}

	err = executor.Execute(context.Background(), opts)

	if err == nil {
		t.Fatal("expected error from factory")
	}

	if !strings.Contains(err.Error(), "failed to create getter") {
		t.Errorf("expected 'failed to create getter' error, got: %v", err)
	}
}

// TestExecutorGetConfigurationError tests error during configuration retrieval
func TestExecutorGetConfigurationError(t *testing.T) {
	t.Parallel()

	tempDir, err := os.MkdirTemp("", "get-executor-getconfig-*")
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
	if err := os.WriteFile(dataPath, []byte(`{}`), 0o644); err != nil {
		t.Fatalf("Failed to write data: %v", err)
	}

	mockAppConfigClient := &mock.MockAppConfigClient{
		ListApplicationsFunc: func(ctx context.Context, params *appconfig.ListApplicationsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListApplicationsOutput, error) {
			return &appconfig.ListApplicationsOutput{
				Items: []types.Application{{Id: aws.String("app-123"), Name: aws.String("test-app")}},
			}, nil
		},
		ListConfigurationProfilesFunc: func(ctx context.Context, params *appconfig.ListConfigurationProfilesInput, optFns ...func(*appconfig.Options)) (*appconfig.ListConfigurationProfilesOutput, error) {
			return &appconfig.ListConfigurationProfilesOutput{
				Items: []types.ConfigurationProfileSummary{{Id: aws.String("profile-123"), Name: aws.String("test-profile")}},
			}, nil
		},
		GetConfigurationProfileFunc: func(ctx context.Context, params *appconfig.GetConfigurationProfileInput, optFns ...func(*appconfig.Options)) (*appconfig.GetConfigurationProfileOutput, error) {
			return &appconfig.GetConfigurationProfileOutput{
				Id:   aws.String("profile-123"),
				Type: aws.String("AWS.Freeform"),
			}, nil
		},
		ListEnvironmentsFunc: func(ctx context.Context, params *appconfig.ListEnvironmentsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListEnvironmentsOutput, error) {
			return &appconfig.ListEnvironmentsOutput{
				Items: []types.Environment{{Id: aws.String("env-123"), Name: aws.String("test-env")}},
			}, nil
		},
	}

	mockAppConfigDataClient := &mock.MockAppConfigDataClient{
		StartConfigurationSessionFunc: func(ctx context.Context, params *appconfigdata.StartConfigurationSessionInput, optFns ...func(*appconfigdata.Options)) (*appconfigdata.StartConfigurationSessionOutput, error) {
			return nil, errors.New("session error")
		},
	}

	getterFactory := func(ctx context.Context, cfg *config.Config) (*Getter, error) {
		awsClient := &awsInternal.Client{
			AppConfig:     mockAppConfigClient,
			AppConfigData: mockAppConfigDataClient,
		}
		return NewWithClient(cfg, awsClient), nil
	}

	reporter := &reportertest.MockReporter{}
	prompter := &prompttest.MockPrompter{}
	executor := NewExecutorWithFactory(reporter, prompter, getterFactory)

	opts := &Options{
		ConfigFile:       configPath,
		SkipConfirmation: true, // Skip confirmation for this test
	}

	err = executor.Execute(context.Background(), opts)

	if err == nil {
		t.Fatal("expected error during configuration retrieval")
	}

	if !strings.Contains(err.Error(), "failed to get configuration") {
		t.Errorf("expected 'failed to get configuration' error, got: %v", err)
	}
}

// TestExecutorResolveProfileError tests error during profile resolution
func TestExecutorResolveProfileError(t *testing.T) {
	t.Parallel()

	tempDir, err := os.MkdirTemp("", "get-executor-profile-*")
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
	if err := os.WriteFile(dataPath, []byte(`{}`), 0o644); err != nil {
		t.Fatalf("Failed to write data: %v", err)
	}

	mockAppConfigClient := &mock.MockAppConfigClient{
		ListApplicationsFunc: func(ctx context.Context, params *appconfig.ListApplicationsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListApplicationsOutput, error) {
			return &appconfig.ListApplicationsOutput{
				Items: []types.Application{{Id: aws.String("app-123"), Name: aws.String("test-app")}},
			}, nil
		},
		ListConfigurationProfilesFunc: func(ctx context.Context, params *appconfig.ListConfigurationProfilesInput, optFns ...func(*appconfig.Options)) (*appconfig.ListConfigurationProfilesOutput, error) {
			return &appconfig.ListConfigurationProfilesOutput{
				Items: []types.ConfigurationProfileSummary{},
			}, nil
		},
	}

	getterFactory := func(ctx context.Context, cfg *config.Config) (*Getter, error) {
		awsClient := &awsInternal.Client{
			AppConfig: mockAppConfigClient,
		}
		return NewWithClient(cfg, awsClient), nil
	}

	reporter := &reportertest.MockReporter{}
	prompter := &prompttest.MockPrompter{}
	executor := NewExecutorWithFactory(reporter, prompter, getterFactory)

	opts := &Options{
		ConfigFile: configPath,
	}

	err = executor.Execute(context.Background(), opts)

	if err == nil {
		t.Fatal("expected error when profile not found")
	}

	if !strings.Contains(err.Error(), "failed to resolve") {
		t.Errorf("expected 'failed to resolve' error, got: %v", err)
	}
}

// TestExecutorResolveEnvironmentError tests error during environment resolution
func TestExecutorResolveEnvironmentError(t *testing.T) {
	t.Parallel()

	tempDir, err := os.MkdirTemp("", "get-executor-env-*")
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
	if err := os.WriteFile(dataPath, []byte(`{}`), 0o644); err != nil {
		t.Fatalf("Failed to write data: %v", err)
	}

	mockAppConfigClient := &mock.MockAppConfigClient{
		ListApplicationsFunc: func(ctx context.Context, params *appconfig.ListApplicationsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListApplicationsOutput, error) {
			return &appconfig.ListApplicationsOutput{
				Items: []types.Application{{Id: aws.String("app-123"), Name: aws.String("test-app")}},
			}, nil
		},
		ListConfigurationProfilesFunc: func(ctx context.Context, params *appconfig.ListConfigurationProfilesInput, optFns ...func(*appconfig.Options)) (*appconfig.ListConfigurationProfilesOutput, error) {
			return &appconfig.ListConfigurationProfilesOutput{
				Items: []types.ConfigurationProfileSummary{{Id: aws.String("profile-123"), Name: aws.String("test-profile")}},
			}, nil
		},
		GetConfigurationProfileFunc: func(ctx context.Context, params *appconfig.GetConfigurationProfileInput, optFns ...func(*appconfig.Options)) (*appconfig.GetConfigurationProfileOutput, error) {
			return &appconfig.GetConfigurationProfileOutput{
				Id:   aws.String("profile-123"),
				Type: aws.String("AWS.Freeform"),
			}, nil
		},
		ListEnvironmentsFunc: func(ctx context.Context, params *appconfig.ListEnvironmentsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListEnvironmentsOutput, error) {
			return &appconfig.ListEnvironmentsOutput{
				Items: []types.Environment{},
			}, nil
		},
	}

	getterFactory := func(ctx context.Context, cfg *config.Config) (*Getter, error) {
		awsClient := &awsInternal.Client{
			AppConfig: mockAppConfigClient,
		}
		return NewWithClient(cfg, awsClient), nil
	}

	reporter := &reportertest.MockReporter{}
	prompter := &prompttest.MockPrompter{}
	executor := NewExecutorWithFactory(reporter, prompter, getterFactory)

	opts := &Options{
		ConfigFile: configPath,
	}

	err = executor.Execute(context.Background(), opts)

	if err == nil {
		t.Fatal("expected error when environment not found")
	}

	if !strings.Contains(err.Error(), "failed to resolve") {
		t.Errorf("expected 'failed to resolve' error, got: %v", err)
	}
}

// TestExecutorTTYCheckFailure tests TTY check failure when confirmation is required
func TestExecutorTTYCheckFailure(t *testing.T) {
	t.Parallel()

	tempDir, err := os.MkdirTemp("", "get-executor-tty-*")
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
	if err := os.WriteFile(dataPath, []byte(`{}`), 0o644); err != nil {
		t.Fatalf("Failed to write data: %v", err)
	}

	mockAppConfigClient := &mock.MockAppConfigClient{
		ListApplicationsFunc: func(ctx context.Context, params *appconfig.ListApplicationsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListApplicationsOutput, error) {
			return &appconfig.ListApplicationsOutput{
				Items: []types.Application{{Id: aws.String("app-123"), Name: aws.String("test-app")}},
			}, nil
		},
		ListConfigurationProfilesFunc: func(ctx context.Context, params *appconfig.ListConfigurationProfilesInput, optFns ...func(*appconfig.Options)) (*appconfig.ListConfigurationProfilesOutput, error) {
			return &appconfig.ListConfigurationProfilesOutput{
				Items: []types.ConfigurationProfileSummary{{Id: aws.String("profile-123"), Name: aws.String("test-profile")}},
			}, nil
		},
		GetConfigurationProfileFunc: func(ctx context.Context, params *appconfig.GetConfigurationProfileInput, optFns ...func(*appconfig.Options)) (*appconfig.GetConfigurationProfileOutput, error) {
			return &appconfig.GetConfigurationProfileOutput{
				Id:   aws.String("profile-123"),
				Type: aws.String("AWS.Freeform"),
			}, nil
		},
		ListEnvironmentsFunc: func(ctx context.Context, params *appconfig.ListEnvironmentsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListEnvironmentsOutput, error) {
			return &appconfig.ListEnvironmentsOutput{
				Items: []types.Environment{{Id: aws.String("env-123"), Name: aws.String("test-env")}},
			}, nil
		},
	}

	getterFactory := func(ctx context.Context, cfg *config.Config) (*Getter, error) {
		awsClient := &awsInternal.Client{
			AppConfig: mockAppConfigClient,
		}
		return NewWithClient(cfg, awsClient), nil
	}

	mockPrompter := &prompttest.MockPrompter{
		CheckTTYFunc: func() error {
			return prompt.ErrNoTTY
		},
	}

	reporter := &reportertest.MockReporter{}
	executor := NewExecutorWithFactory(reporter, mockPrompter, getterFactory)

	opts := &Options{
		ConfigFile:       configPath,
		SkipConfirmation: false, // Require confirmation
	}

	err = executor.Execute(context.Background(), opts)

	if err == nil {
		t.Fatal("expected error when TTY check fails")
	}

	if !strings.Contains(err.Error(), "interactive mode requires a TTY") {
		t.Errorf("expected 'interactive mode requires a TTY' error, got: %v", err)
	}

	if !strings.Contains(err.Error(), "use --yes to skip confirmation") {
		t.Errorf("expected error to suggest --yes flag, got: %v", err)
	}
}

// TestExecutorConfirmationPrompt tests confirmation prompt behavior
func TestExecutorConfirmationPrompt(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		skipConfirmation  bool
		userResponse      string
		expectError       bool
		expectPromptCalls int
	}{
		{
			name:              "User confirms with Yes",
			skipConfirmation:  false,
			userResponse:      "Yes",
			expectError:       false,
			expectPromptCalls: 1,
		},
		{
			name:              "User confirms with yes (lowercase)",
			skipConfirmation:  false,
			userResponse:      "yes",
			expectError:       false,
			expectPromptCalls: 1,
		},
		{
			name:              "User confirms with Y",
			skipConfirmation:  false,
			userResponse:      "Y",
			expectError:       false,
			expectPromptCalls: 1,
		},
		{
			name:              "User confirms with y (lowercase)",
			skipConfirmation:  false,
			userResponse:      "y",
			expectError:       false,
			expectPromptCalls: 1,
		},
		{
			name:              "User confirms with Y and whitespace",
			skipConfirmation:  false,
			userResponse:      " Y ",
			expectError:       false,
			expectPromptCalls: 1,
		},
		{
			name:              "User declines with No",
			skipConfirmation:  false,
			userResponse:      "No",
			expectError:       true,
			expectPromptCalls: 1,
		},
		{
			name:              "User declines with empty input",
			skipConfirmation:  false,
			userResponse:      "",
			expectError:       true,
			expectPromptCalls: 1,
		},
		{
			name:              "User declines with invalid input",
			skipConfirmation:  false,
			userResponse:      "maybe",
			expectError:       true,
			expectPromptCalls: 1,
		},
		{
			name:              "Skip confirmation with flag",
			skipConfirmation:  true,
			userResponse:      "",
			expectError:       false,
			expectPromptCalls: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tempDir, err := os.MkdirTemp("", "get-executor-confirm-*")
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
			if err := os.WriteFile(dataPath, []byte(`{}`), 0o644); err != nil {
				t.Fatalf("Failed to write data: %v", err)
			}

			configData := []byte(`{"feature": "enabled"}`)

			mockAppConfigClient := &mock.MockAppConfigClient{
				ListApplicationsFunc: func(ctx context.Context, params *appconfig.ListApplicationsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListApplicationsOutput, error) {
					return &appconfig.ListApplicationsOutput{
						Items: []types.Application{{Id: aws.String("app-123"), Name: aws.String("test-app")}},
					}, nil
				},
				ListConfigurationProfilesFunc: func(ctx context.Context, params *appconfig.ListConfigurationProfilesInput, optFns ...func(*appconfig.Options)) (*appconfig.ListConfigurationProfilesOutput, error) {
					return &appconfig.ListConfigurationProfilesOutput{
						Items: []types.ConfigurationProfileSummary{{Id: aws.String("profile-123"), Name: aws.String("test-profile")}},
					}, nil
				},
				GetConfigurationProfileFunc: func(ctx context.Context, params *appconfig.GetConfigurationProfileInput, optFns ...func(*appconfig.Options)) (*appconfig.GetConfigurationProfileOutput, error) {
					return &appconfig.GetConfigurationProfileOutput{
						Id:   aws.String("profile-123"),
						Type: aws.String("AWS.Freeform"),
					}, nil
				},
				ListEnvironmentsFunc: func(ctx context.Context, params *appconfig.ListEnvironmentsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListEnvironmentsOutput, error) {
					return &appconfig.ListEnvironmentsOutput{
						Items: []types.Environment{{Id: aws.String("env-123"), Name: aws.String("test-env")}},
					}, nil
				},
			}

			mockAppConfigDataClient := &mock.MockAppConfigDataClient{
				StartConfigurationSessionFunc: func(ctx context.Context, params *appconfigdata.StartConfigurationSessionInput, optFns ...func(*appconfigdata.Options)) (*appconfigdata.StartConfigurationSessionOutput, error) {
					return &appconfigdata.StartConfigurationSessionOutput{
						InitialConfigurationToken: aws.String("initial-token"),
					}, nil
				},
				GetLatestConfigurationFunc: func(ctx context.Context, params *appconfigdata.GetLatestConfigurationInput, optFns ...func(*appconfigdata.Options)) (*appconfigdata.GetLatestConfigurationOutput, error) {
					return &appconfigdata.GetLatestConfigurationOutput{
						Configuration: configData,
					}, nil
				},
			}

			getterFactory := func(ctx context.Context, cfg *config.Config) (*Getter, error) {
				awsClient := &awsInternal.Client{
					AppConfig:     mockAppConfigClient,
					AppConfigData: mockAppConfigDataClient,
				}
				return NewWithClient(cfg, awsClient), nil
			}

			promptCalls := 0
			mockPrompter := &prompttest.MockPrompter{
				InputFunc: func(message string, placeholder string) (string, error) {
					promptCalls++
					return tt.userResponse, nil
				},
			}

			reporter := &reportertest.MockReporter{}
			executor := NewExecutorWithFactory(reporter, mockPrompter, getterFactory)

			opts := &Options{
				ConfigFile:       configPath,
				SkipConfirmation: tt.skipConfirmation,
			}

			err = executor.Execute(context.Background(), opts)

			if tt.expectError && err == nil {
				t.Fatal("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if promptCalls != tt.expectPromptCalls {
				t.Errorf("expected %d prompt calls, got %d", tt.expectPromptCalls, promptCalls)
			}
		})
	}
}
