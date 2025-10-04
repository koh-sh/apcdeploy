package config

import (
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "valid config file",
			path:    "../../testdata/config/valid.yml",
			wantErr: false,
		},
		{
			name:    "file not found",
			path:    "../../testdata/config/nonexistent.yml",
			wantErr: true,
		},
		{
			name:    "malformed YAML",
			path:    "../../testdata/config/malformed.yml",
			wantErr: true,
		},
		{
			name:    "invalid config - missing required fields",
			path:    "../../testdata/config/invalid.yml",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := LoadConfig(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoadConfigValues(t *testing.T) {
	tests := []struct {
		name                       string
		configPath                 string
		expectedApplication        string
		expectedProfile            string
		expectedEnvironment        string
		expectedDeploymentStrategy string
		expectedRegion             string
		expectedDataFileBase       string
		checkDataFileAbsolute      bool
	}{
		{
			name:                       "valid config with all fields",
			configPath:                 "../../testdata/config/valid.yml",
			expectedApplication:        "TestApp",
			expectedProfile:            "TestProfile",
			expectedEnvironment:        "Production",
			expectedDeploymentStrategy: "AppConfig.Linear50PercentEvery30Seconds",
			expectedRegion:             "us-east-1",
			expectedDataFileBase:       "data.json",
			checkDataFileAbsolute:      true,
		},
		{
			name:                       "config with defaults",
			configPath:                 "../../testdata/config/defaults.yml",
			expectedApplication:        "TestApp",
			expectedProfile:            "TestProfile",
			expectedEnvironment:        "Production",
			expectedDeploymentStrategy: "AppConfig.AllAtOnce",
			expectedRegion:             "",
			expectedDataFileBase:       "data.json",
			checkDataFileAbsolute:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := LoadConfig(tt.configPath)
			if err != nil {
				t.Fatalf("LoadConfig() error = %v", err)
			}

			if config.Application != tt.expectedApplication {
				t.Errorf("Expected application = %s, got %s", tt.expectedApplication, config.Application)
			}
			if config.ConfigurationProfile != tt.expectedProfile {
				t.Errorf("Expected configuration_profile = %s, got %s", tt.expectedProfile, config.ConfigurationProfile)
			}
			if config.Environment != tt.expectedEnvironment {
				t.Errorf("Expected environment = %s, got %s", tt.expectedEnvironment, config.Environment)
			}
			if config.DeploymentStrategy != tt.expectedDeploymentStrategy {
				t.Errorf("Expected deployment_strategy = %s, got %s", tt.expectedDeploymentStrategy, config.DeploymentStrategy)
			}
			if config.Region != tt.expectedRegion {
				t.Errorf("Expected region = %s, got %s", tt.expectedRegion, config.Region)
			}

			if tt.checkDataFileAbsolute && !filepath.IsAbs(config.DataFile) {
				t.Errorf("Expected data_file to be absolute path, got %s", config.DataFile)
			}
			if filepath.Base(config.DataFile) != tt.expectedDataFileBase {
				t.Errorf("Expected data_file basename = %s, got %s", tt.expectedDataFileBase, filepath.Base(config.DataFile))
			}
		})
	}
}

func TestResolveDataFilePath(t *testing.T) {
	tests := []struct {
		name       string
		configPath string
		dataFile   string
		wantAbs    bool
	}{
		{
			name:       "relative path resolution",
			configPath: "/path/to/apcdeploy.yml",
			dataFile:   "data.json",
			wantAbs:    true,
		},
		{
			name:       "absolute path unchanged",
			configPath: "/path/to/apcdeploy.yml",
			dataFile:   "/absolute/path/data.json",
			wantAbs:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ResolveDataFilePath(tt.configPath, tt.dataFile)
			if tt.wantAbs && !filepath.IsAbs(result) {
				t.Errorf("Expected absolute path, got %s", result)
			}
			if filepath.IsAbs(tt.dataFile) && result != tt.dataFile {
				t.Errorf("Expected absolute path unchanged, got %s", result)
			}
		})
	}
}
