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

func TestRunInitErrorPaths(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		errContains string
	}{
		{
			name: "AWS client initialization fails without region",
			args: []string{
				"--app", "test-app",
				"--profile", "test-profile",
				"--env", "test-env",
			},
			errContains: "failed to initialize AWS client",
		},
		{
			name: "AWS client initialization fails with invalid region",
			args: []string{
				"--app", "test-app",
				"--profile", "test-profile",
				"--env", "test-env",
				"--region", "invalid-region-123",
			},
			errContains: "failed to resolve application",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := newInitCommand()
			cmd.SetArgs(tt.args)

			err := cmd.Execute()

			if err == nil {
				t.Fatal("expected error, got nil")
			}

			if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
				t.Errorf("error = %v, want to contain %v", err, tt.errContains)
			}
		})
	}
}

func TestRunInitWithRegion(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "apcdeploy.yml")

	cmd := newInitCommand()
	cmd.SetArgs([]string{
		"--app", "test-app",
		"--profile", "test-profile",
		"--env", "test-env",
		"--region", "us-east-1",
		"--config", configPath,
	})

	err := cmd.Execute()

	// This will fail at AWS resource resolution since we don't have real resources
	if err == nil {
		t.Error("expected error, got nil")
	}

	// Should fail at resource resolution, not at client initialization
	if contains(err.Error(), "region must be specified") {
		t.Errorf("should not fail at region validation, got error: %v", err)
	}
}

func TestRunInitWithCustomOutputData(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "apcdeploy.yml")

	cmd := newInitCommand()
	cmd.SetArgs([]string{
		"--app", "test-app",
		"--profile", "test-profile",
		"--env", "test-env",
		"--region", "us-east-1",
		"--config", configPath,
		"--output-data", "custom-data.json",
	})

	err := cmd.Execute()

	// This will fail at AWS resource resolution
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestCliReporter(t *testing.T) {
	tests := []struct {
		name    string
		method  string
		message string
	}{
		{
			name:    "Progress method",
			method:  "Progress",
			message: "Processing...",
		},
		{
			name:    "Success method",
			method:  "Success",
			message: "Operation completed",
		},
		{
			name:    "Warning method",
			method:  "Warning",
			message: "Warning message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reporter := &cliReporter{}

			// Call the method (we're just checking it doesn't panic)
			switch tt.method {
			case "Progress":
				reporter.Progress(tt.message)
			case "Success":
				reporter.Success(tt.message)
			case "Warning":
				reporter.Warning(tt.message)
			}
		})
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || (len(s) > 0 && len(substr) > 0 && containsHelper(s, substr)))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
