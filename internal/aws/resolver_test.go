package aws

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/appconfig"
	"github.com/aws/aws-sdk-go-v2/service/appconfig/types"
	"github.com/koh-sh/apcdeploy/internal/aws/mock"
	"github.com/koh-sh/apcdeploy/internal/config"
)

func TestNewResolver(t *testing.T) {
	// Use the actual client since we just need to verify the constructor
	ctx := context.Background()

	// Set AWS region via environment to avoid errors
	t.Setenv("AWS_REGION", "us-east-1")

	client, err := NewClient(ctx, "us-east-1")
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	resolver := NewResolver(client)

	if resolver.client == nil {
		t.Error("resolver client should not be nil")
	}
}

func TestResolveApplication(t *testing.T) {
	tests := []struct {
		name        string
		appName     string
		mockApps    []types.Application
		mockErr     error
		wantID      string
		wantErr     bool
		errContains string
	}{
		{
			name:    "successful application resolution",
			appName: "test-app",
			mockApps: []types.Application{
				{
					Id:   aws.String("app-123"),
					Name: aws.String("test-app"),
				},
			},
			wantID:  "app-123",
			wantErr: false,
		},
		{
			name:    "application not found",
			appName: "non-existent",
			mockApps: []types.Application{
				{
					Id:   aws.String("app-123"),
					Name: aws.String("test-app"),
				},
			},
			wantErr:     true,
			errContains: "application not found",
		},
		{
			name:    "multiple applications match",
			appName: "test-app",
			mockApps: []types.Application{
				{
					Id:   aws.String("app-123"),
					Name: aws.String("test-app"),
				},
				{
					Id:   aws.String("app-456"),
					Name: aws.String("test-app"),
				},
			},
			wantErr:     true,
			errContains: "multiple applications found",
		},
		{
			name:        "API error",
			appName:     "test-app",
			mockErr:     errors.New("API error"),
			wantErr:     true,
			errContains: "failed to list applications",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &mock.MockAppConfigClient{
				ListApplicationsFunc: func(ctx context.Context, params *appconfig.ListApplicationsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListApplicationsOutput, error) {
					if tt.mockErr != nil {
						return nil, tt.mockErr
					}
					return &appconfig.ListApplicationsOutput{
						Items: tt.mockApps,
					}, nil
				},
			}

			resolver := &Resolver{
				client: mockClient,
			}

			ctx := context.Background()
			appID, err := resolver.ResolveApplication(ctx, tt.appName)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error = %v, want to contain %v", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if appID != tt.wantID {
				t.Errorf("appID = %v, want %v", appID, tt.wantID)
			}
		})
	}
}

func TestResolveApplicationPagination(t *testing.T) {
	t.Run("pagination - application found on second page", func(t *testing.T) {
		callCount := 0
		mockClient := &mock.MockAppConfigClient{
			ListApplicationsFunc: func(ctx context.Context, params *appconfig.ListApplicationsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListApplicationsOutput, error) {
				callCount++
				if callCount == 1 {
					// First page
					return &appconfig.ListApplicationsOutput{
						Items: []types.Application{
							{Id: aws.String("app-1"), Name: aws.String("app-1")},
							{Id: aws.String("app-2"), Name: aws.String("app-2")},
						},
						NextToken: aws.String("page2"),
					}, nil
				}
				// Second page
				return &appconfig.ListApplicationsOutput{
					Items: []types.Application{
						{Id: aws.String("app-3"), Name: aws.String("target-app")},
						{Id: aws.String("app-4"), Name: aws.String("app-4")},
					},
				}, nil
			},
		}

		resolver := &Resolver{client: mockClient}
		ctx := context.Background()
		appID, err := resolver.ResolveApplication(ctx, "target-app")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if appID != "app-3" {
			t.Errorf("appID = %v, want %v", appID, "app-3")
		}

		if callCount != 2 {
			t.Errorf("expected 2 API calls, got %d", callCount)
		}
	})
}

