package init

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/account"
	accountTypes "github.com/aws/aws-sdk-go-v2/service/account/types"
	"github.com/aws/aws-sdk-go-v2/service/appconfig"
	appconfigTypes "github.com/aws/aws-sdk-go-v2/service/appconfig/types"
	"github.com/charmbracelet/huh"
	awsInternal "github.com/koh-sh/apcdeploy/internal/aws"
	awsMock "github.com/koh-sh/apcdeploy/internal/aws/mock"
	promptTesting "github.com/koh-sh/apcdeploy/internal/prompt/testing"
	reporterTesting "github.com/koh-sh/apcdeploy/internal/reporter/testing"
)

func TestNewInteractiveSelector(t *testing.T) {
	t.Parallel()
	mockPrompter := &promptTesting.MockPrompter{}
	mockReporter := &reporterTesting.MockReporter{}

	selector := NewInteractiveSelector(mockPrompter, mockReporter)

	if selector == nil {
		t.Error("expected non-nil InteractiveSelector")
	}
}

func TestInteractiveSelector_SelectRegion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		providedRegion string
		mockSetup      func(*awsMock.MockAccountClient, *promptTesting.MockPrompter)
		wantRegion     string
		wantErr        bool
		errContains    string
	}{
		{
			name:           "provided region skips prompt",
			providedRegion: "us-east-1",
			mockSetup: func(ac *awsMock.MockAccountClient, p *promptTesting.MockPrompter) {
				// Prompter should not be called
				p.SelectFunc = func(message string, options []string) (string, error) {
					t.Error("Select should not be called when region is provided")
					return "", nil
				}
			},
			wantRegion: "us-east-1",
			wantErr:    false,
		},
		{
			name:           "empty region list returns error",
			providedRegion: "",
			mockSetup: func(ac *awsMock.MockAccountClient, p *promptTesting.MockPrompter) {
				ac.ListRegionsFunc = func(ctx context.Context, params *account.ListRegionsInput, optFns ...func(*account.Options)) (*account.ListRegionsOutput, error) {
					return &account.ListRegionsOutput{Regions: []accountTypes.Region{}}, nil
				}
			},
			wantErr:     true,
			errContains: "no enabled regions found",
		},
		{
			name:           "successful region selection",
			providedRegion: "",
			mockSetup: func(ac *awsMock.MockAccountClient, p *promptTesting.MockPrompter) {
				ac.ListRegionsFunc = func(ctx context.Context, params *account.ListRegionsInput, optFns ...func(*account.Options)) (*account.ListRegionsOutput, error) {
					return &account.ListRegionsOutput{
						Regions: []accountTypes.Region{
							{RegionName: aws.String("us-east-1"), RegionOptStatus: accountTypes.RegionOptStatusEnabled},
							{RegionName: aws.String("us-west-2"), RegionOptStatus: accountTypes.RegionOptStatusEnabled},
						},
					}, nil
				}
				p.SelectFunc = func(message string, options []string) (string, error) {
					return "us-west-2", nil
				}
			},
			wantRegion: "us-west-2",
			wantErr:    false,
		},
		{
			name:           "user cancellation returns error",
			providedRegion: "",
			mockSetup: func(ac *awsMock.MockAccountClient, p *promptTesting.MockPrompter) {
				ac.ListRegionsFunc = func(ctx context.Context, params *account.ListRegionsInput, optFns ...func(*account.Options)) (*account.ListRegionsOutput, error) {
					return &account.ListRegionsOutput{
						Regions: []accountTypes.Region{
							{RegionName: aws.String("us-east-1"), RegionOptStatus: accountTypes.RegionOptStatusEnabled},
						},
					}, nil
				}
				p.SelectFunc = func(message string, options []string) (string, error) {
					return "", huh.ErrUserAborted
				}
			},
			wantErr:     true,
			errContains: "operation cancelled",
		},
		{
			name:           "API error returns error",
			providedRegion: "",
			mockSetup: func(ac *awsMock.MockAccountClient, p *promptTesting.MockPrompter) {
				ac.ListRegionsFunc = func(ctx context.Context, params *account.ListRegionsInput, optFns ...func(*account.Options)) (*account.ListRegionsOutput, error) {
					return nil, errors.New("API error")
				}
			},
			wantErr:     true,
			errContains: "failed to list regions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()

			mockAccountClient := &awsMock.MockAccountClient{}
			mockPrompter := &promptTesting.MockPrompter{}
			mockReporter := &reporterTesting.MockReporter{}

			if tt.mockSetup != nil {
				tt.mockSetup(mockAccountClient, mockPrompter)
			}

			selector := NewInteractiveSelector(mockPrompter, mockReporter)

			region, err := selector.SelectRegion(ctx, mockAccountClient, tt.providedRegion)

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
				if region != tt.wantRegion {
					t.Errorf("expected region %q, got %q", tt.wantRegion, region)
				}
			}
		})
	}
}

