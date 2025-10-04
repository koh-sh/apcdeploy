package init

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/appconfig"
	"github.com/aws/aws-sdk-go-v2/service/appconfig/types"
	awsInternal "github.com/koh-sh/apcdeploy/internal/aws"
	"github.com/koh-sh/apcdeploy/internal/aws/mock"
)

// mockReporter is a mock implementation of ProgressReporter for testing
type mockReporter struct {
	progressMessages []string
	successMessages  []string
	warningMessages  []string
}

func (m *mockReporter) Progress(message string) {
	m.progressMessages = append(m.progressMessages, message)
}

func (m *mockReporter) Success(message string) {
	m.successMessages = append(m.successMessages, message)
}

func (m *mockReporter) Warning(message string) {
	m.warningMessages = append(m.warningMessages, message)
}

func TestInitializer_ResolveResources(t *testing.T) {
	tests := []struct {
		name        string
		opts        *Options
		mockSetup   func(*mock.MockAppConfigClient)
		wantErr     bool
		errContains string
		validate    func(*testing.T, *Result)
	}{
		{
			name: "successful resource resolution",
			opts: &Options{
				Application: "test-app",
				Profile:     "test-profile",
				Environment: "test-env",
				ConfigFile:  "apcdeploy.yml",
			},
			mockSetup: func(m *mock.MockAppConfigClient) {
				// Mock ListApplications
				m.ListApplicationsFunc = func(ctx context.Context, params *appconfig.ListApplicationsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListApplicationsOutput, error) {
					return &appconfig.ListApplicationsOutput{
						Items: []types.Application{
							{Id: aws.String("app-123"), Name: aws.String("test-app")},
						},
					}, nil
				}

				// Mock ListConfigurationProfiles
				m.ListConfigurationProfilesFunc = func(ctx context.Context, params *appconfig.ListConfigurationProfilesInput, optFns ...func(*appconfig.Options)) (*appconfig.ListConfigurationProfilesOutput, error) {
					return &appconfig.ListConfigurationProfilesOutput{
						Items: []types.ConfigurationProfileSummary{
							{Id: aws.String("prof-456"), Name: aws.String("test-profile")},
						},
					}, nil
				}

				// Mock GetConfigurationProfile
				m.GetConfigurationProfileFunc = func(ctx context.Context, params *appconfig.GetConfigurationProfileInput, optFns ...func(*appconfig.Options)) (*appconfig.GetConfigurationProfileOutput, error) {
					return &appconfig.GetConfigurationProfileOutput{
						Id:   aws.String("prof-456"),
						Name: aws.String("test-profile"),
						Type: aws.String("AWS.AppConfig.FeatureFlags"),
					}, nil
				}

				// Mock ListEnvironments
				m.ListEnvironmentsFunc = func(ctx context.Context, params *appconfig.ListEnvironmentsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListEnvironmentsOutput, error) {
					return &appconfig.ListEnvironmentsOutput{
						Items: []types.Environment{
							{Id: aws.String("env-789"), Name: aws.String("test-env")},
						},
					}, nil
				}
			},
			wantErr: false,
			validate: func(t *testing.T, result *Result) {
				if result.AppID != "app-123" {
					t.Errorf("expected AppID 'app-123', got %q", result.AppID)
				}
				if result.ProfileID != "prof-456" {
					t.Errorf("expected ProfileID 'prof-456', got %q", result.ProfileID)
				}
				if result.EnvID != "env-789" {
					t.Errorf("expected EnvID 'env-789', got %q", result.EnvID)
				}
			},
		},
		{
			name: "application not found",
			opts: &Options{
				Application: "nonexistent-app",
				Profile:     "test-profile",
				Environment: "test-env",
			},
			mockSetup: func(m *mock.MockAppConfigClient) {
				m.ListApplicationsFunc = func(ctx context.Context, params *appconfig.ListApplicationsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListApplicationsOutput, error) {
					return &appconfig.ListApplicationsOutput{Items: []types.Application{}}, nil
				}
			},
			wantErr:     true,
			errContains: "failed to resolve application",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &mock.MockAppConfigClient{}
			tt.mockSetup(mockClient)

			awsClient := &awsInternal.Client{AppConfig: mockClient}
			reporter := &mockReporter{}
			initializer := New(awsClient, reporter)

			result, err := initializer.resolveResources(context.Background(), tt.opts)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("error = %v, want to contain %q", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if tt.validate != nil {
					tt.validate(t, result)
				}
			}
		})
	}
}

