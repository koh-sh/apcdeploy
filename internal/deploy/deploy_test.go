package deploy

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/appconfig"
	"github.com/aws/aws-sdk-go-v2/service/appconfig/types"
	awsInternal "github.com/koh-sh/apcdeploy/internal/aws"
	"github.com/koh-sh/apcdeploy/internal/aws/mock"
	"github.com/koh-sh/apcdeploy/internal/config"
)

func TestLoadConfiguration(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "deploy-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a valid config file
	validConfigPath := filepath.Join(tempDir, "apcdeploy.yml")
	validConfigContent := `application: test-app
configuration_profile: test-profile
environment: test-env
deployment_strategy: AppConfig.AllAtOnce
data_file: data.json
region: us-east-1
`
	if err := os.WriteFile(validConfigPath, []byte(validConfigContent), 0o644); err != nil {
		t.Fatalf("Failed to write valid config: %v", err)
	}

	// Create a valid data file
	dataPath := filepath.Join(tempDir, "data.json")
	dataContent := `{"key": "value"}`
	if err := os.WriteFile(dataPath, []byte(dataContent), 0o644); err != nil {
		t.Fatalf("Failed to write data file: %v", err)
	}

	tests := []struct {
		name       string
		configPath string
		wantErr    bool
	}{
		{
			name:       "valid config file",
			configPath: validConfigPath,
			wantErr:    false,
		},
		{
			name:       "non-existent config file",
			configPath: filepath.Join(tempDir, "nonexistent.yml"),
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, dataContent, err := loadConfiguration(tt.configPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("loadConfiguration() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if cfg == nil {
					t.Error("Expected config to be non-nil")
				}
				if dataContent == nil {
					t.Error("Expected dataContent to be non-nil")
				}
			}
		})
	}
}

func TestLoadConfigurationDataPath(t *testing.T) {
	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "deploy-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a config file with relative data path
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
	expectedContent := `{"key": "value"}`
	if err := os.WriteFile(dataPath, []byte(expectedContent), 0o644); err != nil {
		t.Fatalf("Failed to write data file: %v", err)
	}

	cfg, dataContent, err := loadConfiguration(configPath)
	if err != nil {
		t.Fatalf("loadConfiguration() error = %v", err)
	}

	if string(dataContent) != expectedContent {
		t.Errorf("Data content = %v, want %v", string(dataContent), expectedContent)
	}

	// Check that the data path is resolved to absolute path
	if !filepath.IsAbs(cfg.DataFile) {
		t.Errorf("Expected absolute path, got: %v", cfg.DataFile)
	}
}