func TestInteractiveSelector_SelectApplication(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		providedApp string
		mockSetup   func(*awsMock.MockAppConfigClient, *promptTesting.MockPrompter)
		wantApp     string
		wantErr     bool
		errContains string
	}{
		{
			name:        "provided app skips prompt",
			providedApp: "my-app",
			mockSetup: func(ac *awsMock.MockAppConfigClient, p *promptTesting.MockPrompter) {
				// Prompter should not be called
				p.SelectFunc = func(message string, options []string) (string, error) {
					t.Error("Select should not be called when app is provided")
					return "", nil
				}
			},
			wantApp: "my-app",
			wantErr: false,
		},
		{
			name:        "empty application list returns error",
			providedApp: "",
			mockSetup: func(ac *awsMock.MockAppConfigClient, p *promptTesting.MockPrompter) {
				ac.ListApplicationsFunc = func(ctx context.Context, params *appconfig.ListApplicationsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListApplicationsOutput, error) {
					return &appconfig.ListApplicationsOutput{Items: []appconfigTypes.Application{}}, nil
				}
			},
			wantErr:     true,
			errContains: "no applications found",
		},
		{
			name:        "successful application selection",
			providedApp: "",
			mockSetup: func(ac *awsMock.MockAppConfigClient, p *promptTesting.MockPrompter) {
				ac.ListApplicationsFunc = func(ctx context.Context, params *appconfig.ListApplicationsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListApplicationsOutput, error) {
					return &appconfig.ListApplicationsOutput{
						Items: []appconfigTypes.Application{
							{Name: aws.String("app1")},
							{Name: aws.String("app2")},
						},
					}, nil
				}
				p.SelectFunc = func(message string, options []string) (string, error) {
					return "app2", nil
				}
			},
			wantApp: "app2",
			wantErr: false,
		},
		{
			name:        "sorts applications alphabetically",
			providedApp: "",
			mockSetup: func(ac *awsMock.MockAppConfigClient, p *promptTesting.MockPrompter) {
				ac.ListApplicationsFunc = func(ctx context.Context, params *appconfig.ListApplicationsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListApplicationsOutput, error) {
					return &appconfig.ListApplicationsOutput{
						Items: []appconfigTypes.Application{
							{Name: aws.String("zebra-app")},
							{Name: aws.String("alpha-app")},
							{Name: aws.String("beta-app")},
						},
					}, nil
				}
				p.SelectFunc = func(message string, options []string) (string, error) {
					// Verify order is sorted
					if len(options) != 3 || options[0] != "alpha-app" || options[1] != "beta-app" || options[2] != "zebra-app" {
						t.Errorf("expected sorted options [alpha-app, beta-app, zebra-app], got %v", options)
					}
					return "alpha-app", nil
				}
			},
			wantApp: "alpha-app",
			wantErr: false,
		},
		{
			name:        "user cancellation returns error",
			providedApp: "",
			mockSetup: func(ac *awsMock.MockAppConfigClient, p *promptTesting.MockPrompter) {
				ac.ListApplicationsFunc = func(ctx context.Context, params *appconfig.ListApplicationsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListApplicationsOutput, error) {
					return &appconfig.ListApplicationsOutput{
						Items: []appconfigTypes.Application{
							{Name: aws.String("app1")},
						},
					}, nil
				}
				p.SelectFunc = func(message string, options []string) (string, error) {
					return "", huh.ErrUserAborted
				}
			},
			wantErr:     true,
			errContains: "operation cancelled",
		},
		{
			name:        "API error returns error",
			providedApp: "",
			mockSetup: func(ac *awsMock.MockAppConfigClient, p *promptTesting.MockPrompter) {
				ac.ListApplicationsFunc = func(ctx context.Context, params *appconfig.ListApplicationsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListApplicationsOutput, error) {
					return nil, errors.New("API error")
				}
			},
			wantErr:     true,
			errContains: "failed to list applications",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()

			mockAppConfigClient := &awsMock.MockAppConfigClient{}
			mockPrompter := &promptTesting.MockPrompter{}
			mockReporter := &reporterTesting.MockReporter{}

			if tt.mockSetup != nil {
				tt.mockSetup(mockAppConfigClient, mockPrompter)
			}

			selector := NewInteractiveSelector(mockPrompter, mockReporter)
			client := &awsInternal.Client{
				AppConfig: mockAppConfigClient,
			}

			app, err := selector.SelectApplication(ctx, client, tt.providedApp)

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
				if app != tt.wantApp {
					t.Errorf("expected app %q, got %q", tt.wantApp, app)
				}
			}
		})
	}
}

