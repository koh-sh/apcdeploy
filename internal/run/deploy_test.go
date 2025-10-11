package run

import (
	"context"
	"encoding/json"
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

	// Create a config file with missing data file
	missingDataConfigPath := filepath.Join(tempDir, "missing-data.yml")
	missingDataContent := `application: test-app
configuration_profile: test-profile
environment: test-env
deployment_strategy: AppConfig.AllAtOnce
data_file: nonexistent.json
region: us-east-1
`
	if err := os.WriteFile(missingDataConfigPath, []byte(missingDataContent), 0o644); err != nil {
		t.Fatalf("Failed to write config with missing data: %v", err)
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
		{
			name:       "config with missing data file",
			configPath: missingDataConfigPath,
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
		{
			name:        "unsupported content type",
			data:        []byte("some data"),
			contentType: "application/xml",
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
			profileType: config.ProfileTypeFeatureFlags,
			dataPath:    "flags.json",
			want:        "application/json",
			wantErr:     false,
		},
		{
			name:        "Freeform JSON file",
			profileType: config.ProfileTypeFreeform,
			dataPath:    "config.json",
			want:        "application/json",
			wantErr:     false,
		},
		{
			name:        "Freeform YAML file",
			profileType: config.ProfileTypeFreeform,
			dataPath:    "config.yaml",
			want:        "application/x-yaml",
			wantErr:     false,
		},
		{
			name:        "Freeform YML file",
			profileType: config.ProfileTypeFreeform,
			dataPath:    "config.yml",
			want:        "application/x-yaml",
			wantErr:     false,
		},
		{
			name:        "Freeform text file",
			profileType: config.ProfileTypeFreeform,
			dataPath:    "config.txt",
			want:        "text/plain",
			wantErr:     false,
		},
		{
			name:        "Freeform unknown extension defaults to text",
			profileType: config.ProfileTypeFreeform,
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
	tests := []struct {
		name    string
		region  string
		wantErr bool
	}{
		{
			name:    "valid region",
			region:  "us-east-1",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			cfg := &config.Config{
				Application:          "test-app",
				ConfigurationProfile: "test-profile",
				Environment:          "test-env",
				DeploymentStrategy:   "AppConfig.AllAtOnce",
				Region:               tt.region,
				DataFile:             "data.json",
			}

			d, err := New(ctx, cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && d == nil {
				t.Error("Expected deployer to be non-nil")
			}
		})
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
		return
	}

	if deployer.cfg != cfg {
		t.Error("expected deployer to have the provided config")
	}

	if deployer.awsClient != awsClient {
		t.Error("expected deployer to have the provided AWS client")
	}
}

func TestResolveResourcesWithMock(t *testing.T) {
	tests := []struct {
		name               string
		listAppsError      error
		wantErr            bool
		expectedAppID      string
		expectedProfileID  string
		expectedEnvID      string
		expectedStrategyID string
	}{
		{
			name:               "successful resolution",
			listAppsError:      nil,
			wantErr:            false,
			expectedAppID:      "app-123",
			expectedProfileID:  "profile-123",
			expectedEnvID:      "env-123",
			expectedStrategyID: "strategy-123",
		},
		{
			name:          "error listing applications",
			listAppsError: errors.New("failed to list applications"),
			wantErr:       true,
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

			mockClient := &mock.MockAppConfigClient{
				ListApplicationsFunc: func(ctx context.Context, params *appconfig.ListApplicationsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListApplicationsOutput, error) {
					if tt.listAppsError != nil {
						return nil, tt.listAppsError
					}
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
								Type: aws.String(config.ProfileTypeFreeform),
							},
						},
					}, nil
				},
				GetConfigurationProfileFunc: func(ctx context.Context, params *appconfig.GetConfigurationProfileInput, optFns ...func(*appconfig.Options)) (*appconfig.GetConfigurationProfileOutput, error) {
					return &appconfig.GetConfigurationProfileOutput{
						Id:   aws.String("profile-123"),
						Name: aws.String("test-profile"),
						Type: aws.String(config.ProfileTypeFreeform),
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
			if (err != nil) != tt.wantErr {
				t.Fatalf("ResolveResources() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr {
				return
			}

			if resolved.ApplicationID != tt.expectedAppID {
				t.Errorf("expected ApplicationID '%s', got '%s'", tt.expectedAppID, resolved.ApplicationID)
			}

			if resolved.Profile.ID != tt.expectedProfileID {
				t.Errorf("expected ProfileID '%s', got '%s'", tt.expectedProfileID, resolved.Profile.ID)
			}

			if resolved.EnvironmentID != tt.expectedEnvID {
				t.Errorf("expected EnvironmentID '%s', got '%s'", tt.expectedEnvID, resolved.EnvironmentID)
			}

			if resolved.DeploymentStrategyID != tt.expectedStrategyID {
				t.Errorf("expected DeploymentStrategyID '%s', got '%s'", tt.expectedStrategyID, resolved.DeploymentStrategyID)
			}
		})
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
					Type: config.ProfileTypeFreeform,
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
			Type: config.ProfileTypeFreeform,
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
			Type: config.ProfileTypeFreeform,
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
		invalidTime bool
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
					Type: config.ProfileTypeFreeform,
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
				if !strings.Contains(result, want) {
					t.Errorf("FormatValidationError() result does not contain %q\nGot: %s", want, result)
				}
			}
		})
	}
}