func TestResolveConfigurationProfile(t *testing.T) {
	tests := []struct {
		name         string
		appID        string
		profileName  string
		mockProfiles []types.ConfigurationProfileSummary
		mockProfile  *appconfig.GetConfigurationProfileOutput
		mockListErr  error
		mockGetErr   error
		wantID       string
		wantType     string
		wantErr      bool
		errContains  string
	}{
		{
			name:        "successful freeform profile resolution",
			appID:       "app-123",
			profileName: "test-profile",
			mockProfiles: []types.ConfigurationProfileSummary{
				{
					Id:   aws.String("prof-456"),
					Name: aws.String("test-profile"),
				},
			},
			mockProfile: &appconfig.GetConfigurationProfileOutput{
				Id:   aws.String("prof-456"),
				Name: aws.String("test-profile"),
				Type: aws.String(config.ProfileTypeFreeform),
			},
			wantID:   "prof-456",
			wantType: config.ProfileTypeFreeform,
			wantErr:  false,
		},
		{
			name:        "successful feature flags profile resolution",
			appID:       "app-123",
			profileName: "feature-flags",
			mockProfiles: []types.ConfigurationProfileSummary{
				{
					Id:   aws.String("prof-789"),
					Name: aws.String("feature-flags"),
				},
			},
			mockProfile: &appconfig.GetConfigurationProfileOutput{
				Id:   aws.String("prof-789"),
				Name: aws.String("feature-flags"),
				Type: aws.String(config.ProfileTypeFeatureFlags),
			},
			wantID:   "prof-789",
			wantType: config.ProfileTypeFeatureFlags,
			wantErr:  false,
		},
		{
			name:        "profile not found",
			appID:       "app-123",
			profileName: "non-existent",
			mockProfiles: []types.ConfigurationProfileSummary{
				{
					Id:   aws.String("prof-456"),
					Name: aws.String("test-profile"),
				},
			},
			wantErr:     true,
			errContains: "configuration profile not found",
		},
		{
			name:        "multiple profiles match",
			appID:       "app-123",
			profileName: "test-profile",
			mockProfiles: []types.ConfigurationProfileSummary{
				{
					Id:   aws.String("prof-456"),
					Name: aws.String("test-profile"),
				},
				{
					Id:   aws.String("prof-789"),
					Name: aws.String("test-profile"),
				},
			},
			wantErr:     true,
			errContains: "multiple configuration profiles found",
		},
		{
			name:        "list API error",
			appID:       "app-123",
			profileName: "test-profile",
			mockListErr: errors.New("API error"),
			wantErr:     true,
			errContains: "failed to list configuration profiles",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &mock.MockAppConfigClient{
				ListConfigurationProfilesFunc: func(ctx context.Context, params *appconfig.ListConfigurationProfilesInput, optFns ...func(*appconfig.Options)) (*appconfig.ListConfigurationProfilesOutput, error) {
					if tt.mockListErr != nil {
						return nil, tt.mockListErr
					}
					return &appconfig.ListConfigurationProfilesOutput{
						Items: tt.mockProfiles,
					}, nil
				},
				GetConfigurationProfileFunc: func(ctx context.Context, params *appconfig.GetConfigurationProfileInput, optFns ...func(*appconfig.Options)) (*appconfig.GetConfigurationProfileOutput, error) {
					if tt.mockGetErr != nil {
						return nil, tt.mockGetErr
					}
					return tt.mockProfile, nil
				},
			}

			resolver := &Resolver{
				client: mockClient,
			}

			ctx := context.Background()
			profile, err := resolver.ResolveConfigurationProfile(ctx, tt.appID, tt.profileName)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error = %v, want to contain %v", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if profile.ID != tt.wantID {
				t.Errorf("profile.ID = %v, want %v", profile.ID, tt.wantID)
			}

			if profile.Type != tt.wantType {
				t.Errorf("profile.Type = %v, want %v", profile.Type, tt.wantType)
			}
		})
	}
}

