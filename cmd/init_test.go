package cmd

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/appconfig"
	"github.com/aws/aws-sdk-go-v2/service/appconfig/types"
	awsInternal "github.com/koh-sh/apcdeploy/internal/aws"
	"github.com/koh-sh/apcdeploy/internal/aws/mock"
	initPkg "github.com/koh-sh/apcdeploy/internal/init"
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

func TestInitCommandWithMock(t *testing.T) {
	// Save original factory and restore after test
	originalFactory := initializerFactory
	defer func() { initializerFactory = originalFactory }()

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "apcdeploy.yml")

	// Create mock AWS client
	mockClient := &mock.MockAppConfigClient{
		ListApplicationsFunc: func(ctx context.Context, params *appconfig.ListApplicationsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListApplicationsOutput, error) {
			return &appconfig.ListApplicationsOutput{
				Items: []types.Application{{Id: aws.String("app-123"), Name: aws.String("test-app")}},
			}, nil
		},
		ListConfigurationProfilesFunc: func(ctx context.Context, params *appconfig.ListConfigurationProfilesInput, optFns ...func(*appconfig.Options)) (*appconfig.ListConfigurationProfilesOutput, error) {
			return &appconfig.ListConfigurationProfilesOutput{
				Items: []types.ConfigurationProfileSummary{{Id: aws.String("profile-123"), Name: aws.String("test-profile"), Type: aws.String("AWS.Freeform")}},
			}, nil
		},
		GetConfigurationProfileFunc: func(ctx context.Context, params *appconfig.GetConfigurationProfileInput, optFns ...func(*appconfig.Options)) (*appconfig.GetConfigurationProfileOutput, error) {
			return &appconfig.GetConfigurationProfileOutput{Id: aws.String("profile-123"), Type: aws.String("AWS.Freeform")}, nil
		},
		ListEnvironmentsFunc: func(ctx context.Context, params *appconfig.ListEnvironmentsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListEnvironmentsOutput, error) {
			return &appconfig.ListEnvironmentsOutput{
				Items: []types.Environment{{Id: aws.String("env-123"), Name: aws.String("test-env")}},
			}, nil
		},
		ListHostedConfigurationVersionsFunc: func(ctx context.Context, params *appconfig.ListHostedConfigurationVersionsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListHostedConfigurationVersionsOutput, error) {
			return &appconfig.ListHostedConfigurationVersionsOutput{
				Items: []types.HostedConfigurationVersionSummary{
					{VersionNumber: 1},
				},
			}, nil
		},
		GetHostedConfigurationVersionFunc: func(ctx context.Context, params *appconfig.GetHostedConfigurationVersionInput, optFns ...func(*appconfig.Options)) (*appconfig.GetHostedConfigurationVersionOutput, error) {
			return &appconfig.GetHostedConfigurationVersionOutput{
				Content:     []byte(`{"test": "data"}`),
				ContentType: aws.String("application/json"),
			}, nil
		},
	}

	// Set factory to return mock-based initializer
	initializerFactory = func(ctx context.Context, region string) (*initPkg.Initializer, error) {
		awsClient := &awsInternal.Client{AppConfig: mockClient}
		return initPkg.New(awsClient, &cliReporter{}), nil
	}

	cmd := newInitCommand()
	cmd.SetArgs([]string{
		"--app", "test-app",
		"--profile", "test-profile",
		"--env", "test-env",
		"--region", "us-east-1",
		"--config", configPath,
	})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify config file was created
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("expected config file to be created")
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
