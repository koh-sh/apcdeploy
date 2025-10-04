package config

import (
	"testing"
)

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: Config{
				Application:          "MyApp",
				ConfigurationProfile: "MyProfile",
				Environment:          "Production",
				DeploymentStrategy:   "AppConfig.AllAtOnce",
				DataFile:             "data.json",
			},
			wantErr: false,
		},
		{
			name: "missing application",
			config: Config{
				ConfigurationProfile: "MyProfile",
				Environment:          "Production",
				DeploymentStrategy:   "AppConfig.AllAtOnce",
				DataFile:             "data.json",
			},
			wantErr: true,
		},
		{
			name: "missing configuration profile",
			config: Config{
				Application:        "MyApp",
				Environment:        "Production",
				DeploymentStrategy: "AppConfig.AllAtOnce",
				DataFile:           "data.json",
			},
			wantErr: true,
		},
		{
			name: "missing environment",
			config: Config{
				Application:          "MyApp",
				ConfigurationProfile: "MyProfile",
				DeploymentStrategy:   "AppConfig.AllAtOnce",
				DataFile:             "data.json",
			},
			wantErr: true,
		},
		{
			name: "missing data file",
			config: Config{
				Application:          "MyApp",
				ConfigurationProfile: "MyProfile",
				Environment:          "Production",
				DeploymentStrategy:   "AppConfig.AllAtOnce",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfigDefaults(t *testing.T) {
	tests := []struct {
		name     string
		config   Config
		expected string
	}{
		{
			name: "default deployment strategy applied",
			config: Config{
				Application:          "MyApp",
				ConfigurationProfile: "MyProfile",
				Environment:          "Production",
				DataFile:             "data.json",
			},
			expected: "AppConfig.AllAtOnce",
		},
		{
			name: "existing deployment strategy not overridden",
			config: Config{
				Application:          "MyApp",
				ConfigurationProfile: "MyProfile",
				Environment:          "Production",
				DataFile:             "data.json",
				DeploymentStrategy:   "AppConfig.Linear50PercentEvery30Seconds",
			},
			expected: "AppConfig.Linear50PercentEvery30Seconds",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.config.setDefaults()
			if tt.config.DeploymentStrategy != tt.expected {
				t.Errorf("Expected deployment strategy '%s', got '%s'", tt.expected, tt.config.DeploymentStrategy)
			}
		})
	}
}

func TestConfigFields(t *testing.T) {
	tests := []struct {
		name          string
		config        Config
		expectedField string
		expectedValue string
	}{
		{
			name: "region field",
			config: Config{
				Application:          "MyApp",
				ConfigurationProfile: "MyProfile",
				Environment:          "Production",
				DataFile:             "data.json",
				Region:               "us-west-2",
			},
			expectedField: "region",
			expectedValue: "us-west-2",
		},
		{
			name: "application field",
			config: Config{
				Application:          "TestApplication",
				ConfigurationProfile: "MyProfile",
				Environment:          "Production",
				DataFile:             "data.json",
			},
			expectedField: "application",
			expectedValue: "TestApplication",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got string
			switch tt.expectedField {
			case "region":
				got = tt.config.Region
			case "application":
				got = tt.config.Application
			}
			if got != tt.expectedValue {
				t.Errorf("Expected %s = '%s', got '%s'", tt.expectedField, tt.expectedValue, got)
			}
		})
	}
}