func TestResolveConfigurationProfilePagination(t *testing.T) {
	t.Run("pagination - profile found on second page", func(t *testing.T) {
		callCount := 0
		mockClient := &mock.MockAppConfigClient{
			ListConfigurationProfilesFunc: func(ctx context.Context, params *appconfig.ListConfigurationProfilesInput, optFns ...func(*appconfig.Options)) (*appconfig.ListConfigurationProfilesOutput, error) {
				callCount++
				if callCount == 1 {
					// First page
					return &appconfig.ListConfigurationProfilesOutput{
						Items: []types.ConfigurationProfileSummary{
							{Id: aws.String("prof-1"), Name: aws.String("prof-1")},
							{Id: aws.String("prof-2"), Name: aws.String("prof-2")},
						},
						NextToken: aws.String("page2"),
					}, nil
				}
				// Second page
				return &appconfig.ListConfigurationProfilesOutput{
					Items: []types.ConfigurationProfileSummary{
						{Id: aws.String("prof-3"), Name: aws.String("target-profile")},
						{Id: aws.String("prof-4"), Name: aws.String("prof-4")},
					},
				}, nil
			},
			GetConfigurationProfileFunc: func(ctx context.Context, params *appconfig.GetConfigurationProfileInput, optFns ...func(*appconfig.Options)) (*appconfig.GetConfigurationProfileOutput, error) {
				return &appconfig.GetConfigurationProfileOutput{
					Id:   aws.String("prof-3"),
					Name: aws.String("target-profile"),
					Type: aws.String(config.ProfileTypeFreeform),
				}, nil
			},
		}

		resolver := &Resolver{client: mockClient}
		ctx := context.Background()
		profile, err := resolver.ResolveConfigurationProfile(ctx, "app-123", "target-profile")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if profile.ID != "prof-3" {
			t.Errorf("profile.ID = %v, want %v", profile.ID, "prof-3")
		}

		if callCount != 2 {
			t.Errorf("expected 2 API calls, got %d", callCount)
		}
	})
}

func TestResolveEnvironment(t *testing.T) {
	tests := []struct {
		name        string
		appID       string
		envName     string
		mockEnvs    []types.Environment
		mockErr     error
		wantID      string
		wantErr     bool
		errContains string
	}{
		{
			name:    "successful environment resolution",
			appID:   "app-123",
			envName: "production",
			mockEnvs: []types.Environment{
				{
					Id:   aws.String("env-456"),
					Name: aws.String("production"),
				},
			},
			wantID:  "env-456",
			wantErr: false,
		},
		{
			name:    "environment not found",
			appID:   "app-123",
			envName: "non-existent",
			mockEnvs: []types.Environment{
				{
					Id:   aws.String("env-456"),
					Name: aws.String("production"),
				},
			},
			wantErr:     true,
			errContains: "environment not found",
		},
		{
			name:    "multiple environments match",
			appID:   "app-123",
			envName: "production",
			mockEnvs: []types.Environment{
				{
					Id:   aws.String("env-456"),
					Name: aws.String("production"),
				},
				{
					Id:   aws.String("env-789"),
					Name: aws.String("production"),
				},
			},
			wantErr:     true,
			errContains: "multiple environments found",
		},
		{
			name:        "API error",
			appID:       "app-123",
			envName:     "production",
			mockErr:     errors.New("API error"),
			wantErr:     true,
			errContains: "failed to list environments",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &mock.MockAppConfigClient{
				ListEnvironmentsFunc: func(ctx context.Context, params *appconfig.ListEnvironmentsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListEnvironmentsOutput, error) {
					if tt.mockErr != nil {
						return nil, tt.mockErr
					}
					return &appconfig.ListEnvironmentsOutput{
						Items: tt.mockEnvs,
					}, nil
				},
			}

			resolver := &Resolver{
				client: mockClient,
			}

			ctx := context.Background()
			envID, err := resolver.ResolveEnvironment(ctx, tt.appID, tt.envName)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error = %v, want to contain %v", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if envID != tt.wantID {
				t.Errorf("envID = %v, want %v", envID, tt.wantID)
			}
		})
	}
}

func TestResolveEnvironmentPagination(t *testing.T) {
	t.Run("pagination - environment found on second page", func(t *testing.T) {
		callCount := 0
		mockClient := &mock.MockAppConfigClient{
			ListEnvironmentsFunc: func(ctx context.Context, params *appconfig.ListEnvironmentsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListEnvironmentsOutput, error) {
				callCount++
				if callCount == 1 {
					// First page
					return &appconfig.ListEnvironmentsOutput{
						Items: []types.Environment{
							{Id: aws.String("env-1"), Name: aws.String("env-1")},
							{Id: aws.String("env-2"), Name: aws.String("env-2")},
						},
						NextToken: aws.String("page2"),
					}, nil
				}
				// Second page
				return &appconfig.ListEnvironmentsOutput{
					Items: []types.Environment{
						{Id: aws.String("env-3"), Name: aws.String("target-env")},
						{Id: aws.String("env-4"), Name: aws.String("env-4")},
					},
				}, nil
			},
		}

		resolver := &Resolver{client: mockClient}
		ctx := context.Background()
		envID, err := resolver.ResolveEnvironment(ctx, "app-123", "target-env")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if envID != "env-3" {
			t.Errorf("envID = %v, want %v", envID, "env-3")
		}

		if callCount != 2 {
			t.Errorf("expected 2 API calls, got %d", callCount)
		}
	})
}