func TestRemoveTimestampFieldsRecursive(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected any
	}{
		{
			name: "map with timestamp fields",
			input: map[string]any{
				"name":       "test",
				"_updatedAt": "2023-01-01",
				"_createdAt": "2023-01-01",
				"value":      42,
			},
			expected: map[string]any{
				"name":  "test",
				"value": 42,
			},
		},
		{
			name: "nested map with timestamp fields",
			input: map[string]any{
				"outer": map[string]any{
					"inner": map[string]any{
						"_updatedAt": "2023-01-01",
						"_createdAt": "2023-01-01",
						"data":       "value",
					},
				},
			},
			expected: map[string]any{
				"outer": map[string]any{
					"inner": map[string]any{
						"data": "value",
					},
				},
			},
		},
		{
			name: "array of maps with timestamp fields",
			input: []any{
				map[string]any{
					"_updatedAt": "2023-01-01",
					"name":       "first",
				},
				map[string]any{
					"_createdAt": "2023-01-01",
					"name":       "second",
				},
			},
			expected: []any{
				map[string]any{
					"name": "first",
				},
				map[string]any{
					"name": "second",
				},
			},
		},
		{
			name:     "primitive string",
			input:    "test",
			expected: "test",
		},
		{
			name:     "primitive number",
			input:    42,
			expected: 42,
		},
		{
			name:     "primitive bool",
			input:    true,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := config.RemoveTimestampFieldsRecursive(tt.input)
			resultJSON, err := json.Marshal(result)
			if err != nil {
				t.Fatalf("failed to marshal result: %v", err)
			}
			expectedJSON, err := json.Marshal(tt.expected)
			if err != nil {
				t.Fatalf("failed to marshal expected: %v", err)
			}
			if string(resultJSON) != string(expectedJSON) {
				t.Errorf("RemoveTimestampFieldsRecursive() = %s, want %s", string(resultJSON), string(expectedJSON))
			}
		})
	}
}

func TestNormalizeJSON(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		profileType string
		wantErr     bool
	}{
		{
			name:        "valid JSON - Freeform",
			content:     `{"key":"value","nested":{"a":1}}`,
			profileType: config.ProfileTypeFreeform,
			wantErr:     false,
		},
		{
			name:        "valid JSON - FeatureFlags",
			content:     `{"flags":{"flag1":{"enabled":true,"_updatedAt":"2023-01-01"}}}`,
			profileType: config.ProfileTypeFeatureFlags,
			wantErr:     false,
		},
		{
			name:        "invalid JSON - syntax error",
			content:     `{invalid json}`,
			profileType: config.ProfileTypeFreeform,
			wantErr:     true,
		},
		{
			name:        "invalid JSON - malformed",
			content:     `{"key":`,
			profileType: config.ProfileTypeFreeform,
			wantErr:     true,
		},
		{
			name:        "empty JSON",
			content:     `{}`,
			profileType: config.ProfileTypeFreeform,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := config.NormalizeJSON(tt.content, tt.profileType)
			if (err != nil) != tt.wantErr {
				t.Errorf("NormalizeJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && result == "" {
				t.Error("NormalizeJSON() returned empty string for valid input")
			}
		})
	}
}

func TestNormalizeYAML(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr bool
	}{
		{
			name:    "valid YAML",
			content: "key: value\nnested:\n  a: 1\n",
			wantErr: false,
		},
		{
			name:    "invalid YAML - malformed",
			content: ":\n  invalid yaml\n:",
			wantErr: true,
		},
		{
			name:    "invalid YAML - bad indentation",
			content: "key:\nvalue",
			wantErr: false, // This is actually valid YAML
		},
		{
			name:    "empty YAML",
			content: "",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := config.NormalizeYAML(tt.content)
			if (err != nil) != tt.wantErr {
				t.Errorf("NormalizeYAML() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && result == "" && tt.content != "" {
				t.Error("NormalizeYAML() returned empty string for non-empty valid input")
			}
		})
	}
}

