package pull

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
	awsInternal "github.com/koh-sh/apcdeploy/internal/aws"
	"github.com/koh-sh/apcdeploy/internal/aws/mock"
	reportertest "github.com/koh-sh/apcdeploy/internal/reporter/testing"
)

func TestNewExecutor(t *testing.T) {
	t.Parallel()

	reporter := &reportertest.MockReporter{}
	executor := NewExecutor(reporter)

	if executor.reporter != reporter {
		t.Error("expected executor to have the provided reporter")
	}
}

func TestExecutorLoadConfigurationError(t *testing.T) {
	t.Parallel()

	reporter := &reportertest.MockReporter{}
	executor := NewExecutor(reporter)

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

// TestExecutorFullWorkflowWithMock tests the complete pull workflow with mocked AWS
func TestExecutorFullWorkflowWithMock(t *testing.T) {
	t.Parallel()

	// Create temporary test files
	tempDir, err := os.MkdirTemp("", "pull-executor-full-*")
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

	// Create data file (existing local data)
	dataPath := filepath.Join(tempDir, "data.json")
	if err := os.WriteFile(dataPath, []byte(`{"key": "old-value"}`), 0o644); err != nil {
		t.Fatalf("Failed to write data: %v", err)
	}

	// Configuration content to be returned by AWS
	remoteConfigData := []byte(`{"key": "new-value", "feature": "enabled"}`)

	// Create mock AWS client
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
		ListDeploymentsFunc: func(ctx context.Context, params *appconfig.ListDeploymentsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListDeploymentsOutput, error) {
			return &appconfig.ListDeploymentsOutput{
				Items: []types.DeploymentSummary{
					{
						DeploymentNumber: 1,
						State:            types.DeploymentStateComplete,
					},
				},
			}, nil
		},
		GetDeploymentFunc: func(ctx context.Context, params *appconfig.GetDeploymentInput, optFns ...func(*appconfig.Options)) (*appconfig.GetDeploymentOutput, error) {
			return &appconfig.GetDeploymentOutput{
				ApplicationId:          aws.String("app-123"),
				EnvironmentId:          aws.String("env-123"),
				DeploymentNumber:       1,
				ConfigurationProfileId: aws.String("profile-123"),
				ConfigurationVersion:   aws.String("1"),
				State:                  types.DeploymentStateComplete,
			}, nil
		},
		GetHostedConfigurationVersionFunc: func(ctx context.Context, params *appconfig.GetHostedConfigurationVersionInput, optFns ...func(*appconfig.Options)) (*appconfig.GetHostedConfigurationVersionOutput, error) {
			return &appconfig.GetHostedConfigurationVersionOutput{
				ApplicationId:          aws.String("app-123"),
				ConfigurationProfileId: aws.String("profile-123"),
				VersionNumber:          1,
				Content:                remoteConfigData,
				ContentType:            aws.String("application/json"),
			}, nil
		},
	}

	// Create client factory that uses the mock client
	clientFactory := func(ctx context.Context, region string) (*awsInternal.Client, error) {
		return awsInternal.NewTestClient(mockAppConfigClient), nil
	}

	reporter := &reportertest.MockReporter{}
	executor := NewExecutorWithFactory(reporter, clientFactory)

	opts := &Options{
		ConfigFile: configPath,
	}

	err = executor.Execute(context.Background(), opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify data file was updated
	updatedData, err := os.ReadFile(dataPath)
	if err != nil {
		t.Fatalf("Failed to read updated data file: %v", err)
	}

	// Should contain the new remote data (formatted JSON)
	if !strings.Contains(string(updatedData), `"new-value"`) {
		t.Errorf("expected data file to contain new value, got: %s", string(updatedData))
	}

	// Verify all expected messages were reported
	expectedMessages := []string{
		"Loading configuration",
		"Resolving resources",
		"Fetching latest deployment",
		"Fetching deployed configuration",
		"Updating data file",
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

// TestExecutorNoDeploymentFound tests error when no deployment exists
func TestExecutorNoDeploymentFound(t *testing.T) {
	t.Parallel()

	tempDir, err := os.MkdirTemp("", "pull-executor-nodepl-*")
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
		ListDeploymentsFunc: func(ctx context.Context, params *appconfig.ListDeploymentsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListDeploymentsOutput, error) {
			// Return empty list - no deployments
			return &appconfig.ListDeploymentsOutput{
				Items: []types.DeploymentSummary{},
			}, nil
		},
	}

	clientFactory := func(ctx context.Context, region string) (*awsInternal.Client, error) {
		return awsInternal.NewTestClient(mockAppConfigClient), nil
	}

	reporter := &reportertest.MockReporter{}
	executor := NewExecutorWithFactory(reporter, clientFactory)

	opts := &Options{
		ConfigFile: configPath,
	}

	err = executor.Execute(context.Background(), opts)

	if err == nil {
		t.Fatal("expected error when no deployment found")
	}

	if !strings.Contains(err.Error(), "no deployment found") {
		t.Errorf("expected 'no deployment found' error, got: %v", err)
	}
}

// TestExecutorResolveResourcesError tests error during resource resolution
func TestExecutorResolveResourcesError(t *testing.T) {
	t.Parallel()

	tempDir, err := os.MkdirTemp("", "pull-executor-resolve-*")
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

	clientFactory := func(ctx context.Context, region string) (*awsInternal.Client, error) {
		return awsInternal.NewTestClient(mockAppConfigClient), nil
	}

	reporter := &reportertest.MockReporter{}
	executor := NewExecutorWithFactory(reporter, clientFactory)

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

// TestExecutorClientFactoryError tests error during client creation
func TestExecutorClientFactoryError(t *testing.T) {
	t.Parallel()

	tempDir, err := os.MkdirTemp("", "pull-executor-factory-*")
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

	clientFactory := func(ctx context.Context, region string) (*awsInternal.Client, error) {
		return nil, errors.New("factory error")
	}

	reporter := &reportertest.MockReporter{}
	executor := NewExecutorWithFactory(reporter, clientFactory)

	opts := &Options{
		ConfigFile: configPath,
	}

	err = executor.Execute(context.Background(), opts)

	if err == nil {
		t.Fatal("expected error from factory")
	}

	if !strings.Contains(err.Error(), "failed to initialize AWS client") {
		t.Errorf("expected 'failed to initialize AWS client' error, got: %v", err)
	}
}

// TestExecutorGetDeploymentError tests error during deployment retrieval
func TestExecutorGetDeploymentError(t *testing.T) {
	t.Parallel()

	tempDir, err := os.MkdirTemp("", "pull-executor-getdepl-*")
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
		ListDeploymentsFunc: func(ctx context.Context, params *appconfig.ListDeploymentsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListDeploymentsOutput, error) {
			return nil, errors.New("API error")
		},
	}

	clientFactory := func(ctx context.Context, region string) (*awsInternal.Client, error) {
		return awsInternal.NewTestClient(mockAppConfigClient), nil
	}

	reporter := &reportertest.MockReporter{}
	executor := NewExecutorWithFactory(reporter, clientFactory)

	opts := &Options{
		ConfigFile: configPath,
	}

	err = executor.Execute(context.Background(), opts)

	if err == nil {
		t.Fatal("expected error during deployment retrieval")
	}

	if !strings.Contains(err.Error(), "failed to get latest deployment") {
		t.Errorf("expected 'failed to get latest deployment' error, got: %v", err)
	}
}

// TestExecutorGetConfigurationVersionError tests error during configuration version retrieval
func TestExecutorGetConfigurationVersionError(t *testing.T) {
	t.Parallel()

	tempDir, err := os.MkdirTemp("", "pull-executor-getver-*")
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
		ListDeploymentsFunc: func(ctx context.Context, params *appconfig.ListDeploymentsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListDeploymentsOutput, error) {
			return &appconfig.ListDeploymentsOutput{
				Items: []types.DeploymentSummary{{DeploymentNumber: 1, State: types.DeploymentStateComplete}},
			}, nil
		},
		GetDeploymentFunc: func(ctx context.Context, params *appconfig.GetDeploymentInput, optFns ...func(*appconfig.Options)) (*appconfig.GetDeploymentOutput, error) {
			return &appconfig.GetDeploymentOutput{
				ApplicationId:          aws.String("app-123"),
				EnvironmentId:          aws.String("env-123"),
				DeploymentNumber:       1,
				ConfigurationProfileId: aws.String("profile-123"),
				ConfigurationVersion:   aws.String("1"),
				State:                  types.DeploymentStateComplete,
			}, nil
		},
		GetHostedConfigurationVersionFunc: func(ctx context.Context, params *appconfig.GetHostedConfigurationVersionInput, optFns ...func(*appconfig.Options)) (*appconfig.GetHostedConfigurationVersionOutput, error) {
			return nil, errors.New("version not found")
		},
	}

	clientFactory := func(ctx context.Context, region string) (*awsInternal.Client, error) {
		return awsInternal.NewTestClient(mockAppConfigClient), nil
	}

	reporter := &reportertest.MockReporter{}
	executor := NewExecutorWithFactory(reporter, clientFactory)

	opts := &Options{
		ConfigFile: configPath,
	}

	err = executor.Execute(context.Background(), opts)

	if err == nil {
		t.Fatal("expected error during configuration version retrieval")
	}

	if !strings.Contains(err.Error(), "failed to get deployed configuration") {
		t.Errorf("expected 'failed to get deployed configuration' error, got: %v", err)
	}
}

// TestExecutorNoChanges tests pull when local and remote are identical
func TestExecutorNoChanges(t *testing.T) {
	t.Parallel()

	tempDir, err := os.MkdirTemp("", "pull-executor-nochange-*")
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

	// Remote and local data are identical (after normalization)
	identicalData := []byte(`{
  "key": "value",
  "feature": "enabled"
}`)

	dataPath := filepath.Join(tempDir, "data.json")
	if err := os.WriteFile(dataPath, identicalData, 0o644); err != nil {
		t.Fatalf("Failed to write data: %v", err)
	}

	// Store original modification time
	originalInfo, err := os.Stat(dataPath)
	if err != nil {
		t.Fatalf("Failed to stat data file: %v", err)
	}
	originalModTime := originalInfo.ModTime()

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
		ListDeploymentsFunc: func(ctx context.Context, params *appconfig.ListDeploymentsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListDeploymentsOutput, error) {
			return &appconfig.ListDeploymentsOutput{
				Items: []types.DeploymentSummary{{DeploymentNumber: 1, State: types.DeploymentStateComplete}},
			}, nil
		},
		GetDeploymentFunc: func(ctx context.Context, params *appconfig.GetDeploymentInput, optFns ...func(*appconfig.Options)) (*appconfig.GetDeploymentOutput, error) {
			return &appconfig.GetDeploymentOutput{
				ApplicationId:          aws.String("app-123"),
				EnvironmentId:          aws.String("env-123"),
				DeploymentNumber:       1,
				ConfigurationProfileId: aws.String("profile-123"),
				ConfigurationVersion:   aws.String("1"),
				State:                  types.DeploymentStateComplete,
			}, nil
		},
		GetHostedConfigurationVersionFunc: func(ctx context.Context, params *appconfig.GetHostedConfigurationVersionInput, optFns ...func(*appconfig.Options)) (*appconfig.GetHostedConfigurationVersionOutput, error) {
			return &appconfig.GetHostedConfigurationVersionOutput{
				ApplicationId:          aws.String("app-123"),
				ConfigurationProfileId: aws.String("profile-123"),
				VersionNumber:          1,
				Content:                identicalData,
				ContentType:            aws.String("application/json"),
			}, nil
		},
	}

	clientFactory := func(ctx context.Context, region string) (*awsInternal.Client, error) {
		return awsInternal.NewTestClient(mockAppConfigClient), nil
	}

	reporter := &reportertest.MockReporter{}
	executor := NewExecutorWithFactory(reporter, clientFactory)

	opts := &Options{
		ConfigFile: configPath,
	}

	err = executor.Execute(context.Background(), opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify file was NOT modified
	newInfo, err := os.Stat(dataPath)
	if err != nil {
		t.Fatalf("Failed to stat data file: %v", err)
	}
	newModTime := newInfo.ModTime()

	if !newModTime.Equal(originalModTime) {
		t.Error("expected data file to NOT be modified when no changes exist")
	}

	// Verify "no changes" message was reported
	foundNoChanges := false
	for _, msg := range reporter.Messages {
		if strings.Contains(msg, "No changes") || strings.Contains(msg, "already up to date") {
			foundNoChanges = true
			break
		}
	}
	if !foundNoChanges {
		t.Errorf("expected 'no changes' message in reporter output, got: %v", reporter.Messages)
	}
}

// TestExecutorFeatureFlagsProfile tests pull for FeatureFlags profile type
func TestExecutorFeatureFlagsProfile(t *testing.T) {
	t.Parallel()

	tempDir, err := os.MkdirTemp("", "pull-executor-ff-*")
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

	// Remote data with timestamp fields that should be removed
	remoteConfigData := []byte(`{
  "flags": {
    "feature1": {
      "name": "feature1",
      "_updatedAt": "2024-01-01T00:00:00Z",
      "_createdAt": "2024-01-01T00:00:00Z"
    }
  }
}`)

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
				Type: aws.String("AWS.AppConfig.FeatureFlags"),
			}, nil
		},
		ListEnvironmentsFunc: func(ctx context.Context, params *appconfig.ListEnvironmentsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListEnvironmentsOutput, error) {
			return &appconfig.ListEnvironmentsOutput{
				Items: []types.Environment{{Id: aws.String("env-123"), Name: aws.String("test-env")}},
			}, nil
		},
		ListDeploymentsFunc: func(ctx context.Context, params *appconfig.ListDeploymentsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListDeploymentsOutput, error) {
			return &appconfig.ListDeploymentsOutput{
				Items: []types.DeploymentSummary{{DeploymentNumber: 1, State: types.DeploymentStateComplete}},
			}, nil
		},
		GetDeploymentFunc: func(ctx context.Context, params *appconfig.GetDeploymentInput, optFns ...func(*appconfig.Options)) (*appconfig.GetDeploymentOutput, error) {
			return &appconfig.GetDeploymentOutput{
				ApplicationId:          aws.String("app-123"),
				EnvironmentId:          aws.String("env-123"),
				DeploymentNumber:       1,
				ConfigurationProfileId: aws.String("profile-123"),
				ConfigurationVersion:   aws.String("1"),
				State:                  types.DeploymentStateComplete,
			}, nil
		},
		GetHostedConfigurationVersionFunc: func(ctx context.Context, params *appconfig.GetHostedConfigurationVersionInput, optFns ...func(*appconfig.Options)) (*appconfig.GetHostedConfigurationVersionOutput, error) {
			return &appconfig.GetHostedConfigurationVersionOutput{
				ApplicationId:          aws.String("app-123"),
				ConfigurationProfileId: aws.String("profile-123"),
				VersionNumber:          1,
				Content:                remoteConfigData,
				ContentType:            aws.String("application/json"),
			}, nil
		},
	}

	clientFactory := func(ctx context.Context, region string) (*awsInternal.Client, error) {
		return awsInternal.NewTestClient(mockAppConfigClient), nil
	}

	reporter := &reportertest.MockReporter{}
	executor := NewExecutorWithFactory(reporter, clientFactory)

	opts := &Options{
		ConfigFile: configPath,
	}

	err = executor.Execute(context.Background(), opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify data file was updated and timestamp fields were removed
	updatedData, err := os.ReadFile(dataPath)
	if err != nil {
		t.Fatalf("Failed to read updated data file: %v", err)
	}

	// Should NOT contain timestamp fields
	if strings.Contains(string(updatedData), "_updatedAt") {
		t.Errorf("expected data file to not contain _updatedAt field, got: %s", string(updatedData))
	}
	if strings.Contains(string(updatedData), "_createdAt") {
		t.Errorf("expected data file to not contain _createdAt field, got: %s", string(updatedData))
	}

	// Should contain the feature data
	if !strings.Contains(string(updatedData), `"feature1"`) {
		t.Errorf("expected data file to contain feature1, got: %s", string(updatedData))
	}
}
