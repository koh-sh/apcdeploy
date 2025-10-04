package status

import (
	"context"
	"fmt"
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
	reportertest "github.com/koh-sh/apcdeploy/internal/reporter/testing"
)

func TestNewExecutor(t *testing.T) {
	reporter := &reportertest.MockReporter{}
	executor := NewExecutor(reporter)

	if executor == nil {
		t.Fatal("expected executor to be non-nil")
	}

	if executor.reporter != reporter {
		t.Error("expected executor to have the provided reporter")
	}
}

func TestExecutorLoadConfigurationError(t *testing.T) {
	reporter := &reportertest.MockReporter{}
	executor := NewExecutor(reporter)

	opts := &Options{
		ConfigFile: "nonexistent.yml",
		Region:     "us-east-1",
	}

	err := executor.Execute(context.Background(), opts)

	if err == nil {
		t.Error("expected error when loading non-existent config file")
	}

	if !strings.Contains(err.Error(), "failed to load configuration") {
		t.Errorf("expected 'failed to load configuration' error, got: %v", err)
	}
}

func TestExecutorNoDeployment(t *testing.T) {
	// Create temporary test files
	tempDir, err := os.MkdirTemp("", "executor-nodeploy-*")
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

	// Create mock AWS client that returns no deployments
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
			// No deployments
			return &appconfig.ListDeploymentsOutput{
				Items: []types.DeploymentSummary{},
			}, nil
		},
	}

	reporter := &reportertest.MockReporter{}
	executor := NewExecutorWithFactory(reporter, func(ctx context.Context, region string) (*awsInternal.Client, error) {
		return &awsInternal.Client{
			AppConfig: mockClient,
		}, nil
	})

	opts := &Options{
		ConfigFile: configPath,
	}

	err = executor.Execute(context.Background(), opts)
	if err != nil {
		t.Errorf("expected no error for no deployments case, got: %v", err)
	}

	// Check that warning was reported
	foundWarning := false
	for _, msg := range reporter.Messages {
		if strings.Contains(msg, "warning") && strings.Contains(msg, "No deployments found") {
			foundWarning = true
			break
		}
	}
	if !foundWarning {
		t.Error("expected warning message about no deployments")
	}
}

func TestExecutorWithDeployment(t *testing.T) {
	// Create temporary test files
	tempDir, err := os.MkdirTemp("", "executor-deploy-*")
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

	now := time.Now()
	deploymentNumber := int32(1)

	// Create mock AWS client with a deployment
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
				Items: []types.DeploymentSummary{
					{
						DeploymentNumber: deploymentNumber,
						State:            types.DeploymentStateComplete,
					},
				},
			}, nil
		},
		GetDeploymentFunc: func(ctx context.Context, params *appconfig.GetDeploymentInput, optFns ...func(*appconfig.Options)) (*appconfig.GetDeploymentOutput, error) {
			return &appconfig.GetDeploymentOutput{
				DeploymentNumber:       deploymentNumber,
				ConfigurationProfileId: aws.String("profile-123"),
				ConfigurationVersion:   aws.String("1"),
				DeploymentStrategyId:   aws.String("strategy-123"),
				State:                  types.DeploymentStateComplete,
				Description:            aws.String("Test deployment"),
				StartedAt:              &now,
				CompletedAt:            &now,
				PercentageComplete:     aws.Float32(100),
				GrowthFactor:           aws.Float32(100),
				FinalBakeTimeInMinutes: 0,
			}, nil
		},
	}

	reporter := &reportertest.MockReporter{}
	executor := NewExecutorWithFactory(reporter, func(ctx context.Context, region string) (*awsInternal.Client, error) {
		return &awsInternal.Client{
			AppConfig: mockClient,
		}, nil
	})

	opts := &Options{
		ConfigFile: configPath,
	}

	err = executor.Execute(context.Background(), opts)
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestGetDeploymentByIDInvalidID(t *testing.T) {
	reporter := &reportertest.MockReporter{}
	executor := NewExecutor(reporter)

	_, err := executor.getDeploymentByID(
		context.Background(),
		&awsInternal.Client{},
		&awsInternal.ResolvedResources{},
		"invalid",
	)

	if err == nil {
		t.Error("expected error for invalid deployment ID")
	}

	if !strings.Contains(err.Error(), "invalid deployment ID") {
		t.Errorf("expected 'invalid deployment ID' error, got: %v", err)
	}
}

