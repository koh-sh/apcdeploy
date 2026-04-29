package run

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
	reportertest "github.com/koh-sh/apcdeploy/internal/reporter/testing"
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
	awsClient := awsInternal.NewTestClient(mockClient)

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

			awsClient := awsInternal.NewTestClient(mockClient)

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

			awsClient := awsInternal.NewTestClient(mockClient)
			awsClient.PollingInterval = 100 * time.Millisecond // Fast polling for tests

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

	awsClient := awsInternal.NewTestClient(mockClient)

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

	awsClient := awsInternal.NewTestClient(mockClient)

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

			awsClient := awsInternal.NewTestClient(mockClient)

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

// TestMakeDeployTick verifies the state-machine branches of the deploy
// tick factory across both bakingLabel modes:
//   - "Baking..."     (--wait-deploy: bar terminates with bake label)
//   - "Deploying..."  (--wait-bake's deploy phase: bar terminates with
//     deploy label, then a separate spinner takes over)
//
// All other states route to the "Deploying..." label with a wall-clock
// remaining-time suffix. Tests fire the tick immediately after closure
// construction so elapsed is effectively zero, making the suffix
// deterministic.
func TestMakeDeployTick(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		bakingLabel   string
		state         types.DeploymentState
		percent       float64
		totalDuration time.Duration
		wantPercent   float64
		wantMsg       string
	}{
		// --wait-deploy mode (bakingLabel = "Baking...")
		{"wait-deploy: deploying mid no total", "Baking...", types.DeploymentStateDeploying, 42.5, 0, 42.5, "Deploying... (<1 min left)"},
		{"wait-deploy: deploying with total", "Baking...", types.DeploymentStateDeploying, 30, 10 * time.Minute, 30, "Deploying... (~10 min left)"},
		{"wait-deploy: baking pins to 100 with bake label", "Baking...", types.DeploymentStateBaking, 30, 10 * time.Minute, 100, "Baking..."},
		{"wait-deploy: complete pins to 100 with bake label", "Baking...", types.DeploymentStateComplete, 100, 10 * time.Minute, 100, "Baking..."},

		// --wait-bake's deploy phase (bakingLabel = "Deploying...")
		{"wait-bake: deploying mid no total", "Deploying...", types.DeploymentStateDeploying, 42.5, 0, 42.5, "Deploying... (<1 min left)"},
		{"wait-bake: deploying with total", "Deploying...", types.DeploymentStateDeploying, 25, 8 * time.Minute, 25, "Deploying... (~8 min left)"},
		{"wait-bake: baking pins to 100 with deploy label", "Deploying...", types.DeploymentStateBaking, 30, 8 * time.Minute, 100, "Deploying..."},
		{"wait-bake: complete pins to 100 with deploy label", "Deploying...", types.DeploymentStateComplete, 100, 8 * time.Minute, 100, "Deploying..."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := &reportertest.MockReporter{}
			pb := m.Progress("init")
			tick := MakeDeployTick(pb, tt.bakingLabel)
			tick(tt.state, tt.percent, tt.totalDuration)

			if len(m.ProgressCalls) != 1 || len(m.ProgressCalls[0].Updates) != 1 {
				t.Fatalf("expected exactly one update; got %+v", m.ProgressCalls)
			}
			got := m.ProgressCalls[0].Updates[0]
			if got.Percent != tt.wantPercent || got.Message != tt.wantMsg {
				t.Errorf("update = %+v, want percent=%v msg=%q", got, tt.wantPercent, tt.wantMsg)
			}
		})
	}
}

// TestRemainingFromElapsedSuffix exercises the time-boundary edge cases
// (sub-minute, exact minute, overshoot, zero total) directly. The
// MakeDeployTick / MakeBakeTick tests cover routing
// only because elapsed is timing-dependent; this table-driven test
// covers the formatting decisions independently.
func TestRemainingFromElapsedSuffix(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		elapsed time.Duration
		total   time.Duration
		want    string
	}{
		{"zero total falls back to <1 min", 0, 0, " (<1 min left)"},
		{"negative-looking total falls back to <1 min", 0, -5 * time.Minute, " (<1 min left)"},
		{"start of window shows full duration", 0, 10 * time.Minute, " (~10 min left)"},
		{"mid window rounds up partial minute", 3 * time.Minute, 10 * time.Minute, " (~7 min left)"},
		{"exactly one minute remaining renders as 1 min", 9 * time.Minute, 10 * time.Minute, " (~1 min left)"},
		{"non-integer remaining rounds up", 0, 2*time.Minute + 30*time.Second, " (~3 min left)"},
		{"thirty seconds remaining clamps to <1 min", 9*time.Minute + 30*time.Second, 10 * time.Minute, " (<1 min left)"},
		{"sub-second remaining clamps to <1 min", 9*time.Minute + 59*time.Second + 500*time.Millisecond, 10 * time.Minute, " (<1 min left)"},
		{"elapsed equals total clamps to <1 min", 10 * time.Minute, 10 * time.Minute, " (<1 min left)"},
		{"elapsed exceeds total clamps to <1 min", 12 * time.Minute, 10 * time.Minute, " (<1 min left)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := remainingFromElapsedSuffix(tt.elapsed, tt.total); got != tt.want {
				t.Errorf("remainingFromElapsedSuffix(%v, %v) = %q, want %q", tt.elapsed, tt.total, got, tt.want)
			}
		})
	}
}

// TestMakeBakeTick verifies that the bake-spinner tick updates the
// underlying Spinner's label with a "(~N min left)" suffix derived from
// elapsed/total.
func TestMakeBakeTick(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		elapsed time.Duration
		total   time.Duration
		wantMsg string
	}{
		{"zero total falls back to <1 min", 0, 0, "Baking... (<1 min left)"},
		{"early in bake shows full window", 0, 10 * time.Minute, "Baking... (~10 min left)"},
		{"mid bake shows remaining", 5 * time.Minute, 10 * time.Minute, "Baking... (~5 min left)"},
		{"sub-minute remaining clamps to <1 min", 9*time.Minute + 30*time.Second, 10 * time.Minute, "Baking... (<1 min left)"},
		{"elapsed equals total clamps to <1 min", 10 * time.Minute, 10 * time.Minute, "Baking... (<1 min left)"},
		{"elapsed exceeds total clamps to <1 min", 11 * time.Minute, 10 * time.Minute, "Baking... (<1 min left)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := &reportertest.MockReporter{}
			sp := m.Spin("init")
			tick := MakeBakeTick(sp)
			tick(tt.elapsed, tt.total)

			if len(m.SpinnerCalls) != 1 || len(m.SpinnerCalls[0].Updates) != 1 {
				t.Fatalf("expected exactly one update; got %+v", m.SpinnerCalls)
			}
			got := m.SpinnerCalls[0].Updates[0]
			if got != tt.wantMsg {
				t.Errorf("update = %q, want %q", got, tt.wantMsg)
			}
		})
	}
}