func TestNormalizeText(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name:     "unix line endings",
			content:  "line1\nline2\nline3",
			expected: "line1\nline2\nline3\n",
		},
		{
			name:     "windows line endings",
			content:  "line1\r\nline2\r\nline3",
			expected: "line1\nline2\nline3\n",
		},
		{
			name:     "mixed line endings",
			content:  "line1\nline2\r\nline3",
			expected: "line1\nline2\nline3\n",
		},
		{
			name:     "multiple trailing newlines",
			content:  "line1\nline2\n\n\n",
			expected: "line1\nline2\n",
		},
		{
			name:     "no trailing newline",
			content:  "line1\nline2",
			expected: "line1\nline2\n",
		},
		{
			name:     "empty string",
			content:  "",
			expected: "\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := config.NormalizeText(tt.content)
			if result != tt.expected {
				t.Errorf("NormalizeText() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestNormalizeContentForComparison(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		ext         string
		profileType string
		wantErr     bool
	}{
		{
			name:        "JSON file",
			content:     `{"key":"value"}`,
			ext:         ".json",
			profileType: config.ProfileTypeFreeform,
			wantErr:     false,
		},
		{
			name:        "YAML file",
			content:     "key: value\n",
			ext:         ".yaml",
			profileType: config.ProfileTypeFreeform,
			wantErr:     false,
		},
		{
			name:        "YML file",
			content:     "key: value\n",
			ext:         ".yml",
			profileType: config.ProfileTypeFreeform,
			wantErr:     false,
		},
		{
			name:        "text file",
			content:     "plain text\ncontent",
			ext:         ".txt",
			profileType: config.ProfileTypeFreeform,
			wantErr:     false,
		},
		{
			name:        "invalid JSON",
			content:     `{invalid}`,
			ext:         ".json",
			profileType: config.ProfileTypeFreeform,
			wantErr:     true,
		},
		{
			name:        "invalid YAML",
			content:     ":\ninvalid\n:",
			ext:         ".yaml",
			profileType: config.ProfileTypeFreeform,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := normalizeContentForComparison(tt.content, tt.ext, tt.profileType)
			if (err != nil) != tt.wantErr {
				t.Errorf("normalizeContentForComparison() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && result == "" {
				t.Error("normalizeContentForComparison() returned empty string for valid input")
			}
		})
	}
}

func TestHasConfigurationChanges(t *testing.T) {
	tests := []struct {
		name                 string
		localContent         []byte
		remoteContent        []byte
		fileName             string
		profileType          string
		hasDeployment        bool
		wantChanges          bool
		wantErr              bool
		mockDeployment       *types.DeploymentSummary
		listDeploymentsError error
		getVersionError      error
	}{
		{
			name:          "no previous deployment",
			localContent:  []byte(`{"key":"value"}`),
			fileName:      "config.json",
			profileType:   config.ProfileTypeFreeform,
			hasDeployment: false,
			wantChanges:   true,
			wantErr:       false,
		},
		{
			name:                 "error listing deployments",
			localContent:         []byte(`{"key":"value"}`),
			fileName:             "config.json",
			profileType:          config.ProfileTypeFreeform,
			hasDeployment:        false,
			wantChanges:          false,
			wantErr:              true,
			listDeploymentsError: errors.New("failed to list deployments"),
		},
		{
			name:          "error getting version",
			localContent:  []byte(`{"key":"value"}`),
			fileName:      "config.json",
			profileType:   config.ProfileTypeFreeform,
			hasDeployment: true,
			wantChanges:   false,
			wantErr:       true,
			mockDeployment: &types.DeploymentSummary{
				ConfigurationVersion: aws.String("1"),
				State:                types.DeploymentStateComplete,
			},
			getVersionError: errors.New("failed to get version"),
		},
		{
			name:          "invalid local JSON",
			localContent:  []byte(`{invalid}`),
			remoteContent: []byte(`{"key":"value"}`),
			fileName:      "config.json",
			profileType:   config.ProfileTypeFreeform,
			hasDeployment: true,
			wantChanges:   false,
			wantErr:       true,
			mockDeployment: &types.DeploymentSummary{
				ConfigurationVersion: aws.String("1"),
				State:                types.DeploymentStateComplete,
			},
		},
		{
			name:          "invalid remote JSON",
			localContent:  []byte(`{"key":"value"}`),
			remoteContent: []byte(`{invalid}`),
			fileName:      "config.json",
			profileType:   config.ProfileTypeFreeform,
			hasDeployment: true,
			wantChanges:   false,
			wantErr:       true,
			mockDeployment: &types.DeploymentSummary{
				ConfigurationVersion: aws.String("1"),
				State:                types.DeploymentStateComplete,
			},
		},
		{
			name:          "identical JSON content",
			localContent:  []byte(`{"key":"value"}`),
			remoteContent: []byte(`{"key":"value"}`),
			fileName:      "config.json",
			profileType:   config.ProfileTypeFreeform,
			hasDeployment: true,
			wantChanges:   false,
			wantErr:       false,
			mockDeployment: &types.DeploymentSummary{
				ConfigurationVersion: aws.String("1"),
				State:                types.DeploymentStateComplete,
			},
		},
		{
			name:          "different JSON content",
			localContent:  []byte(`{"key":"new-value"}`),
			remoteContent: []byte(`{"key":"old-value"}`),
			fileName:      "config.json",
			profileType:   config.ProfileTypeFreeform,
			hasDeployment: true,
			wantChanges:   true,
			wantErr:       false,
			mockDeployment: &types.DeploymentSummary{
				ConfigurationVersion: aws.String("1"),
				State:                types.DeploymentStateComplete,
			},
		},
		{
			name:          "identical YAML content",
			localContent:  []byte("key: value\n"),
			remoteContent: []byte("key: value\n"),
			fileName:      "config.yaml",
			profileType:   config.ProfileTypeFreeform,
			hasDeployment: true,
			wantChanges:   false,
			wantErr:       false,
			mockDeployment: &types.DeploymentSummary{
				ConfigurationVersion: aws.String("1"),
				State:                types.DeploymentStateComplete,
			},
		},
		{
			name:          "different YAML content",
			localContent:  []byte("key: new-value\n"),
			remoteContent: []byte("key: old-value\n"),
			fileName:      "config.yaml",
			profileType:   config.ProfileTypeFreeform,
			hasDeployment: true,
			wantChanges:   true,
			wantErr:       false,
			mockDeployment: &types.DeploymentSummary{
				ConfigurationVersion: aws.String("1"),
				State:                types.DeploymentStateComplete,
			},
		},
		{
			name:          "identical text content",
			localContent:  []byte("text content"),
			remoteContent: []byte("text content"),
			fileName:      "config.txt",
			profileType:   config.ProfileTypeFreeform,
			hasDeployment: true,
			wantChanges:   false,
			wantErr:       false,
			mockDeployment: &types.DeploymentSummary{
				ConfigurationVersion: aws.String("1"),
				State:                types.DeploymentStateComplete,
			},
		},
		{
			name:          "different text content",
			localContent:  []byte("new text content"),
			remoteContent: []byte("old text content"),
			fileName:      "config.txt",
			profileType:   config.ProfileTypeFreeform,
			hasDeployment: true,
			wantChanges:   true,
			wantErr:       false,
			mockDeployment: &types.DeploymentSummary{
				ConfigurationVersion: aws.String("1"),
				State:                types.DeploymentStateComplete,
			},
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

			mockClient := &mock.MockAppConfigClient{
				ListDeploymentsFunc: func(ctx context.Context, params *appconfig.ListDeploymentsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListDeploymentsOutput, error) {
					if tt.listDeploymentsError != nil {
						return nil, tt.listDeploymentsError
					}
					if !tt.hasDeployment {
						return &appconfig.ListDeploymentsOutput{
							Items: []types.DeploymentSummary{},
						}, nil
					}
					return &appconfig.ListDeploymentsOutput{
						Items: []types.DeploymentSummary{*tt.mockDeployment},
					}, nil
				},
				GetDeploymentFunc: func(ctx context.Context, params *appconfig.GetDeploymentInput, optFns ...func(*appconfig.Options)) (*appconfig.GetDeploymentOutput, error) {
					if !tt.hasDeployment {
						return nil, errors.New("no deployment found")
					}
					return &appconfig.GetDeploymentOutput{
						ConfigurationProfileId: aws.String("profile-123"),
						ConfigurationVersion:   tt.mockDeployment.ConfigurationVersion,
						State:                  types.DeploymentStateComplete,
					}, nil
				},
				GetHostedConfigurationVersionFunc: func(ctx context.Context, params *appconfig.GetHostedConfigurationVersionInput, optFns ...func(*appconfig.Options)) (*appconfig.GetHostedConfigurationVersionOutput, error) {
					if tt.getVersionError != nil {
						return nil, tt.getVersionError
					}
					return &appconfig.GetHostedConfigurationVersionOutput{
						Content: tt.remoteContent,
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
					Type: tt.profileType,
				},
			}

			hasChanges, err := deployer.HasConfigurationChanges(context.Background(), resolved, tt.localContent, tt.fileName, "application/json")
			if (err != nil) != tt.wantErr {
				t.Fatalf("HasConfigurationChanges() error = %v, wantErr %v", err, tt.wantErr)
			}

			if hasChanges != tt.wantChanges {
				t.Errorf("HasConfigurationChanges() = %v, want %v", hasChanges, tt.wantChanges)
			}
		})
	}
}