func TestGetDeploymentByIDWrongProfile(t *testing.T) {
	deploymentNumber := int32(1)
	now := time.Now()

	mockClient := &mock.MockAppConfigClient{
		GetDeploymentFunc: func(ctx context.Context, params *appconfig.GetDeploymentInput, optFns ...func(*appconfig.Options)) (*appconfig.GetDeploymentOutput, error) {
			return &appconfig.GetDeploymentOutput{
				DeploymentNumber:       deploymentNumber,
				ConfigurationProfileId: aws.String("wrong-profile-123"),
				ConfigurationVersion:   aws.String("1"),
				DeploymentStrategyId:   aws.String("strategy-123"),
				State:                  types.DeploymentStateComplete,
				StartedAt:              &now,
				CompletedAt:            &now,
				PercentageComplete:     aws.Float32(100),
				GrowthFactor:           aws.Float32(100),
			}, nil
		},
	}

	reporter := &reportertest.MockReporter{}
	executor := NewExecutorWithFactory(reporter, func(ctx context.Context, region string) (*awsInternal.Client, error) {
		return &awsInternal.Client{
			AppConfig: mockClient,
		}, nil
	})

	resources := &awsInternal.ResolvedResources{
		ApplicationID: "app-123",
		EnvironmentID: "env-123",
		Profile: &awsInternal.ProfileInfo{
			ID:   "profile-123",
			Name: "test-profile",
		},
	}

	_, err := executor.getDeploymentByID(
		context.Background(),
		&awsInternal.Client{AppConfig: mockClient},
		resources,
		"1",
	)

	if err == nil {
		t.Error("expected error for wrong profile")
	}

	if !strings.Contains(err.Error(), "not for configuration profile") {
		t.Errorf("expected 'not for configuration profile' error, got: %v", err)
	}
}

func TestGetDeploymentByIDSuccess(t *testing.T) {
	deploymentNumber := int32(1)
	now := time.Now()

	mockClient := &mock.MockAppConfigClient{
		GetDeploymentFunc: func(ctx context.Context, params *appconfig.GetDeploymentInput, optFns ...func(*appconfig.Options)) (*appconfig.GetDeploymentOutput, error) {
			return &appconfig.GetDeploymentOutput{
				DeploymentNumber:       deploymentNumber,
				ConfigurationProfileId: aws.String("profile-123"),
				ConfigurationVersion:   aws.String("1"),
				DeploymentStrategyId:   aws.String("strategy-123"),
				State:                  types.DeploymentStateComplete,
				StartedAt:              &now,
				CompletedAt:            &now,
				PercentageComplete:     aws.Float32(100),
				GrowthFactor:           aws.Float32(100),
			}, nil
		},
		ListDeploymentStrategiesFunc: func(ctx context.Context, params *appconfig.ListDeploymentStrategiesInput, optFns ...func(*appconfig.Options)) (*appconfig.ListDeploymentStrategiesOutput, error) {
			return &appconfig.ListDeploymentStrategiesOutput{
				Items: []types.DeploymentStrategy{
					{
						Id:   aws.String("strategy-123"),
						Name: aws.String("TestStrategy"),
					},
				},
			}, nil
		},
	}

	reporter := &reportertest.MockReporter{}
	executor := NewExecutorWithFactory(reporter, func(ctx context.Context, region string) (*awsInternal.Client, error) {
		return &awsInternal.Client{
			AppConfig: mockClient,
		}, nil
	})

	resources := &awsInternal.ResolvedResources{
		ApplicationID: "app-123",
		EnvironmentID: "env-123",
		Profile: &awsInternal.ProfileInfo{
			ID:   "profile-123",
			Name: "test-profile",
		},
	}

	deployment, err := executor.getDeploymentByID(
		context.Background(),
		&awsInternal.Client{AppConfig: mockClient},
		resources,
		"1",
	)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if deployment == nil {
		t.Fatal("expected deployment to be non-nil")
	}

	if deployment.DeploymentNumber != deploymentNumber {
		t.Errorf("expected deployment number %d, got %d", deploymentNumber, deployment.DeploymentNumber)
	}
}