func TestResolveDeploymentStrategy(t *testing.T) {
	tests := []struct {
		name           string
		strategyName   string
		mockStrategies []types.DeploymentStrategy
		mockErr        error
		wantID         string
		wantErr        bool
		errContains    string
	}{
		{
			name:         "successful strategy resolution",
			strategyName: "AppConfig.Linear50PercentEvery30Seconds",
			mockStrategies: []types.DeploymentStrategy{
				{
					Id:   aws.String("strategy-123"),
					Name: aws.String("AppConfig.Linear50PercentEvery30Seconds"),
				},
			},
			wantID:  "strategy-123",
			wantErr: false,
		},
		{
			name:         "successful default strategy resolution",
			strategyName: "AppConfig.AllAtOnce",
			mockStrategies: []types.DeploymentStrategy{
				{
					Id:   aws.String("strategy-456"),
					Name: aws.String("AppConfig.AllAtOnce"),
				},
			},
			wantID:  "strategy-456",
			wantErr: false,
		},
		{
			name:         "strategy not found",
			strategyName: "NonExistentStrategy",
			mockStrategies: []types.DeploymentStrategy{
				{
					Id:   aws.String("strategy-123"),
					Name: aws.String("AppConfig.AllAtOnce"),
				},
			},
			wantErr:     true,
			errContains: "deployment strategy not found",
		},
		{
			name:         "API error",
			strategyName: "AppConfig.AllAtOnce",
			mockErr:      errors.New("API error"),
			wantErr:      true,
			errContains:  "failed to list deployment strategies",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &mock.MockAppConfigClient{
				ListDeploymentStrategiesFunc: func(ctx context.Context, params *appconfig.ListDeploymentStrategiesInput, optFns ...func(*appconfig.Options)) (*appconfig.ListDeploymentStrategiesOutput, error) {
					if tt.mockErr != nil {
						return nil, tt.mockErr
					}
					return &appconfig.ListDeploymentStrategiesOutput{
						Items: tt.mockStrategies,
					}, nil
				},
			}

			resolver := &Resolver{
				client: mockClient,
			}

			ctx := context.Background()
			strategyID, err := resolver.ResolveDeploymentStrategy(ctx, tt.strategyName)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error = %v, want to contain %v", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if strategyID != tt.wantID {
				t.Errorf("strategyID = %v, want %v", strategyID, tt.wantID)
			}
		})
	}
}

func TestResolveDeploymentStrategyPagination(t *testing.T) {
	t.Run("pagination - strategy found on second page", func(t *testing.T) {
		callCount := 0
		mockClient := &mock.MockAppConfigClient{
			ListDeploymentStrategiesFunc: func(ctx context.Context, params *appconfig.ListDeploymentStrategiesInput, optFns ...func(*appconfig.Options)) (*appconfig.ListDeploymentStrategiesOutput, error) {
				callCount++
				if callCount == 1 {
					// First page
					return &appconfig.ListDeploymentStrategiesOutput{
						Items: []types.DeploymentStrategy{
							{Id: aws.String("start-1"), Name: aws.String("start-1")},
							{Id: aws.String("start-2"), Name: aws.String("start-2")},
						},
						NextToken: aws.String("page2"),
					}, nil
				}
				// Second page
				return &appconfig.ListDeploymentStrategiesOutput{
					Items: []types.DeploymentStrategy{
						{Id: aws.String("start-3"), Name: aws.String("target-strategy")},
						{Id: aws.String("start-4"), Name: aws.String("start-4")},
					},
				}, nil
			},
		}

		resolver := &Resolver{client: mockClient}
		ctx := context.Background()
		strategyID, err := resolver.ResolveDeploymentStrategy(ctx, "target-strategy")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if strategyID != "start-3" {
			t.Errorf("strategyID = %v, want %v", strategyID, "start-3")
		}

		if callCount != 2 {
			t.Errorf("expected 2 API calls, got %d", callCount)
		}
	})
}

