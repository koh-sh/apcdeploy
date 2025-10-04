package cmd

import (
	"path/filepath"
	"testing"

	"github.com/koh-sh/apcdeploy/internal/cli"
)

func TestInitCommand(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "all required flags provided",
			args:    []string{"--app", "test-app", "--profile", "test-profile", "--env", "test-env", "--region", "us-east-1"},
			wantErr: false,
		},
		{
			name:    "with optional config flag",
			args:    []string{"--app", "test-app", "--profile", "test-profile", "--env", "test-env", "--region", "us-east-1", "--config", "custom.yml"},
			wantErr: false,
		},
		{
			name:    "with optional output-data flag",
			args:    []string{"--app", "test-app", "--profile", "test-profile", "--env", "test-env", "--region", "us-east-1", "--output-data", "custom-data.json"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset global flags for each test
			initApp = ""
			initProfile = ""
			initEnv = ""
			initRegion = ""
			initConfig = "apcdeploy.yml"
			initOutputData = ""

			cmd := newInitCmd()
			cmd.SetArgs(tt.args)

			err := cmd.ParseFlags(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseFlags() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestInitCommandRequiredFlags(t *testing.T) {
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset command flags for each test
			initCmd := newInitCmd()
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
			cmd := newInitCmd()
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

	cmd := newInitCmd()
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
			reporter := cli.NewReporter()

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

func TestRunInit(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T)
		wantErr bool
	}{
		{
			name: "no region specified",
			setup: func(t *testing.T) {
				initApp = "test-app"
				initProfile = "test-profile"
				initEnv = "test-env"
				initRegion = ""
				initConfig = "apcdeploy.yml"
				initOutputData = ""
			},
			wantErr: true,
		},
		{
			name: "valid flags but AWS error",
			setup: func(t *testing.T) {
				tmpDir := t.TempDir()
				initApp = "test-app"
				initProfile = "test-profile"
				initEnv = "test-env"
				initRegion = "us-east-1"
				initConfig = filepath.Join(tmpDir, "apcdeploy.yml")
				initOutputData = ""
			},
			wantErr: true, // Will fail due to AWS credentials/connection
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup(t)

			err := runInit(nil, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("runInit() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