func TestGetLatestDeploymentSuccess(t *testing.T) {
	deploymentNumber := int32(5)
	now := time.Now()

	mockClient := &mock.MockAppConfigClient{
		ListDeploymentsFunc: func(ctx context.Context, params *appconfig.ListDeploymentsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListDeploymentsOutput, error) {
			return &appconfig.ListDeploymentsOutput{
				Items: []types.DeploymentSummary{
					{
						DeploymentNumber: deploymentNumber,
						State:            types.DeploymentStateComplete,
					},
					{
						DeploymentNumber: 4,
						State:            types.DeploymentStateComplete,
					},
				},
			}, nil
		},
		GetDeploymentFunc: func(ctx context.Context, params *appconfig.GetDeploymentInput, optFns ...func(*appconfig.Options)) (*appconfig.GetDeploymentOutput, error) {
			return &appconfig.GetDeploymentOutput{
				DeploymentNumber:       deploymentNumber,
				ConfigurationProfileId: aws.String("profile-123"),
				ConfigurationVersion:   aws.String("5"),
				DeploymentStrategyId:   aws.String("strategy-123"),
				State:                  types.DeploymentStateComplete,
				StartedAt:              &now,
				CompletedAt:            &now,
				PercentageComplete:     aws.Float32(100),
				GrowthFactor:           aws.Float32(100),
			}, nil
		},
		ListDeploymentStrategiesFunc: func(ctx context.Context, params *appconfig.ListDeploymentStrategiesInput, optFns ...func(*appconfig.Options)) (*appconfig.ListDeploymentStrategiesOutput, error) {
			return &appconfig.ListDeploymentStrategiesOutput{
				Items: []types.DeploymentStrategy{
					{
						Id:   aws.String("strategy-123"),
						Name: aws.String("TestStrategy"),
					},
				},
			}, nil
		},
	}

	reporter := &reportertest.MockReporter{}
	executor := NewExecutorWithFactory(reporter, func(ctx context.Context, region string) (*awsInternal.Client, error) {
		return &awsInternal.Client{
			AppConfig: mockClient,
		}, nil
	})

	resources := &awsInternal.ResolvedResources{
		ApplicationID: "app-123",
		EnvironmentID: "env-123",
		Profile: &awsInternal.ProfileInfo{
			ID:   "profile-123",
			Name: "test-profile",
		},
	}

	deployment, err := executor.getLatestDeployment(
		context.Background(),
		&awsInternal.Client{AppConfig: mockClient},
		resources,
	)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if deployment == nil {
		t.Fatal("expected deployment to be non-nil")
	}

	if deployment.DeploymentNumber != deploymentNumber {
		t.Errorf("expected deployment number %d, got %d", deploymentNumber, deployment.DeploymentNumber)
	}
}

func TestExecutorWithDeploymentID(t *testing.T) {
	// Create temporary test files
	tempDir, err := os.MkdirTemp("", "executor-deployid-*")
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

	now := time.Now()
	deploymentNumber := int32(3)

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
		GetDeploymentFunc: func(ctx context.Context, params *appconfig.GetDeploymentInput, optFns ...func(*appconfig.Options)) (*appconfig.GetDeploymentOutput, error) {
			return &appconfig.GetDeploymentOutput{
				DeploymentNumber:       deploymentNumber,
				ConfigurationProfileId: aws.String("profile-123"),
				ConfigurationVersion:   aws.String("3"),
				DeploymentStrategyId:   aws.String("strategy-123"),
				State:                  types.DeploymentStateComplete,
				Description:            aws.String("Test deployment #3"),
				StartedAt:              &now,
				CompletedAt:            &now,
				PercentageComplete:     aws.Float32(100),
				GrowthFactor:           aws.Float32(100),
			}, nil
		},
	}

	reporter := &reportertest.MockReporter{}
	executor := NewExecutorWithFactory(reporter, func(ctx context.Context, region string) (*awsInternal.Client, error) {
		return &awsInternal.Client{
			AppConfig: mockClient,
		}, nil
	})

	opts := &Options{
		ConfigFile:   configPath,
		DeploymentID: "3",
	}

	err = executor.Execute(context.Background(), opts)
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}

	// Verify that the correct deployment ID was fetched
	foundProgress := false
	for _, msg := range reporter.Messages {
		if strings.Contains(msg, "Fetching deployment #3") {
			foundProgress = true
			break
		}
	}
	if !foundProgress {
		t.Error("expected progress message for deployment #3")
	}
}