func TestResolveDeploymentStrategyIDToName(t *testing.T) {
	tests := []struct {
		name           string
		strategyID     string
		mockStrategies []types.DeploymentStrategy
		mockErr        error
		wantName       string
		wantErr        bool
		errContains    string
	}{
		{
			name:       "predefined strategy - AllAtOnce",
			strategyID: config.StrategyPrefixPredefined + "AllAtOnce",
			wantName:   config.StrategyPrefixPredefined + "AllAtOnce",
			wantErr:    false,
		},
		{
			name:       "predefined strategy - Linear50PercentEvery30Seconds",
			strategyID: config.StrategyPrefixPredefined + "Linear50PercentEvery30Seconds",
			wantName:   config.StrategyPrefixPredefined + "Linear50PercentEvery30Seconds",
			wantErr:    false,
		},
		{
			name:       "custom strategy found",
			strategyID: "abc123def",
			mockStrategies: []types.DeploymentStrategy{
				{
					Id:   aws.String("abc123def"),
					Name: aws.String("MyCustomStrategy"),
				},
			},
			wantName: "MyCustomStrategy",
			wantErr:  false,
		},
		{
			name:       "custom strategy not found - return ID",
			strategyID: "xyz789",
			mockStrategies: []types.DeploymentStrategy{
				{
					Id:   aws.String("abc123def"),
					Name: aws.String("MyCustomStrategy"),
				},
			},
			wantName: "xyz789",
			wantErr:  false,
		},
		{
			name:       "custom strategy with no name - return ID",
			strategyID: "abc123def",
			mockStrategies: []types.DeploymentStrategy{
				{
					Id:   aws.String("abc123def"),
					Name: nil,
				},
			},
			wantName: "abc123def",
			wantErr:  false,
		},
		{
			name:        "API error",
			strategyID:  "abc123def",
			mockErr:     errors.New("API error"),
			wantErr:     true,
			errContains: "failed to list deployment strategies",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &mock.MockAppConfigClient{
				ListDeploymentStrategiesFunc: func(ctx context.Context, params *appconfig.ListDeploymentStrategiesInput, optFns ...func(*appconfig.Options)) (*appconfig.ListDeploymentStrategiesOutput, error) {
					if tt.mockErr != nil {
						return nil, tt.mockErr
					}
					return &appconfig.ListDeploymentStrategiesOutput{
						Items: tt.mockStrategies,
					}, nil
				},
			}

			resolver := &Resolver{
				client: mockClient,
			}

			ctx := context.Background()
			strategyName, err := resolver.ResolveDeploymentStrategyIDToName(ctx, tt.strategyID)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error = %v, want to contain %v", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if strategyName != tt.wantName {
				t.Errorf("strategyName = %v, want %v", strategyName, tt.wantName)
			}
		})
	}
}

func TestResolveDeploymentStrategyIDToNamePagination(t *testing.T) {
	t.Run("pagination - strategy found on second page", func(t *testing.T) {
		callCount := 0
		mockClient := &mock.MockAppConfigClient{
			ListDeploymentStrategiesFunc: func(ctx context.Context, params *appconfig.ListDeploymentStrategiesInput, optFns ...func(*appconfig.Options)) (*appconfig.ListDeploymentStrategiesOutput, error) {
				callCount++
				if callCount == 1 {
					// First page
					return &appconfig.ListDeploymentStrategiesOutput{
						Items: []types.DeploymentStrategy{
							{Id: aws.String("id-1"), Name: aws.String("start-1")},
							{Id: aws.String("id-2"), Name: aws.String("start-2")},
						},
						NextToken: aws.String("page2"),
					}, nil
				}
				// Second page
				return &appconfig.ListDeploymentStrategiesOutput{
					Items: []types.DeploymentStrategy{
						{Id: aws.String("target-id"), Name: aws.String("target-strategy")},
						{Id: aws.String("id-4"), Name: aws.String("start-4")},
					},
				}, nil
			},
		}

		resolver := &Resolver{client: mockClient}
		ctx := context.Background()
		strategyName, err := resolver.ResolveDeploymentStrategyIDToName(ctx, "target-id")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if strategyName != "target-strategy" {
			t.Errorf("strategyName = %v, want %v", strategyName, "target-strategy")
		}

		if callCount != 2 {
			t.Errorf("expected 2 API calls, got %d", callCount)
		}
	})
}

