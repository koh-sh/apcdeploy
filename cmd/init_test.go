package cmd

import (
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
			name:    "with optional output-data flag",
			args:    []string{"--app", "test-app", "--profile", "test-profile", "--env", "test-env", "--region", "us-east-1", "--output-data", "custom-data.json"},
			wantErr: false,
		},
		{
			name:    "with optional force flag",
			args:    []string{"--app", "test-app", "--profile", "test-profile", "--env", "test-env", "--region", "us-east-1", "--force"},
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
			configFile = "apcdeploy.yml"
			initOutputData = ""
			initForce = false

			cmd := newInitCmd()
			cmd.SetArgs(tt.args)

			err := cmd.ParseFlags(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseFlags() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestInitCommandSilenceUsage(t *testing.T) {
	cmd := newInitCmd()

	// SilenceUsage should be true to prevent usage display on runtime errors
	if !cmd.SilenceUsage {
		t.Error("init command should have SilenceUsage set to true")
	}
}

func TestInitCommandInteractiveMode(t *testing.T) {
	t.Skip("Interactive mode tests require TTY and would hang in automated test environments")
	// Note: Interactive mode is tested via e2e tests and manual testing.
	// Unit tests cannot properly test TTY interactions without mocking the entire flow.
}

func TestInitCommandFlagDefaults(t *testing.T) {
	// Config flag is tested in root_test.go as a persistent flag
	// Test init-specific flags
	cmd := newInitCmd()

	// Verify force flag exists with correct default
	forceFlag := cmd.Flags().Lookup("force")
	if forceFlag == nil {
		t.Error("force flag not found")
		return
	}
	if forceFlag.DefValue != "false" {
		t.Errorf("force flag default = %v, want false", forceFlag.DefValue)
	}

	// Verify output-data flag exists
	outputFlag := cmd.Flags().Lookup("output-data")
	if outputFlag == nil {
		t.Error("output-data flag not found")
	}
}

func TestInitCommandIntegration(t *testing.T) {
	t.Skip("Integration tests require TTY and AWS credentials, tested via e2e tests")
	// Note: This test would attempt to open a TTY for region selection
	// when --region flag is not provided. Integration is tested in e2e/e2e-test.sh
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

func TestRunInitWithAllFlags(t *testing.T) {
	tests := []struct {
		name        string
		app         string
		profile     string
		env         string
		region      string
		config      string
		outputData  string
		force       bool
		expectError bool
	}{
		{
			name:        "all flags provided",
			app:         "test-app",
			profile:     "test-profile",
			env:         "test-env",
			region:      "us-east-1",
			config:      "apcdeploy.yml",
			outputData:  "data.json",
			force:       false,
			expectError: true, // AWS client creation will fail without credentials
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set global flags
			initApp = tt.app
			initProfile = tt.profile
			initEnv = tt.env
			initRegion = tt.region
			configFile = tt.config
			initOutputData = tt.outputData
			initForce = tt.force

			err := runInit(nil, nil)

			if tt.expectError && err == nil {
				t.Error("expected error due to AWS client creation without credentials")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}