func TestExecutorAWSClientError(t *testing.T) {
	// Create temporary test files
	tempDir, err := os.MkdirTemp("", "executor-aws-error-*")
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

	reporter := &reportertest.MockReporter{}
	executor := NewExecutorWithFactory(reporter, func(ctx context.Context, region string) (*awsInternal.Client, error) {
		return nil, fmt.Errorf("AWS client initialization failed")
	})

	opts := &Options{
		ConfigFile: configPath,
	}

	err = executor.Execute(context.Background(), opts)
	if err == nil {
		t.Error("expected error when AWS client fails to initialize")
	}

	if !strings.Contains(err.Error(), "failed to initialize AWS client") {
		t.Errorf("expected 'failed to initialize AWS client' error, got: %v", err)
	}
}

func TestExecutorResolveResourcesError(t *testing.T) {
	// Create temporary test files
	tempDir, err := os.MkdirTemp("", "executor-resolve-error-*")
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

	// Create mock AWS client that fails to list applications
	mockClient := &mock.MockAppConfigClient{
		ListApplicationsFunc: func(ctx context.Context, params *appconfig.ListApplicationsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListApplicationsOutput, error) {
			return nil, fmt.Errorf("failed to list applications")
		},
	}

	reporter := &reportertest.MockReporter{}
	executor := NewExecutorWithFactory(reporter, func(ctx context.Context, region string) (*awsInternal.Client, error) {
		return &awsInternal.Client{
			AppConfig: mockClient,
		}, nil
	})

	opts := &Options{
		ConfigFile: configPath,
	}

	err = executor.Execute(context.Background(), opts)
	if err == nil {
		t.Error("expected error when resolving resources fails")
	}

	if !strings.Contains(err.Error(), "failed to resolve resources") {
		t.Errorf("expected 'failed to resolve resources' error, got: %v", err)
	}
}

func TestGetDeploymentByIDGetDetailsError(t *testing.T) {
	mockClient := &mock.MockAppConfigClient{
		GetDeploymentFunc: func(ctx context.Context, params *appconfig.GetDeploymentInput, optFns ...func(*appconfig.Options)) (*appconfig.GetDeploymentOutput, error) {
			return nil, fmt.Errorf("failed to get deployment")
		},
	}

	reporter := &reportertest.MockReporter{}
	executor := NewExecutorWithFactory(reporter, func(ctx context.Context, region string) (*awsInternal.Client, error) {
		return &awsInternal.Client{
			AppConfig: mockClient,
		}, nil
	})

	resources := &awsInternal.ResolvedResources{
		ApplicationID: "app-123",
		EnvironmentID: "env-123",
		Profile: &awsInternal.ProfileInfo{
			ID:   "profile-123",
			Name: "test-profile",
		},
	}

	_, err := executor.getDeploymentByID(
		context.Background(),
		&awsInternal.Client{AppConfig: mockClient},
		resources,
		"1",
	)

	if err == nil {
		t.Error("expected error when getting deployment details fails")
	}
}

func TestGetLatestDeploymentListError(t *testing.T) {
	mockClient := &mock.MockAppConfigClient{
		ListDeploymentsFunc: func(ctx context.Context, params *appconfig.ListDeploymentsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListDeploymentsOutput, error) {
			return nil, fmt.Errorf("failed to list deployments")
		},
	}

	reporter := &reportertest.MockReporter{}
	executor := NewExecutorWithFactory(reporter, func(ctx context.Context, region string) (*awsInternal.Client, error) {
		return &awsInternal.Client{
			AppConfig: mockClient,
		}, nil
	})

	resources := &awsInternal.ResolvedResources{
		ApplicationID: "app-123",
		EnvironmentID: "env-123",
		Profile: &awsInternal.ProfileInfo{
			ID:   "profile-123",
			Name: "test-profile",
		},
	}

	_, err := executor.getLatestDeployment(
		context.Background(),
		&awsInternal.Client{AppConfig: mockClient},
		resources,
	)

	if err == nil {
		t.Error("expected error when listing deployments fails")
	}
}