func TestResolveAll(t *testing.T) {
	tests := []struct {
		name            string
		appName         string
		profileName     string
		envName         string
		strategyName    string
		mockApps        []types.Application
		mockProfiles    []types.ConfigurationProfileSummary
		mockProfile     *appconfig.GetConfigurationProfileOutput
		mockEnvs        []types.Environment
		mockStrategies  []types.DeploymentStrategy
		wantAppID       string
		wantProfileID   string
		wantProfileType string
		wantEnvID       string
		wantStrategyID  string
		wantErr         bool
	}{
		{
			name:         "successful resolution of all resources",
			appName:      "test-app",
			profileName:  "test-profile",
			envName:      "production",
			strategyName: "AppConfig.AllAtOnce",
			mockApps: []types.Application{
				{
					Id:   aws.String("app-123"),
					Name: aws.String("test-app"),
				},
			},
			mockProfiles: []types.ConfigurationProfileSummary{
				{
					Id:   aws.String("prof-456"),
					Name: aws.String("test-profile"),
				},
			},
			mockProfile: &appconfig.GetConfigurationProfileOutput{
				Id:   aws.String("prof-456"),
				Name: aws.String("test-profile"),
				Type: aws.String(config.ProfileTypeFreeform),
			},
			mockEnvs: []types.Environment{
				{
					Id:   aws.String("env-789"),
					Name: aws.String("production"),
				},
			},
			mockStrategies: []types.DeploymentStrategy{
				{
					Id:   aws.String("strategy-101"),
					Name: aws.String("AppConfig.AllAtOnce"),
				},
			},
			wantAppID:       "app-123",
			wantProfileID:   "prof-456",
			wantProfileType: config.ProfileTypeFreeform,
			wantEnvID:       "env-789",
			wantStrategyID:  "strategy-101",
			wantErr:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &mock.MockAppConfigClient{
				ListApplicationsFunc: func(ctx context.Context, params *appconfig.ListApplicationsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListApplicationsOutput, error) {
					return &appconfig.ListApplicationsOutput{
						Items: tt.mockApps,
					}, nil
				},
				ListConfigurationProfilesFunc: func(ctx context.Context, params *appconfig.ListConfigurationProfilesInput, optFns ...func(*appconfig.Options)) (*appconfig.ListConfigurationProfilesOutput, error) {
					return &appconfig.ListConfigurationProfilesOutput{
						Items: tt.mockProfiles,
					}, nil
				},
				GetConfigurationProfileFunc: func(ctx context.Context, params *appconfig.GetConfigurationProfileInput, optFns ...func(*appconfig.Options)) (*appconfig.GetConfigurationProfileOutput, error) {
					return tt.mockProfile, nil
				},
				ListEnvironmentsFunc: func(ctx context.Context, params *appconfig.ListEnvironmentsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListEnvironmentsOutput, error) {
					return &appconfig.ListEnvironmentsOutput{
						Items: tt.mockEnvs,
					}, nil
				},
				ListDeploymentStrategiesFunc: func(ctx context.Context, params *appconfig.ListDeploymentStrategiesInput, optFns ...func(*appconfig.Options)) (*appconfig.ListDeploymentStrategiesOutput, error) {
					return &appconfig.ListDeploymentStrategiesOutput{
						Items: tt.mockStrategies,
					}, nil
				},
			}

			resolver := &Resolver{
				client: mockClient,
			}

			ctx := context.Background()
			result, err := resolver.ResolveAll(ctx, tt.appName, tt.profileName, tt.envName, tt.strategyName)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if result.ApplicationID != tt.wantAppID {
				t.Errorf("result.ApplicationID = %v, want %v", result.ApplicationID, tt.wantAppID)
			}

			if result.Profile.ID != tt.wantProfileID {
				t.Errorf("result.Profile.ID = %v, want %v", result.Profile.ID, tt.wantProfileID)
			}

			if result.Profile.Type != tt.wantProfileType {
				t.Errorf("result.Profile.Type = %v, want %v", result.Profile.Type, tt.wantProfileType)
			}

			if result.EnvironmentID != tt.wantEnvID {
				t.Errorf("result.EnvironmentID = %v, want %v", result.EnvironmentID, tt.wantEnvID)
			}

			if result.DeploymentStrategyID != tt.wantStrategyID {
				t.Errorf("result.DeploymentStrategyID = %v, want %v", result.DeploymentStrategyID, tt.wantStrategyID)
			}
		})
	}
}