func TestInitializer_FetchConfigVersion(t *testing.T) {
	tests := []struct {
		name        string
		mockSetup   func(*mock.MockAppConfigClient)
		wantWarning bool
		validate    func(*testing.T, *Result, *mockReporter)
	}{
		{
			name: "version found",
			mockSetup: func(m *mock.MockAppConfigClient) {
				m.ListHostedConfigurationVersionsFunc = func(ctx context.Context, params *appconfig.ListHostedConfigurationVersionsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListHostedConfigurationVersionsOutput, error) {
					return &appconfig.ListHostedConfigurationVersionsOutput{
						Items: []types.HostedConfigurationVersionSummary{
							{VersionNumber: 1, ContentType: aws.String("application/json")},
						},
					}, nil
				}
				m.GetHostedConfigurationVersionFunc = func(ctx context.Context, params *appconfig.GetHostedConfigurationVersionInput, optFns ...func(*appconfig.Options)) (*appconfig.GetHostedConfigurationVersionOutput, error) {
					return &appconfig.GetHostedConfigurationVersionOutput{
						Content:     []byte(`{"key":"value"}`),
						ContentType: aws.String("application/json"),
					}, nil
				}
			},
			wantWarning: false,
			validate: func(t *testing.T, result *Result, reporter *mockReporter) {
				if result.VersionInfo == nil {
					t.Error("expected VersionInfo to be set")
				}
				if result.VersionInfo != nil && result.VersionInfo.VersionNumber != 1 {
					t.Errorf("expected version number 1, got %d", result.VersionInfo.VersionNumber)
				}
			},
		},
		{
			name: "no versions found",
			mockSetup: func(m *mock.MockAppConfigClient) {
				m.ListHostedConfigurationVersionsFunc = func(ctx context.Context, params *appconfig.ListHostedConfigurationVersionsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListHostedConfigurationVersionsOutput, error) {
					return nil, errors.New("no configuration versions found")
				}
			},
			wantWarning: true,
			validate: func(t *testing.T, result *Result, reporter *mockReporter) {
				if result.VersionInfo != nil {
					t.Error("expected VersionInfo to be nil")
				}
				if len(reporter.warningMessages) == 0 {
					t.Error("expected warning message")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &mock.MockAppConfigClient{}
			tt.mockSetup(mockClient)

			awsClient := &awsInternal.Client{AppConfig: mockClient}
			reporter := &mockReporter{}
			initializer := New(awsClient, reporter)

			result := &Result{
				AppID:     "app-123",
				ProfileID: "prof-456",
			}

			err := initializer.fetchConfigVersion(context.Background(), result)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.validate != nil {
				tt.validate(t, result, reporter)
			}
		})
	}
}

func TestInitializer_DetermineDataFileName(t *testing.T) {
	tests := []struct {
		name     string
		opts     *Options
		version  *awsInternal.ConfigVersionInfo
		expected string
	}{
		{
			name: "custom output data specified",
			opts: &Options{
				OutputData: "custom.json",
			},
			version:  nil,
			expected: "custom.json",
		},
		{
			name: "determine from version content type",
			opts: &Options{},
			version: &awsInternal.ConfigVersionInfo{
				ContentType: "application/json",
			},
			expected: "data.json",
		},
		{
			name:     "default when no version",
			opts:     &Options{},
			version:  nil,
			expected: "data.json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &mock.MockAppConfigClient{}
			awsClient := &awsInternal.Client{AppConfig: mockClient}
			reporter := &mockReporter{}
			initializer := New(awsClient, reporter)

			result := &Result{
				VersionInfo: tt.version,
			}

			initializer.determineDataFileName(tt.opts, result)

			if result.DataFile != tt.expected {
				t.Errorf("expected DataFile %q, got %q", tt.expected, result.DataFile)
			}
		})
	}
}

