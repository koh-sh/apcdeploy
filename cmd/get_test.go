package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetCommand(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "no config file specified uses default",
			args:    []string{},
			wantErr: false,
		},
		{
			name:    "custom config file",
			args:    []string{"--config", "custom.yml"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset global flags for each test
			getConfigFile = "apcdeploy.yml"

			cmd := newGetCmd()
			cmd.SetArgs(tt.args)

			err := cmd.ParseFlags(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseFlags() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRunGet(t *testing.T) {
	tests := []struct {
		name       string
		setupFiles func(t *testing.T, dir string) string
		args       []string
		wantErr    bool
	}{
		{
			name: "missing config file",
			setupFiles: func(t *testing.T, dir string) string {
				return filepath.Join(dir, "nonexistent.yml")
			},
			args:    []string{},
			wantErr: true,
		},
		{
			name: "invalid config file",
			setupFiles: func(t *testing.T, dir string) string {
				configPath := filepath.Join(dir, "invalid.yml")
				err := os.WriteFile(configPath, []byte("invalid: yaml: content:\n  - bad"), 0o644)
				if err != nil {
					t.Fatalf("Failed to create test file: %v", err)
				}
				return configPath
			},
			args:    []string{},
			wantErr: true,
		},
		{
			name: "valid config but AWS error",
			setupFiles: func(t *testing.T, dir string) string {
				configPath := filepath.Join(dir, "valid.yml")
				content := `application: test-app
environment: test-env
configuration_profile: test-profile
deployment_strategy: test-strategy
data_file: data.json
`
				err := os.WriteFile(configPath, []byte(content), 0o644)
				if err != nil {
					t.Fatalf("Failed to create test file: %v", err)
				}
				return configPath
			},
			args:    []string{},
			wantErr: true, // Will fail due to AWS credentials/connection
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory
			tmpDir, err := os.MkdirTemp("", "get-test-*")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			// Setup test files
			configPath := tt.setupFiles(t, tmpDir)

			// Reset global flags
			getConfigFile = configPath

			// Create command
			cmd := newGetCmd()
			cmd.SetArgs(tt.args)

			// Execute command
			err = runGet(cmd, tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("runGet() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetCommandStructure(t *testing.T) {
	cmd := newGetCmd()

	if cmd.Use != "get" {
		t.Errorf("Use = %v, want get", cmd.Use)
	}

	if cmd.Short == "" {
		t.Error("Short description should not be empty")
	}

	if cmd.Long == "" {
		t.Error("Long description should not be empty")
	}

	if cmd.RunE == nil {
		t.Error("RunE should be set")
	}
}

func TestGetCommandFlags(t *testing.T) {
	getConfigFile = "apcdeploy.yml"

	cmd := newGetCmd()

	tests := []struct {
		name         string
		flagName     string
		defaultValue string
	}{
		{
			name:         "config flag has default",
			flagName:     "config",
			defaultValue: "apcdeploy.yml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag := cmd.Flags().Lookup(tt.flagName)
			if flag == nil {
				t.Errorf("Flag %s not found", tt.flagName)
				return
			}

			if flag.DefValue != tt.defaultValue {
				t.Errorf("Flag %s default = %v, want %v", tt.flagName, flag.DefValue, tt.defaultValue)
			}
		})
	}
}

func TestGetCommandSilenceUsage(t *testing.T) {
	cmd := newGetCmd()

	// SilenceUsage should be true to prevent usage display on runtime errors
	if !cmd.SilenceUsage {
		t.Error("get command should have SilenceUsage set to true")
	}
}
