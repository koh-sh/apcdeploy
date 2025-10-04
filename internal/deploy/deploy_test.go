package deploy

import (
	"context"
	"os"
	"path/filepath"
	"testing"

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
			cfg, dataContent, err := LoadConfiguration(tt.configPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadConfiguration() error = %v, wantErr %v", err, tt.wantErr)
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

	cfg, dataContent, err := LoadConfiguration(configPath)
	if err != nil {
		t.Fatalf("LoadConfiguration() error = %v", err)
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