func TestInteractiveSelector_SelectConfigurationProfile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		appID           string
		providedProfile string
		mockSetup       func(*awsMock.MockAppConfigClient, *promptTesting.MockPrompter)
		wantProfile     string
		wantErr         bool
		errContains     string
	}{
		{
			name:            "provided profile skips prompt",
			appID:           "app-123",
			providedProfile: "my-profile",
			mockSetup: func(ac *awsMock.MockAppConfigClient, p *promptTesting.MockPrompter) {
				// Prompter should not be called
				p.SelectFunc = func(message string, options []string) (string, error) {
					t.Error("Select should not be called when profile is provided")
					return "", nil
				}
			},
			wantProfile: "my-profile",
			wantErr:     false,
		},
		{
			name:            "empty profile list returns error",
			appID:           "app-123",
			providedProfile: "",
			mockSetup: func(ac *awsMock.MockAppConfigClient, p *promptTesting.MockPrompter) {
				ac.ListConfigurationProfilesFunc = func(ctx context.Context, params *appconfig.ListConfigurationProfilesInput, optFns ...func(*appconfig.Options)) (*appconfig.ListConfigurationProfilesOutput, error) {
					return &appconfig.ListConfigurationProfilesOutput{Items: []appconfigTypes.ConfigurationProfileSummary{}}, nil
				}
			},
			wantErr:     true,
			errContains: "no configuration profiles found",
		},
		{
			name:            "successful profile selection",
			appID:           "app-123",
			providedProfile: "",
			mockSetup: func(ac *awsMock.MockAppConfigClient, p *promptTesting.MockPrompter) {
				ac.ListConfigurationProfilesFunc = func(ctx context.Context, params *appconfig.ListConfigurationProfilesInput, optFns ...func(*appconfig.Options)) (*appconfig.ListConfigurationProfilesOutput, error) {
					return &appconfig.ListConfigurationProfilesOutput{
						Items: []appconfigTypes.ConfigurationProfileSummary{
							{Name: aws.String("profile1")},
							{Name: aws.String("profile2")},
						},
					}, nil
				}
				p.SelectFunc = func(message string, options []string) (string, error) {
					return "profile2", nil
				}
			},
			wantProfile: "profile2",
			wantErr:     false,
		},
		{
			name:            "sorts profiles alphabetically",
			appID:           "app-123",
			providedProfile: "",
			mockSetup: func(ac *awsMock.MockAppConfigClient, p *promptTesting.MockPrompter) {
				ac.ListConfigurationProfilesFunc = func(ctx context.Context, params *appconfig.ListConfigurationProfilesInput, optFns ...func(*appconfig.Options)) (*appconfig.ListConfigurationProfilesOutput, error) {
					return &appconfig.ListConfigurationProfilesOutput{
						Items: []appconfigTypes.ConfigurationProfileSummary{
							{Name: aws.String("zebra-profile")},
							{Name: aws.String("alpha-profile")},
							{Name: aws.String("beta-profile")},
						},
					}, nil
				}
				p.SelectFunc = func(message string, options []string) (string, error) {
					// Verify order is sorted
					if len(options) != 3 || options[0] != "alpha-profile" || options[1] != "beta-profile" || options[2] != "zebra-profile" {
						t.Errorf("expected sorted options [alpha-profile, beta-profile, zebra-profile], got %v", options)
					}
					return "alpha-profile", nil
				}
			},
			wantProfile: "alpha-profile",
			wantErr:     false,
		},
		{
			name:            "user cancellation returns error",
			appID:           "app-123",
			providedProfile: "",
			mockSetup: func(ac *awsMock.MockAppConfigClient, p *promptTesting.MockPrompter) {
				ac.ListConfigurationProfilesFunc = func(ctx context.Context, params *appconfig.ListConfigurationProfilesInput, optFns ...func(*appconfig.Options)) (*appconfig.ListConfigurationProfilesOutput, error) {
					return &appconfig.ListConfigurationProfilesOutput{
						Items: []appconfigTypes.ConfigurationProfileSummary{
							{Name: aws.String("profile1")},
						},
					}, nil
				}
				p.SelectFunc = func(message string, options []string) (string, error) {
					return "", huh.ErrUserAborted
				}
			},
			wantErr:     true,
			errContains: "operation cancelled",
		},
		{
			name:            "API error returns error",
			appID:           "app-123",
			providedProfile: "",
			mockSetup: func(ac *awsMock.MockAppConfigClient, p *promptTesting.MockPrompter) {
				ac.ListConfigurationProfilesFunc = func(ctx context.Context, params *appconfig.ListConfigurationProfilesInput, optFns ...func(*appconfig.Options)) (*appconfig.ListConfigurationProfilesOutput, error) {
					return nil, errors.New("API error")
				}
			},
			wantErr:     true,
			errContains: "failed to list configuration profiles",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()

			mockAppConfigClient := &awsMock.MockAppConfigClient{}
			mockPrompter := &promptTesting.MockPrompter{}
			mockReporter := &reporterTesting.MockReporter{}

			if tt.mockSetup != nil {
				tt.mockSetup(mockAppConfigClient, mockPrompter)
			}

			selector := NewInteractiveSelector(mockPrompter, mockReporter)
			client := &awsInternal.Client{
				AppConfig: mockAppConfigClient,
			}

			profile, err := selector.SelectConfigurationProfile(ctx, client, tt.appID, tt.providedProfile)

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
				if profile != tt.wantProfile {
					t.Errorf("expected profile %q, got %q", tt.wantProfile, profile)
				}
			}
		})
	}
}

