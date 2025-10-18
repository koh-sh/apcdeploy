package init

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/appconfig"
	"github.com/aws/aws-sdk-go-v2/service/appconfig/types"
	awsInternal "github.com/koh-sh/apcdeploy/internal/aws"
	"github.com/koh-sh/apcdeploy/internal/aws/mock"
	"github.com/koh-sh/apcdeploy/internal/config"
	reportertest "github.com/koh-sh/apcdeploy/internal/reporter/testing"
)

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
						Type: aws.String(config.ProfileTypeFeatureFlags),
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
			errContains: "application not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &mock.MockAppConfigClient{}
			tt.mockSetup(mockClient)

			awsClient := awsInternal.NewTestClient(mockClient)
			reporter := &reportertest.MockReporter{}
			initializer := New(awsClient, reporter)

			result, err := initializer.resolveResources(context.Background(), tt.opts)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
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
		validate    func(*testing.T, *Result, *reportertest.MockReporter)
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
			validate: func(t *testing.T, result *Result, reporter *reportertest.MockReporter) {
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
			validate: func(t *testing.T, result *Result, reporter *reportertest.MockReporter) {
				if result.VersionInfo != nil {
					t.Error("expected VersionInfo to be nil")
				}
				hasWarning := false
				for _, msg := range reporter.Messages {
					if len(msg) > 8 && msg[:8] == "warning:" {
						hasWarning = true
						break
					}
				}
				if !hasWarning {
					t.Error("expected warning message")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &mock.MockAppConfigClient{}
			tt.mockSetup(mockClient)

			awsClient := awsInternal.NewTestClient(mockClient)
			reporter := &reportertest.MockReporter{}
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

func TestInitializer_FetchDeploymentStrategy(t *testing.T) {
	tests := []struct {
		name          string
		mockSetup     func(*mock.MockAppConfigClient)
		wantStrategy  string
		checkMessages func(*testing.T, *reportertest.MockReporter)
	}{
		{
			name: "custom deployment strategy resolved to name",
			mockSetup: func(m *mock.MockAppConfigClient) {
				// Mock ListDeployments - return a deployment with custom strategy
				m.ListDeploymentsFunc = func(ctx context.Context, params *appconfig.ListDeploymentsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListDeploymentsOutput, error) {
					return &appconfig.ListDeploymentsOutput{
						Items: []types.DeploymentSummary{
							{DeploymentNumber: 1, State: types.DeploymentStateComplete},
						},
					}, nil
				}
				// Mock GetDeployment - return deployment with custom strategy ID
				m.GetDeploymentFunc = func(ctx context.Context, params *appconfig.GetDeploymentInput, optFns ...func(*appconfig.Options)) (*appconfig.GetDeploymentOutput, error) {
					return &appconfig.GetDeploymentOutput{
						DeploymentNumber:       1,
						ConfigurationProfileId: aws.String("prof-456"),
						ConfigurationVersion:   aws.String("1"),
						DeploymentStrategyId:   aws.String("abc123def"), // Custom strategy ID
						State:                  types.DeploymentStateComplete,
					}, nil
				}
				// Mock ListDeploymentStrategies - return custom strategy
				m.ListDeploymentStrategiesFunc = func(ctx context.Context, params *appconfig.ListDeploymentStrategiesInput, optFns ...func(*appconfig.Options)) (*appconfig.ListDeploymentStrategiesOutput, error) {
					return &appconfig.ListDeploymentStrategiesOutput{
						Items: []types.DeploymentStrategy{
							{Id: aws.String("abc123def"), Name: aws.String("MyCustomStrategy")},
						},
					}, nil
				}
			},
			wantStrategy: "MyCustomStrategy",
			checkMessages: func(t *testing.T, reporter *reportertest.MockReporter) {
				hasSuccess := false
				for _, msg := range reporter.Messages {
					if len(msg) > 8 && msg[:8] == "success:" {
						hasSuccess = true
						break
					}
				}
				if !hasSuccess {
					t.Error("expected success message")
				}
			},
		},
		{
			name: "predefined deployment strategy",
			mockSetup: func(m *mock.MockAppConfigClient) {
				m.ListDeploymentsFunc = func(ctx context.Context, params *appconfig.ListDeploymentsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListDeploymentsOutput, error) {
					return &appconfig.ListDeploymentsOutput{
						Items: []types.DeploymentSummary{
							{DeploymentNumber: 1, State: types.DeploymentStateComplete},
						},
					}, nil
				}
				m.GetDeploymentFunc = func(ctx context.Context, params *appconfig.GetDeploymentInput, optFns ...func(*appconfig.Options)) (*appconfig.GetDeploymentOutput, error) {
					return &appconfig.GetDeploymentOutput{
						DeploymentNumber:       1,
						ConfigurationProfileId: aws.String("prof-456"),
						ConfigurationVersion:   aws.String("1"),
						DeploymentStrategyId:   aws.String("AppConfig.AllAtOnce"),
						State:                  types.DeploymentStateComplete,
					}, nil
				}
			},
			wantStrategy: "AppConfig.AllAtOnce",
		},
		{
			name: "no previous deployments",
			mockSetup: func(m *mock.MockAppConfigClient) {
				m.ListDeploymentsFunc = func(ctx context.Context, params *appconfig.ListDeploymentsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListDeploymentsOutput, error) {
					return &appconfig.ListDeploymentsOutput{
						Items: []types.DeploymentSummary{},
					}, nil
				}
			},
			wantStrategy: "AppConfig.AllAtOnce",
			checkMessages: func(t *testing.T, reporter *reportertest.MockReporter) {
				hasWarning := false
				for _, msg := range reporter.Messages {
					if len(msg) > 8 && msg[:8] == "warning:" {
						hasWarning = true
						break
					}
				}
				if !hasWarning {
					t.Error("expected warning message")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &mock.MockAppConfigClient{}
			tt.mockSetup(mockClient)

			awsClient := awsInternal.NewTestClient(mockClient)
			reporter := &reportertest.MockReporter{}
			initializer := New(awsClient, reporter)

			result := &Result{
				AppID:     "app-123",
				ProfileID: "prof-456",
				EnvID:     "env-789",
			}

			initializer.fetchDeploymentStrategy(context.Background(), result)

			if result.DeploymentStrategy != tt.wantStrategy {
				t.Errorf("expected DeploymentStrategy %q, got %q", tt.wantStrategy, result.DeploymentStrategy)
			}

			if tt.checkMessages != nil {
				tt.checkMessages(t, reporter)
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
			awsClient := awsInternal.NewTestClient(mockClient)
			reporter := &reportertest.MockReporter{}
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
		validate    func(*testing.T, *Result, *reportertest.MockReporter)
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
						Type: aws.String(config.ProfileTypeFreeform),
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

				// Mock ListDeployments - return empty (no previous deployments)
				m.ListDeploymentsFunc = func(ctx context.Context, params *appconfig.ListDeploymentsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListDeploymentsOutput, error) {
					return &appconfig.ListDeploymentsOutput{
						Items: []types.DeploymentSummary{},
					}, nil
				}

				// Mock ListDeploymentStrategies - not needed but for completeness
				m.ListDeploymentStrategiesFunc = func(ctx context.Context, params *appconfig.ListDeploymentStrategiesInput, optFns ...func(*appconfig.Options)) (*appconfig.ListDeploymentStrategiesOutput, error) {
					return &appconfig.ListDeploymentStrategiesOutput{
						Items: []types.DeploymentStrategy{},
					}, nil
				}
			},
			wantErr: false,
			validate: func(t *testing.T, result *Result, reporter *reportertest.MockReporter) {
				if result.AppID != "app-123" {
					t.Errorf("expected AppID 'app-123', got %q", result.AppID)
				}
				if result.DataFile != "data.json" {
					t.Errorf("expected DataFile 'data.json', got %q", result.DataFile)
				}
				hasProgress := false
				for _, msg := range reporter.Messages {
					if len(msg) > 9 && msg[:9] == "progress:" {
						hasProgress = true
						break
					}
				}
				if !hasProgress {
					t.Error("expected progress messages")
				}
			},
		},
		{
			name: "error during resource resolution",
			opts: &Options{
				Application: "test-app",
				Profile:     "test-profile",
				Environment: "test-env",
				ConfigFile:  "test.yml",
			},
			mockSetup: func(m *mock.MockAppConfigClient) {
				m.ListApplicationsFunc = func(ctx context.Context, params *appconfig.ListApplicationsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListApplicationsOutput, error) {
					return nil, errors.New("api error")
				}
			},
			wantErr:     true,
			errContains: "api error",
		},
		{
			name: "error during file generation",
			opts: &Options{
				Application: "test-app",
				Profile:     "test-profile",
				Environment: "test-env",
				ConfigFile:  "/invalid/path/test.yml",
			},
			mockSetup: func(m *mock.MockAppConfigClient) {
				m.ListApplicationsFunc = func(ctx context.Context, params *appconfig.ListApplicationsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListApplicationsOutput, error) {
					return &appconfig.ListApplicationsOutput{
						Items: []types.Application{
							{Id: aws.String("app-123"), Name: aws.String("test-app")},
						},
					}, nil
				}
				m.ListConfigurationProfilesFunc = func(ctx context.Context, params *appconfig.ListConfigurationProfilesInput, optFns ...func(*appconfig.Options)) (*appconfig.ListConfigurationProfilesOutput, error) {
					return &appconfig.ListConfigurationProfilesOutput{
						Items: []types.ConfigurationProfileSummary{
							{Id: aws.String("prof-456"), Name: aws.String("test-profile")},
						},
					}, nil
				}
				m.GetConfigurationProfileFunc = func(ctx context.Context, params *appconfig.GetConfigurationProfileInput, optFns ...func(*appconfig.Options)) (*appconfig.GetConfigurationProfileOutput, error) {
					return &appconfig.GetConfigurationProfileOutput{
						Id:   aws.String("prof-456"),
						Name: aws.String("test-profile"),
						Type: aws.String(config.ProfileTypeFreeform),
					}, nil
				}
				m.ListEnvironmentsFunc = func(ctx context.Context, params *appconfig.ListEnvironmentsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListEnvironmentsOutput, error) {
					return &appconfig.ListEnvironmentsOutput{
						Items: []types.Environment{
							{Id: aws.String("env-789"), Name: aws.String("test-env")},
						},
					}, nil
				}
				m.ListHostedConfigurationVersionsFunc = func(ctx context.Context, params *appconfig.ListHostedConfigurationVersionsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListHostedConfigurationVersionsOutput, error) {
					return nil, errors.New("no versions")
				}
				// Mock ListDeployments - return empty
				m.ListDeploymentsFunc = func(ctx context.Context, params *appconfig.ListDeploymentsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListDeploymentsOutput, error) {
					return &appconfig.ListDeploymentsOutput{
						Items: []types.DeploymentSummary{},
					}, nil
				}

				// Mock ListDeploymentStrategies
				m.ListDeploymentStrategiesFunc = func(ctx context.Context, params *appconfig.ListDeploymentStrategiesInput, optFns ...func(*appconfig.Options)) (*appconfig.ListDeploymentStrategiesOutput, error) {
					return &appconfig.ListDeploymentStrategiesOutput{
						Items: []types.DeploymentStrategy{},
					}, nil
				}
			},
			wantErr:     true,
			errContains: "failed to generate config file",
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
						Type: aws.String(config.ProfileTypeFreeform),
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
				// Mock ListDeployments - return empty
				m.ListDeploymentsFunc = func(ctx context.Context, params *appconfig.ListDeploymentsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListDeploymentsOutput, error) {
					return &appconfig.ListDeploymentsOutput{
						Items: []types.DeploymentSummary{},
					}, nil
				}

				// Mock ListDeploymentStrategies
				m.ListDeploymentStrategiesFunc = func(ctx context.Context, params *appconfig.ListDeploymentStrategiesInput, optFns ...func(*appconfig.Options)) (*appconfig.ListDeploymentStrategiesOutput, error) {
					return &appconfig.ListDeploymentStrategiesOutput{
						Items: []types.DeploymentStrategy{},
					}, nil
				}
			},
			wantErr: false,
			validate: func(t *testing.T, result *Result, reporter *reportertest.MockReporter) {
				if result.VersionInfo != nil {
					t.Error("expected VersionInfo to be nil")
				}
				hasWarning := false
				for _, msg := range reporter.Messages {
					if len(msg) > 8 && msg[:8] == "warning:" {
						hasWarning = true
						break
					}
				}
				if !hasWarning {
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

			awsClient := awsInternal.NewTestClient(mockClient)
			reporter := &reportertest.MockReporter{}
			initializer := New(awsClient, reporter)

			result, err := initializer.Run(context.Background(), tt.opts)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
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
			awsClient := awsInternal.NewTestClient(mockClient)
			reporter := &reportertest.MockReporter{}
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