func TestDeployer_ValidateLocalData(t *testing.T) {
	tests := []struct {
		name        string
		data        []byte
		contentType string
		wantErr     bool
	}{
		{
			name:        "valid JSON",
			data:        []byte(`{"key": "value"}`),
			contentType: "application/json",
			wantErr:     false,
		},
		{
			name:        "invalid JSON",
			data:        []byte(`{invalid json}`),
			contentType: "application/json",
			wantErr:     true,
		},
		{
			name:        "valid YAML",
			data:        []byte("key: value\n"),
			contentType: "application/x-yaml",
			wantErr:     false,
		},
		{
			name:        "invalid YAML",
			data:        []byte(":\n  invalid yaml\n:"),
			contentType: "application/x-yaml",
			wantErr:     true,
		},
		{
			name:        "text content always valid",
			data:        []byte("any text content"),
			contentType: "text/plain",
			wantErr:     false,
		},
		{
			name:        "data too large",
			data:        make([]byte, 2*1024*1024+1), // 2MB + 1 byte
			contentType: "application/json",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &Deployer{}
			err := d.ValidateLocalData(tt.data, tt.contentType)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateLocalData() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDeployer_DetermineContentType(t *testing.T) {
	tests := []struct {
		name        string
		profileType string
		dataPath    string
		want        string
		wantErr     bool
	}{
		{
			name:        "Feature Flags always JSON",
			profileType: "AWS.AppConfig.FeatureFlags",
			dataPath:    "flags.json",
			want:        "application/json",
			wantErr:     false,
		},
		{
			name:        "Freeform JSON file",
			profileType: "AWS.Freeform",
			dataPath:    "config.json",
			want:        "application/json",
			wantErr:     false,
		},
		{
			name:        "Freeform YAML file",
			profileType: "AWS.Freeform",
			dataPath:    "config.yaml",
			want:        "application/x-yaml",
			wantErr:     false,
		},
		{
			name:        "Freeform YML file",
			profileType: "AWS.Freeform",
			dataPath:    "config.yml",
			want:        "application/x-yaml",
			wantErr:     false,
		},
		{
			name:        "Freeform text file",
			profileType: "AWS.Freeform",
			dataPath:    "config.txt",
			want:        "text/plain",
			wantErr:     false,
		},
		{
			name:        "Freeform unknown extension defaults to text",
			profileType: "AWS.Freeform",
			dataPath:    "config.conf",
			want:        "text/plain",
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &Deployer{}
			got, err := d.DetermineContentType(tt.profileType, tt.dataPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("DetermineContentType() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("DetermineContentType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNew(t *testing.T) {
	ctx := context.Background()
	cfg := &config.Config{
		Application:          "test-app",
		ConfigurationProfile: "test-profile",
		Environment:          "test-env",
		DeploymentStrategy:   "AppConfig.AllAtOnce",
		Region:               "us-east-1",
		DataFile:             "data.json",
	}

	d, err := New(ctx, cfg)
	if err != nil {
		t.Errorf("New() error = %v", err)
	}
	if d == nil {
		t.Error("Expected deployer to be non-nil")
	}
}

func TestDeployer_ResolveResources(t *testing.T) {
	// This is a placeholder test - actual resource resolution will use AWS mocks
	// For now, we just verify the structure exists
	ctx := context.Background()
	cfg := &config.Config{
		Application:          "test-app",
		ConfigurationProfile: "test-profile",
		Environment:          "test-env",
		DeploymentStrategy:   "AppConfig.AllAtOnce",
		Region:               "us-east-1",
		DataFile:             "data.json",
	}

	d, err := New(ctx, cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// We can't test actual AWS resolution without mocks
	// This test just verifies the deployer has the AWS client
	if d.awsClient == nil {
		t.Error("Expected awsClient to be non-nil")
	}
}

func TestNewWithClient(t *testing.T) {
	cfg := &config.Config{
		Application:          "test-app",
		ConfigurationProfile: "test-profile",
		Environment:          "test-env",
		DeploymentStrategy:   "AppConfig.AllAtOnce",
		DataFile:             "data.json",
		Region:               "us-east-1",
	}

	mockClient := &mock.MockAppConfigClient{}
	awsClient := &awsInternal.Client{
		AppConfig: mockClient,
	}

	deployer := NewWithClient(cfg, awsClient)

	if deployer == nil {
		t.Fatal("expected deployer to be non-nil")
	}

	if deployer.cfg != cfg {
		t.Error("expected deployer to have the provided config")
	}

	if deployer.awsClient != awsClient {
		t.Error("expected deployer to have the provided AWS client")
	}
}

func TestResolveResourcesWithMock(t *testing.T) {
	cfg := &config.Config{
		Application:          "test-app",
		ConfigurationProfile: "test-profile",
		Environment:          "test-env",
		DeploymentStrategy:   "AppConfig.AllAtOnce",
		DataFile:             "data.json",
		Region:               "us-east-1",
	}

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
	}

	awsClient := &awsInternal.Client{
		AppConfig: mockClient,
	}

	deployer := NewWithClient(cfg, awsClient)
	resolved, err := deployer.ResolveResources(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resolved.ApplicationID != "app-123" {
		t.Errorf("expected ApplicationID 'app-123', got '%s'", resolved.ApplicationID)
	}

	if resolved.Profile.ID != "profile-123" {
		t.Errorf("expected ProfileID 'profile-123', got '%s'", resolved.Profile.ID)
	}

	if resolved.EnvironmentID != "env-123" {
		t.Errorf("expected EnvironmentID 'env-123', got '%s'", resolved.EnvironmentID)
	}

	if resolved.DeploymentStrategyID != "strategy-123" {
		t.Errorf("expected DeploymentStrategyID 'strategy-123', got '%s'", resolved.DeploymentStrategyID)
	}
}

func TestCheckOngoingDeploymentWithMock(t *testing.T) {
	cfg := &config.Config{
		Application:          "test-app",
		ConfigurationProfile: "test-profile",
		Environment:          "test-env",
		DeploymentStrategy:   "AppConfig.AllAtOnce",
		DataFile:             "data.json",
		Region:               "us-east-1",
	}

	tests := []struct {
		name        string
		deployments []types.DeploymentSummary
		wantOngoing bool
	}{
		{
			name:        "no ongoing deployments",
			deployments: []types.DeploymentSummary{},
			wantOngoing: false,
		},
		{
			name: "has deploying deployment",
			deployments: []types.DeploymentSummary{
				{
					DeploymentNumber: 1,
					State:            types.DeploymentStateDeploying,
				},
			},
			wantOngoing: true,
		},
		{
			name: "has completed deployment",
			deployments: []types.DeploymentSummary{
				{
					DeploymentNumber: 1,
					State:            types.DeploymentStateComplete,
				},
			},
			wantOngoing: false,
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

			awsClient := &awsInternal.Client{
				AppConfig:       mockClient,
				PollingInterval: 100 * time.Millisecond, // Fast polling for tests
			}

			deployer := NewWithClient(cfg, awsClient)

			resolved := &awsInternal.ResolvedResources{
				ApplicationID:        "app-123",
				EnvironmentID:        "env-123",
				DeploymentStrategyID: "strategy-123",
				Profile: &awsInternal.ProfileInfo{
					ID:   "profile-123",
					Type: "AWS.Freeform",
				},
			}

			hasOngoing, _, err := deployer.CheckOngoingDeployment(context.Background(), resolved)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if hasOngoing != tt.wantOngoing {
				t.Errorf("expected hasOngoing=%v, got %v", tt.wantOngoing, hasOngoing)
			}
		})
	}
}

func TestCreateVersionWithMock(t *testing.T) {
	cfg := &config.Config{
		Application:          "test-app",
		ConfigurationProfile: "test-profile",
		Environment:          "test-env",
		DeploymentStrategy:   "AppConfig.AllAtOnce",
		DataFile:             "data.json",
		Region:               "us-east-1",
	}

	mockClient := &mock.MockAppConfigClient{
		CreateHostedConfigurationVersionFunc: func(ctx context.Context, params *appconfig.CreateHostedConfigurationVersionInput, optFns ...func(*appconfig.Options)) (*appconfig.CreateHostedConfigurationVersionOutput, error) {
			return &appconfig.CreateHostedConfigurationVersionOutput{
				VersionNumber: 5,
			}, nil
		},
	}

	awsClient := &awsInternal.Client{
		AppConfig: mockClient,
	}

	deployer := NewWithClient(cfg, awsClient)

	resolved := &awsInternal.ResolvedResources{
		ApplicationID:        "app-123",
		EnvironmentID:        "env-123",
		DeploymentStrategyID: "strategy-123",
		Profile: &awsInternal.ProfileInfo{
			ID:   "profile-123",
			Type: "AWS.Freeform",
		},
	}

	versionNumber, err := deployer.CreateVersion(context.Background(), resolved, []byte(`{"key":"value"}`), "application/json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if versionNumber != 5 {
		t.Errorf("expected version number 5, got %d", versionNumber)
	}
}

func TestStartDeploymentWithMock(t *testing.T) {
	cfg := &config.Config{
		Application:          "test-app",
		ConfigurationProfile: "test-profile",
		Environment:          "test-env",
		DeploymentStrategy:   "AppConfig.AllAtOnce",
		DataFile:             "data.json",
		Region:               "us-east-1",
	}

	mockClient := &mock.MockAppConfigClient{
		StartDeploymentFunc: func(ctx context.Context, params *appconfig.StartDeploymentInput, optFns ...func(*appconfig.Options)) (*appconfig.StartDeploymentOutput, error) {
			return &appconfig.StartDeploymentOutput{
				DeploymentNumber: 3,
			}, nil
		},
	}

	awsClient := &awsInternal.Client{
		AppConfig: mockClient,
	}

	deployer := NewWithClient(cfg, awsClient)

	resolved := &awsInternal.ResolvedResources{
		ApplicationID:        "app-123",
		EnvironmentID:        "env-123",
		DeploymentStrategyID: "strategy-123",
		Profile: &awsInternal.ProfileInfo{
			ID:   "profile-123",
			Type: "AWS.Freeform",
		},
	}

	deploymentNumber, err := deployer.StartDeployment(context.Background(), resolved, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if deploymentNumber != 3 {
		t.Errorf("expected deployment number 3, got %d", deploymentNumber)
	}
}

func TestWaitForDeploymentWithMock(t *testing.T) {
	tests := []struct {
		name        string
		mockStates  []types.DeploymentState
		timeout     int
		wantErr     bool
		description string
	}{
		{
			name:        "immediate completion",
			mockStates:  []types.DeploymentState{types.DeploymentStateComplete},
			timeout:     30,
			wantErr:     false,
			description: "Tests immediate deployment completion",
		},
		{
			name:        "completion after polling (5s wait)",
			mockStates:  []types.DeploymentState{types.DeploymentStateDeploying, types.DeploymentStateComplete},
			timeout:     30,
			wantErr:     false,
			description: "Tests deployment that completes after one poll - this takes 5 seconds",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Application:          "test-app",
				ConfigurationProfile: "test-profile",
				Environment:          "test-env",
				DeploymentStrategy:   "AppConfig.AllAtOnce",
				DataFile:             "data.json",
				Region:               "us-east-1",
			}

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
						State: state,
					}, nil
				},
			}

			awsClient := &awsInternal.Client{
				AppConfig:       mockClient,
				PollingInterval: 100 * time.Millisecond, // Fast polling for tests
			}

			deployer := NewWithClient(cfg, awsClient)

			resolved := &awsInternal.ResolvedResources{
				ApplicationID:        "app-123",
				EnvironmentID:        "env-123",
				DeploymentStrategyID: "strategy-123",
				Profile: &awsInternal.ProfileInfo{
					ID:   "profile-123",
					Type: "AWS.Freeform",
				},
			}

			err := deployer.WaitForDeployment(context.Background(), resolved, 1, tt.timeout)
			if (err != nil) != tt.wantErr {
				t.Fatalf("unexpected error: %v, wantErr: %v", err, tt.wantErr)
			}
		})
	}
}

func TestIsValidationError(t *testing.T) {
	cfg := &config.Config{
		Application:          "test-app",
		ConfigurationProfile: "test-profile",
		Environment:          "test-env",
		DeploymentStrategy:   "AppConfig.AllAtOnce",
		DataFile:             "data.json",
		Region:               "us-east-1",
	}

	mockClient := &mock.MockAppConfigClient{}
	awsClient := &awsInternal.Client{AppConfig: mockClient}
	deployer := NewWithClient(cfg, awsClient)

	tests := []struct {
		name    string
		err     error
		wantErr bool
	}{
		{
			name:    "nil error",
			err:     nil,
			wantErr: false,
		},
		{
			name:    "non-validation error",
			err:     errors.New("generic error"),
			wantErr: false,
		},
		{
			name:    "BadRequestException",
			err:     &types.BadRequestException{Message: aws.String("validation failed")},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := deployer.IsValidationError(tt.err)
			if result != tt.wantErr {
				t.Errorf("IsValidationError() = %v, want %v", result, tt.wantErr)
			}
		})
	}
}

func TestFormatValidationError(t *testing.T) {
	cfg := &config.Config{
		Application:          "test-app",
		ConfigurationProfile: "test-profile",
		Environment:          "test-env",
		DeploymentStrategy:   "AppConfig.AllAtOnce",
		DataFile:             "data.json",
		Region:               "us-east-1",
	}

	mockClient := &mock.MockAppConfigClient{}
	awsClient := &awsInternal.Client{AppConfig: mockClient}
	deployer := NewWithClient(cfg, awsClient)

	tests := []struct {
		name         string
		err          error
		wantContains []string
	}{
		{
			name: "BadRequestException with message",
			err:  &types.BadRequestException{Message: aws.String("JSON schema validation failed")},
			wantContains: []string{
				"Configuration validation failed",
				"JSON schema validation failed",
			},
		},
		{
			name: "generic error",
			err:  errors.New("some error"),
			wantContains: []string{
				"Configuration validation failed",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := deployer.FormatValidationError(tt.err)
			for _, want := range tt.wantContains {
				if !contains(result, want) {
					t.Errorf("FormatValidationError() result does not contain %q\nGot: %s", want, result)
				}
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && (s[:len(substr)] == substr || contains(s[1:], substr))))
}