func TestGetLatestDeploymentNoMatchingProfile(t *testing.T) {
	deploymentNumber := int32(1)
	now := time.Now()

	mockClient := &mock.MockAppConfigClient{
		ListDeploymentsFunc: func(ctx context.Context, params *appconfig.ListDeploymentsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListDeploymentsOutput, error) {
			return &appconfig.ListDeploymentsOutput{
				Items: []types.DeploymentSummary{
					{
						DeploymentNumber: deploymentNumber,
						State:            types.DeploymentStateComplete,
					},
				},
			}, nil
		},
		GetDeploymentFunc: func(ctx context.Context, params *appconfig.GetDeploymentInput, optFns ...func(*appconfig.Options)) (*appconfig.GetDeploymentOutput, error) {
			// Return deployment for a different profile
			return &appconfig.GetDeploymentOutput{
				DeploymentNumber:       deploymentNumber,
				ConfigurationProfileId: aws.String("wrong-profile-123"),
				ConfigurationVersion:   aws.String("1"),
				DeploymentStrategyId:   aws.String("strategy-123"),
				State:                  types.DeploymentStateComplete,
				StartedAt:              &now,
				CompletedAt:            &now,
				PercentageComplete:     aws.Float32(100),
				GrowthFactor:           aws.Float32(100),
			}, nil
		},
	}

	reporter := &reportertest.MockReporter{}
	executor := NewExecutorWithFactory(reporter, func(ctx context.Context, region string) (*awsInternal.Client, error) {
		return &awsInternal.Client{
			AppConfig: mockClient,
		}, nil
	})

	resources := &awsInternal.ResolvedResources{
		ApplicationID: "app-123",
		EnvironmentID: "env-123",
		Profile: &awsInternal.ProfileInfo{
			ID:   "profile-123",
			Name: "test-profile",
		},
	}

	deployment, err := executor.getLatestDeployment(
		context.Background(),
		&awsInternal.Client{AppConfig: mockClient},
		resources,
	)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Should return nil when no matching profile is found
	if deployment != nil {
		t.Error("expected nil deployment when no matching profile found")
	}
}

func TestExecutorRegionOverride(t *testing.T) {
	// Create temporary test files
	tempDir, err := os.MkdirTemp("", "executor-region-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create config file with different region
	configPath := filepath.Join(tempDir, "apcdeploy.yml")
	configContent := `application: test-app
configuration_profile: test-profile
environment: test-env
deployment_strategy: AppConfig.AllAtOnce
data_file: data.json
region: us-west-2
`
	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	// Create data file
	dataPath := filepath.Join(tempDir, "data.json")
	if err := os.WriteFile(dataPath, []byte(`{"key": "value"}`), 0o644); err != nil {
		t.Fatalf("Failed to write data: %v", err)
	}

	now := time.Now()
	deploymentNumber := int32(1)
	usedRegion := ""

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
				Items: []types.DeploymentSummary{
					{
						DeploymentNumber: deploymentNumber,
						State:            types.DeploymentStateComplete,
					},
				},
			}, nil
		},
		GetDeploymentFunc: func(ctx context.Context, params *appconfig.GetDeploymentInput, optFns ...func(*appconfig.Options)) (*appconfig.GetDeploymentOutput, error) {
			return &appconfig.GetDeploymentOutput{
				DeploymentNumber:       deploymentNumber,
				ConfigurationProfileId: aws.String("profile-123"),
				ConfigurationVersion:   aws.String("1"),
				DeploymentStrategyId:   aws.String("strategy-123"),
				State:                  types.DeploymentStateComplete,
				StartedAt:              &now,
				CompletedAt:            &now,
				PercentageComplete:     aws.Float32(100),
				GrowthFactor:           aws.Float32(100),
			}, nil
		},
	}

	reporter := &reportertest.MockReporter{}
	executor := NewExecutorWithFactory(reporter, func(ctx context.Context, region string) (*awsInternal.Client, error) {
		usedRegion = region
		return &awsInternal.Client{
			AppConfig: mockClient,
		}, nil
	})

	opts := &Options{
		ConfigFile: configPath,
		Region:     "ap-northeast-1", // Override region
	}

	err = executor.Execute(context.Background(), opts)
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}

	if usedRegion != "ap-northeast-1" {
		t.Errorf("expected region to be overridden to ap-northeast-1, got %s", usedRegion)
	}
}