func TestInitializer_Run(t *testing.T) {
	tests := []struct {
		name        string
		opts        *Options
		mockSetup   func(*mock.MockAppConfigClient)
		wantErr     bool
		errContains string
		validate    func(*testing.T, *Result, *mockReporter)
	}{
		{
			name: "successful complete flow",
			opts: &Options{
				Application: "test-app",
				Profile:     "test-profile",
				Environment: "test-env",
				ConfigFile:  "test.yml",
			},
			mockSetup: func(m *mock.MockAppConfigClient) {
				// Mock ListApplications
				m.ListApplicationsFunc = func(ctx context.Context, params *appconfig.ListApplicationsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListApplicationsOutput, error) {
					return &appconfig.ListApplicationsOutput{
						Items: []types.Application{
							{Id: aws.String("app-123"), Name: aws.String("test-app")},
						},
					}, nil
				}

				// Mock ListConfigurationProfiles
				m.ListConfigurationProfilesFunc = func(ctx context.Context, params *appconfig.ListConfigurationProfilesInput, optFns ...func(*appconfig.Options)) (*appconfig.ListConfigurationProfilesOutput, error) {
					return &appconfig.ListConfigurationProfilesOutput{
						Items: []types.ConfigurationProfileSummary{
							{Id: aws.String("prof-456"), Name: aws.String("test-profile")},
						},
					}, nil
				}

				// Mock GetConfigurationProfile
				m.GetConfigurationProfileFunc = func(ctx context.Context, params *appconfig.GetConfigurationProfileInput, optFns ...func(*appconfig.Options)) (*appconfig.GetConfigurationProfileOutput, error) {
					return &appconfig.GetConfigurationProfileOutput{
						Id:   aws.String("prof-456"),
						Name: aws.String("test-profile"),
						Type: aws.String("AWS.AppConfig.Freeform"),
					}, nil
				}

				// Mock ListEnvironments
				m.ListEnvironmentsFunc = func(ctx context.Context, params *appconfig.ListEnvironmentsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListEnvironmentsOutput, error) {
					return &appconfig.ListEnvironmentsOutput{
						Items: []types.Environment{
							{Id: aws.String("env-789"), Name: aws.String("test-env")},
						},
					}, nil
				}

				// Mock ListHostedConfigurationVersions
				m.ListHostedConfigurationVersionsFunc = func(ctx context.Context, params *appconfig.ListHostedConfigurationVersionsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListHostedConfigurationVersionsOutput, error) {
					return &appconfig.ListHostedConfigurationVersionsOutput{
						Items: []types.HostedConfigurationVersionSummary{
							{VersionNumber: 1, ContentType: aws.String("application/json")},
						},
					}, nil
				}

				// Mock GetHostedConfigurationVersion
				m.GetHostedConfigurationVersionFunc = func(ctx context.Context, params *appconfig.GetHostedConfigurationVersionInput, optFns ...func(*appconfig.Options)) (*appconfig.GetHostedConfigurationVersionOutput, error) {
					return &appconfig.GetHostedConfigurationVersionOutput{
						Content:     []byte(`{"key":"value"}`),
						ContentType: aws.String("application/json"),
					}, nil
				}
			},
			wantErr: false,
			validate: func(t *testing.T, result *Result, reporter *mockReporter) {
				if result.AppID != "app-123" {
					t.Errorf("expected AppID 'app-123', got %q", result.AppID)
				}
				if result.DataFile != "data.json" {
					t.Errorf("expected DataFile 'data.json', got %q", result.DataFile)
				}
				if len(reporter.progressMessages) == 0 {
					t.Error("expected progress messages")
				}
			},
		},
		{
			name: "flow without version",
			opts: &Options{
				Application: "test-app",
				Profile:     "test-profile",
				Environment: "test-env",
				ConfigFile:  "test.yml",
			},
			mockSetup: func(m *mock.MockAppConfigClient) {
				// Mock ListApplications
				m.ListApplicationsFunc = func(ctx context.Context, params *appconfig.ListApplicationsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListApplicationsOutput, error) {
					return &appconfig.ListApplicationsOutput{
						Items: []types.Application{
							{Id: aws.String("app-123"), Name: aws.String("test-app")},
						},
					}, nil
				}

				// Mock ListConfigurationProfiles
				m.ListConfigurationProfilesFunc = func(ctx context.Context, params *appconfig.ListConfigurationProfilesInput, optFns ...func(*appconfig.Options)) (*appconfig.ListConfigurationProfilesOutput, error) {
					return &appconfig.ListConfigurationProfilesOutput{
						Items: []types.ConfigurationProfileSummary{
							{Id: aws.String("prof-456"), Name: aws.String("test-profile")},
						},
					}, nil
				}

				// Mock GetConfigurationProfile
				m.GetConfigurationProfileFunc = func(ctx context.Context, params *appconfig.GetConfigurationProfileInput, optFns ...func(*appconfig.Options)) (*appconfig.GetConfigurationProfileOutput, error) {
					return &appconfig.GetConfigurationProfileOutput{
						Id:   aws.String("prof-456"),
						Name: aws.String("test-profile"),
						Type: aws.String("AWS.AppConfig.Freeform"),
					}, nil
				}

				// Mock ListEnvironments
				m.ListEnvironmentsFunc = func(ctx context.Context, params *appconfig.ListEnvironmentsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListEnvironmentsOutput, error) {
					return &appconfig.ListEnvironmentsOutput{
						Items: []types.Environment{
							{Id: aws.String("env-789"), Name: aws.String("test-env")},
						},
					}, nil
				}

				// Mock no versions
				m.ListHostedConfigurationVersionsFunc = func(ctx context.Context, params *appconfig.ListHostedConfigurationVersionsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListHostedConfigurationVersionsOutput, error) {
					return nil, errors.New("no configuration versions found")
				}
			},
			wantErr: false,
			validate: func(t *testing.T, result *Result, reporter *mockReporter) {
				if result.VersionInfo != nil {
					t.Error("expected VersionInfo to be nil")
				}
				if len(reporter.warningMessages) == 0 {
					t.Error("expected warning message about no versions")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory for config files
			tempDir := t.TempDir()
			tt.opts.ConfigFile = tempDir + "/" + tt.opts.ConfigFile

			mockClient := &mock.MockAppConfigClient{}
			tt.mockSetup(mockClient)

			awsClient := &awsInternal.Client{AppConfig: mockClient}
			reporter := &mockReporter{}
			initializer := New(awsClient, reporter)

			result, err := initializer.Run(context.Background(), tt.opts)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("error = %v, want to contain %q", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if tt.validate != nil {
					tt.validate(t, result, reporter)
				}
			}
		})
	}
}

func TestInitializer_GenerateFiles(t *testing.T) {
	tests := []struct {
		name      string
		opts      *Options
		result    *Result
		wantErr   bool
		wantFiles []string
	}{
		{
			name: "generate config and data files",
			opts: &Options{
				ConfigFile: "apcdeploy.yml",
			},
			result: &Result{
				AppName:     "test-app",
				ProfileName: "test-profile",
				EnvName:     "test-env",
				DataFile:    "data.json",
				ConfigFile:  "apcdeploy.yml",
				VersionInfo: &awsInternal.ConfigVersionInfo{
					VersionNumber: 1,
					Content:       []byte(`{"key":"value"}`),
					ContentType:   "application/json",
				},
			},
			wantErr:   false,
			wantFiles: []string{"apcdeploy.yml", "data.json"},
		},
		{
			name: "generate only config file without version",
			opts: &Options{
				ConfigFile: "apcdeploy.yml",
			},
			result: &Result{
				AppName:     "test-app",
				ProfileName: "test-profile",
				EnvName:     "test-env",
				DataFile:    "data.json",
				ConfigFile:  "apcdeploy.yml",
				VersionInfo: nil,
			},
			wantErr:   false,
			wantFiles: []string{"apcdeploy.yml"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory
			tempDir := t.TempDir()
			tt.opts.ConfigFile = tempDir + "/" + tt.opts.ConfigFile
			tt.result.ConfigFile = tt.opts.ConfigFile

			mockClient := &mock.MockAppConfigClient{}
			awsClient := &awsInternal.Client{AppConfig: mockClient}
			reporter := &mockReporter{}
			initializer := New(awsClient, reporter)

			err := initializer.generateFiles(tt.opts, tt.result)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				// Check files exist
				for _, filename := range tt.wantFiles {
					fullPath := tempDir + "/" + filename
					if _, err := os.Stat(fullPath); err != nil {
						t.Errorf("expected file %s to exist, got error: %v", filename, err)
					}
				}
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