func TestInteractiveSelector_SelectEnvironment(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		appID       string
		providedEnv string
		mockSetup   func(*awsMock.MockAppConfigClient, *promptTesting.MockPrompter)
		wantEnv     string
		wantErr     bool
		errContains string
	}{
		{
			name:        "provided env skips prompt",
			appID:       "app-123",
			providedEnv: "production",
			mockSetup: func(ac *awsMock.MockAppConfigClient, p *promptTesting.MockPrompter) {
				// Prompter should not be called
				p.SelectFunc = func(message string, options []string) (string, error) {
					t.Error("Select should not be called when env is provided")
					return "", nil
				}
			},
			wantEnv: "production",
			wantErr: false,
		},
		{
			name:        "empty environment list returns error",
			appID:       "app-123",
			providedEnv: "",
			mockSetup: func(ac *awsMock.MockAppConfigClient, p *promptTesting.MockPrompter) {
				ac.ListEnvironmentsFunc = func(ctx context.Context, params *appconfig.ListEnvironmentsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListEnvironmentsOutput, error) {
					return &appconfig.ListEnvironmentsOutput{Items: []appconfigTypes.Environment{}}, nil
				}
			},
			wantErr:     true,
			errContains: "no environments found",
		},
		{
			name:        "successful environment selection",
			appID:       "app-123",
			providedEnv: "",
			mockSetup: func(ac *awsMock.MockAppConfigClient, p *promptTesting.MockPrompter) {
				ac.ListEnvironmentsFunc = func(ctx context.Context, params *appconfig.ListEnvironmentsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListEnvironmentsOutput, error) {
					return &appconfig.ListEnvironmentsOutput{
						Items: []appconfigTypes.Environment{
							{Name: aws.String("dev")},
							{Name: aws.String("prod")},
						},
					}, nil
				}
				p.SelectFunc = func(message string, options []string) (string, error) {
					return "prod", nil
				}
			},
			wantEnv: "prod",
			wantErr: false,
		},
		{
			name:        "sorts environments alphabetically",
			appID:       "app-123",
			providedEnv: "",
			mockSetup: func(ac *awsMock.MockAppConfigClient, p *promptTesting.MockPrompter) {
				ac.ListEnvironmentsFunc = func(ctx context.Context, params *appconfig.ListEnvironmentsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListEnvironmentsOutput, error) {
					return &appconfig.ListEnvironmentsOutput{
						Items: []appconfigTypes.Environment{
							{Name: aws.String("staging")},
							{Name: aws.String("dev")},
							{Name: aws.String("prod")},
						},
					}, nil
				}
				p.SelectFunc = func(message string, options []string) (string, error) {
					// Verify order is sorted
					if len(options) != 3 || options[0] != "dev" || options[1] != "prod" || options[2] != "staging" {
						t.Errorf("expected sorted options [dev, prod, staging], got %v", options)
					}
					return "dev", nil
				}
			},
			wantEnv: "dev",
			wantErr: false,
		},
		{
			name:        "user cancellation returns error",
			appID:       "app-123",
			providedEnv: "",
			mockSetup: func(ac *awsMock.MockAppConfigClient, p *promptTesting.MockPrompter) {
				ac.ListEnvironmentsFunc = func(ctx context.Context, params *appconfig.ListEnvironmentsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListEnvironmentsOutput, error) {
					return &appconfig.ListEnvironmentsOutput{
						Items: []appconfigTypes.Environment{
							{Name: aws.String("dev")},
						},
					}, nil
				}
				p.SelectFunc = func(message string, options []string) (string, error) {
					return "", huh.ErrUserAborted
				}
			},
			wantErr:     true,
			errContains: "operation cancelled",
		},
		{
			name:        "API error returns error",
			appID:       "app-123",
			providedEnv: "",
			mockSetup: func(ac *awsMock.MockAppConfigClient, p *promptTesting.MockPrompter) {
				ac.ListEnvironmentsFunc = func(ctx context.Context, params *appconfig.ListEnvironmentsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListEnvironmentsOutput, error) {
					return nil, errors.New("API error")
				}
			},
			wantErr:     true,
			errContains: "failed to list environments",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()

			mockAppConfigClient := &awsMock.MockAppConfigClient{}
			mockPrompter := &promptTesting.MockPrompter{}
			mockReporter := &reporterTesting.MockReporter{}

			if tt.mockSetup != nil {
				tt.mockSetup(mockAppConfigClient, mockPrompter)
			}

			selector := NewInteractiveSelector(mockPrompter, mockReporter)
			client := &awsInternal.Client{
				AppConfig: mockAppConfigClient,
			}

			env, err := selector.SelectEnvironment(ctx, client, tt.appID, tt.providedEnv)

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
				if env != tt.wantEnv {
					t.Errorf("expected env %q, got %q", tt.wantEnv, env)
				}
			}
		})
	}
}
