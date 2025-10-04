package cmd

import (
	"path/filepath"
	"testing"
)

func TestInitCommand(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "missing required app flag",
			args:    []string{"--profile", "test-profile", "--env", "test-env"},
			wantErr: true,
			errMsg:  "required flag(s) \"app\" not set",
		},
		{
			name:    "missing required profile flag",
			args:    []string{"--app", "test-app", "--env", "test-env"},
			wantErr: true,
			errMsg:  "required flag(s) \"profile\" not set",
		},
		{
			name:    "missing required env flag",
			args:    []string{"--app", "test-app", "--profile", "test-profile"},
			wantErr: true,
			errMsg:  "required flag(s) \"env\" not set",
		},
		{
			name:    "all required flags provided without region",
			args:    []string{"--app", "test-app", "--profile", "test-profile", "--env", "test-env"},
			wantErr: true, // Expects region error
			errMsg:  "failed to initialize AWS client: region must be specified either via --region flag or AWS_REGION/AWS_DEFAULT_REGION environment variable",
		},
		{
			name:    "with optional config flag without region",
			args:    []string{"--app", "test-app", "--profile", "test-profile", "--env", "test-env", "--config", "custom.yml"},
			wantErr: true, // Expects region error
			errMsg:  "failed to initialize AWS client: region must be specified either via --region flag or AWS_REGION/AWS_DEFAULT_REGION environment variable",
		},
		{
			name:    "with optional region flag",
			args:    []string{"--app", "test-app", "--profile", "test-profile", "--env", "test-env", "--region", "us-west-2"},
			wantErr: true, // Will try to call AWS but fail on missing resources
		},
		{
			name:    "with optional output-data flag without region",
			args:    []string{"--app", "test-app", "--profile", "test-profile", "--env", "test-env", "--output-data", "custom-data.json"},
			wantErr: true, // Expects region error
			errMsg:  "failed to initialize AWS client: region must be specified either via --region flag or AWS_REGION/AWS_DEFAULT_REGION environment variable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset command flags for each test
			initCmd := newInitCommand()
			initCmd.SetArgs(tt.args)

			err := initCmd.Execute()

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got none")
				} else if tt.errMsg != "" && err.Error() != tt.errMsg {
					t.Errorf("expected error message %q, got %q", tt.errMsg, err.Error())
				}
			} else if err != nil && !tt.wantErr {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestInitCommandFlagDefaults(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		expectedFile string
	}{
		{
			name:         "default config file",
			args:         []string{"--app", "test-app", "--profile", "test-profile", "--env", "test-env"},
			expectedFile: "apcdeploy.yml",
		},
		{
			name:         "custom config file",
			args:         []string{"--app", "test-app", "--profile", "test-profile", "--env", "test-env", "-c", "custom.yml"},
			expectedFile: "custom.yml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := newInitCommand()
			cmd.SetArgs(tt.args)

			// Parse flags only (don't execute)
			err := cmd.ParseFlags(tt.args)
			if err != nil {
				t.Fatalf("failed to parse flags: %v", err)
			}

			configFile, _ := cmd.Flags().GetString("config")
			if configFile != tt.expectedFile {
				t.Errorf("expected config file %q, got %q", tt.expectedFile, configFile)
			}
		})
	}
}

func TestInitCommandIntegration(t *testing.T) {
	// This test verifies the integration flow
	// Without real AWS credentials or mocks, it will fail at AWS client initialization

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "apcdeploy.yml")

	cmd := newInitCommand()
	cmd.SetArgs([]string{
		"--app", "test-app",
		"--profile", "test-profile",
		"--env", "test-env",
		"--config", configPath,
	})

	err := cmd.Execute()

	// We expect an error because no region is specified and no AWS credentials are available
	if err == nil {
		t.Error("expected error but got none")
	}
	// The error should be about missing region or AWS configuration
	// This is expected behavior in a test environment
}
