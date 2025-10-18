package aws

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/appconfig"
	"github.com/aws/aws-sdk-go-v2/service/appconfig/types"
	"github.com/koh-sh/apcdeploy/internal/aws/mock"
)

func TestListAllApplications(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		setupMock func() *mock.MockAppConfigClient
		wantCount int
		wantErr   bool
	}{
		{
			name: "single page",
			setupMock: func() *mock.MockAppConfigClient {
				return &mock.MockAppConfigClient{
					ListApplicationsFunc: func(ctx context.Context, params *appconfig.ListApplicationsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListApplicationsOutput, error) {
						return &appconfig.ListApplicationsOutput{
							Items: []types.Application{
								{Id: aws.String("app-1"), Name: aws.String("app-1")},
								{Id: aws.String("app-2"), Name: aws.String("app-2")},
							},
						}, nil
					},
				}
			},
			wantCount: 2,
			wantErr:   false,
		},
		{
			name: "multiple pages",
			setupMock: func() *mock.MockAppConfigClient {
				callCount := 0
				return &mock.MockAppConfigClient{
					ListApplicationsFunc: func(ctx context.Context, params *appconfig.ListApplicationsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListApplicationsOutput, error) {
						callCount++
						if callCount == 1 {
							return &appconfig.ListApplicationsOutput{
								Items: []types.Application{
									{Id: aws.String("app-1"), Name: aws.String("app-1")},
									{Id: aws.String("app-2"), Name: aws.String("app-2")},
								},
								NextToken: aws.String("page2"),
							}, nil
						}
						return &appconfig.ListApplicationsOutput{
							Items: []types.Application{
								{Id: aws.String("app-3"), Name: aws.String("app-3")},
							},
						}, nil
					},
				}
			},
			wantCount: 3,
			wantErr:   false,
		},
		{
			name: "API error",
			setupMock: func() *mock.MockAppConfigClient {
				return &mock.MockAppConfigClient{
					ListApplicationsFunc: func(ctx context.Context, params *appconfig.ListApplicationsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListApplicationsOutput, error) {
						return nil, errors.New("API error")
					},
				}
			},
			wantErr: true,
		},
		{
			name: "empty result",
			setupMock: func() *mock.MockAppConfigClient {
				return &mock.MockAppConfigClient{
					ListApplicationsFunc: func(ctx context.Context, params *appconfig.ListApplicationsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListApplicationsOutput, error) {
						return &appconfig.ListApplicationsOutput{
							Items: []types.Application{},
						}, nil
					},
				}
			},
			wantCount: 0,
			wantErr:   false,
		},
		{
			name: "pagination token is correctly passed",
			setupMock: func() *mock.MockAppConfigClient {
				callCount := 0
				return &mock.MockAppConfigClient{
					ListApplicationsFunc: func(ctx context.Context, params *appconfig.ListApplicationsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListApplicationsOutput, error) {
						callCount++
						if callCount == 1 {
							// First call should have nil NextToken
							if params.NextToken != nil {
								t.Errorf("first call should have nil NextToken, got %v", *params.NextToken)
							}
							return &appconfig.ListApplicationsOutput{
								Items: []types.Application{
									{Id: aws.String("app-1"), Name: aws.String("app-1")},
								},
								NextToken: aws.String("token123"),
							}, nil
						}
						// Second call should pass the token
						if params.NextToken == nil {
							t.Error("second call should have NextToken")
						} else if *params.NextToken != "token123" {
							t.Errorf("second call NextToken = %v, want token123", *params.NextToken)
						}
						return &appconfig.ListApplicationsOutput{
							Items: []types.Application{
								{Id: aws.String("app-2"), Name: aws.String("app-2")},
							},
						}, nil
					},
				}
			},
			wantCount: 2,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			client := &Client{
				appConfig: tt.setupMock(),
			}

			ctx := context.Background()
			apps, err := client.ListAllApplications(ctx)

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

			if len(apps) != tt.wantCount {
				t.Errorf("got %d applications, want %d", len(apps), tt.wantCount)
			}
		})
	}
}

func TestListAllConfigurationProfiles(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		appID     string
		setupMock func() *mock.MockAppConfigClient
		wantCount int
		wantErr   bool
	}{
		{
			name:  "single page",
			appID: "app-123",
			setupMock: func() *mock.MockAppConfigClient {
				return &mock.MockAppConfigClient{
					ListConfigurationProfilesFunc: func(ctx context.Context, params *appconfig.ListConfigurationProfilesInput, optFns ...func(*appconfig.Options)) (*appconfig.ListConfigurationProfilesOutput, error) {
						return &appconfig.ListConfigurationProfilesOutput{
							Items: []types.ConfigurationProfileSummary{
								{Id: aws.String("prof-1"), Name: aws.String("profile-1")},
								{Id: aws.String("prof-2"), Name: aws.String("profile-2")},
							},
						}, nil
					},
				}
			},
			wantCount: 2,
			wantErr:   false,
		},
		{
			name:  "multiple pages",
			appID: "app-123",
			setupMock: func() *mock.MockAppConfigClient {
				callCount := 0
				return &mock.MockAppConfigClient{
					ListConfigurationProfilesFunc: func(ctx context.Context, params *appconfig.ListConfigurationProfilesInput, optFns ...func(*appconfig.Options)) (*appconfig.ListConfigurationProfilesOutput, error) {
						callCount++
						if callCount == 1 {
							return &appconfig.ListConfigurationProfilesOutput{
								Items: []types.ConfigurationProfileSummary{
									{Id: aws.String("prof-1"), Name: aws.String("profile-1")},
								},
								NextToken: aws.String("page2"),
							}, nil
						}
						return &appconfig.ListConfigurationProfilesOutput{
							Items: []types.ConfigurationProfileSummary{
								{Id: aws.String("prof-2"), Name: aws.String("profile-2")},
								{Id: aws.String("prof-3"), Name: aws.String("profile-3")},
							},
						}, nil
					},
				}
			},
			wantCount: 3,
			wantErr:   false,
		},
		{
			name:  "API error",
			appID: "app-123",
			setupMock: func() *mock.MockAppConfigClient {
				return &mock.MockAppConfigClient{
					ListConfigurationProfilesFunc: func(ctx context.Context, params *appconfig.ListConfigurationProfilesInput, optFns ...func(*appconfig.Options)) (*appconfig.ListConfigurationProfilesOutput, error) {
						return nil, errors.New("API error")
					},
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			client := &Client{
				appConfig: tt.setupMock(),
			}

			ctx := context.Background()
			profiles, err := client.ListAllConfigurationProfiles(ctx, tt.appID)

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

			if len(profiles) != tt.wantCount {
				t.Errorf("got %d profiles, want %d", len(profiles), tt.wantCount)
			}
		})
	}
}

func TestListAllEnvironments(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		appID     string
		setupMock func() *mock.MockAppConfigClient
		wantCount int
		wantErr   bool
	}{
		{
			name:  "single page",
			appID: "app-123",
			setupMock: func() *mock.MockAppConfigClient {
				return &mock.MockAppConfigClient{
					ListEnvironmentsFunc: func(ctx context.Context, params *appconfig.ListEnvironmentsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListEnvironmentsOutput, error) {
						return &appconfig.ListEnvironmentsOutput{
							Items: []types.Environment{
								{Id: aws.String("env-1"), Name: aws.String("dev")},
								{Id: aws.String("env-2"), Name: aws.String("prod")},
							},
						}, nil
					},
				}
			},
			wantCount: 2,
			wantErr:   false,
		},
		{
			name:  "multiple pages",
			appID: "app-123",
			setupMock: func() *mock.MockAppConfigClient {
				callCount := 0
				return &mock.MockAppConfigClient{
					ListEnvironmentsFunc: func(ctx context.Context, params *appconfig.ListEnvironmentsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListEnvironmentsOutput, error) {
						callCount++
						if callCount == 1 {
							return &appconfig.ListEnvironmentsOutput{
								Items: []types.Environment{
									{Id: aws.String("env-1"), Name: aws.String("dev")},
								},
								NextToken: aws.String("page2"),
							}, nil
						}
						return &appconfig.ListEnvironmentsOutput{
							Items: []types.Environment{
								{Id: aws.String("env-2"), Name: aws.String("prod")},
							},
						}, nil
					},
				}
			},
			wantCount: 2,
			wantErr:   false,
		},
		{
			name:  "API error",
			appID: "app-123",
			setupMock: func() *mock.MockAppConfigClient {
				return &mock.MockAppConfigClient{
					ListEnvironmentsFunc: func(ctx context.Context, params *appconfig.ListEnvironmentsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListEnvironmentsOutput, error) {
						return nil, errors.New("API error")
					},
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			client := &Client{
				appConfig: tt.setupMock(),
			}

			ctx := context.Background()
			envs, err := client.ListAllEnvironments(ctx, tt.appID)

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

			if len(envs) != tt.wantCount {
				t.Errorf("got %d environments, want %d", len(envs), tt.wantCount)
			}
		})
	}
}

func TestListAllDeploymentStrategies(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		setupMock func() *mock.MockAppConfigClient
		wantCount int
		wantErr   bool
	}{
		{
			name: "single page",
			setupMock: func() *mock.MockAppConfigClient {
				return &mock.MockAppConfigClient{
					ListDeploymentStrategiesFunc: func(ctx context.Context, params *appconfig.ListDeploymentStrategiesInput, optFns ...func(*appconfig.Options)) (*appconfig.ListDeploymentStrategiesOutput, error) {
						return &appconfig.ListDeploymentStrategiesOutput{
							Items: []types.DeploymentStrategy{
								{Id: aws.String("start-1"), Name: aws.String("AllAtOnce")},
								{Id: aws.String("start-2"), Name: aws.String("Linear50PercentEvery30Seconds")},
							},
						}, nil
					},
				}
			},
			wantCount: 2,
			wantErr:   false,
		},
		{
			name: "multiple pages",
			setupMock: func() *mock.MockAppConfigClient {
				callCount := 0
				return &mock.MockAppConfigClient{
					ListDeploymentStrategiesFunc: func(ctx context.Context, params *appconfig.ListDeploymentStrategiesInput, optFns ...func(*appconfig.Options)) (*appconfig.ListDeploymentStrategiesOutput, error) {
						callCount++
						if callCount == 1 {
							return &appconfig.ListDeploymentStrategiesOutput{
								Items: []types.DeploymentStrategy{
									{Id: aws.String("start-1"), Name: aws.String("AllAtOnce")},
								},
								NextToken: aws.String("page2"),
							}, nil
						}
						return &appconfig.ListDeploymentStrategiesOutput{
							Items: []types.DeploymentStrategy{
								{Id: aws.String("start-2"), Name: aws.String("Custom")},
							},
						}, nil
					},
				}
			},
			wantCount: 2,
			wantErr:   false,
		},
		{
			name: "API error",
			setupMock: func() *mock.MockAppConfigClient {
				return &mock.MockAppConfigClient{
					ListDeploymentStrategiesFunc: func(ctx context.Context, params *appconfig.ListDeploymentStrategiesInput, optFns ...func(*appconfig.Options)) (*appconfig.ListDeploymentStrategiesOutput, error) {
						return nil, errors.New("API error")
					},
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			client := &Client{
				appConfig: tt.setupMock(),
			}

			ctx := context.Background()
			strategies, err := client.ListAllDeploymentStrategies(ctx)

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

			if len(strategies) != tt.wantCount {
				t.Errorf("got %d strategies, want %d", len(strategies), tt.wantCount)
			}
		})
	}
}

func TestListAllDeployments(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		appID     string
		envID     string
		setupMock func() *mock.MockAppConfigClient
		wantCount int
		wantErr   bool
	}{
		{
			name:  "single page",
			appID: "app-123",
			envID: "env-456",
			setupMock: func() *mock.MockAppConfigClient {
				return &mock.MockAppConfigClient{
					ListDeploymentsFunc: func(ctx context.Context, params *appconfig.ListDeploymentsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListDeploymentsOutput, error) {
						return &appconfig.ListDeploymentsOutput{
							Items: []types.DeploymentSummary{
								{DeploymentNumber: 1},
								{DeploymentNumber: 2},
							},
						}, nil
					},
				}
			},
			wantCount: 2,
			wantErr:   false,
		},
		{
			name:  "multiple pages",
			appID: "app-123",
			envID: "env-456",
			setupMock: func() *mock.MockAppConfigClient {
				callCount := 0
				return &mock.MockAppConfigClient{
					ListDeploymentsFunc: func(ctx context.Context, params *appconfig.ListDeploymentsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListDeploymentsOutput, error) {
						callCount++
						if callCount == 1 {
							return &appconfig.ListDeploymentsOutput{
								Items: []types.DeploymentSummary{
									{DeploymentNumber: 1},
								},
								NextToken: aws.String("page2"),
							}, nil
						}
						return &appconfig.ListDeploymentsOutput{
							Items: []types.DeploymentSummary{
								{DeploymentNumber: 2},
								{DeploymentNumber: 3},
							},
						}, nil
					},
				}
			},
			wantCount: 3,
			wantErr:   false,
		},
		{
			name:  "API error",
			appID: "app-123",
			envID: "env-456",
			setupMock: func() *mock.MockAppConfigClient {
				return &mock.MockAppConfigClient{
					ListDeploymentsFunc: func(ctx context.Context, params *appconfig.ListDeploymentsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListDeploymentsOutput, error) {
						return nil, errors.New("API error")
					},
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			client := &Client{
				appConfig: tt.setupMock(),
			}

			ctx := context.Background()
			deployments, err := client.ListAllDeployments(ctx, tt.appID, tt.envID)

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

			if len(deployments) != tt.wantCount {
				t.Errorf("got %d deployments, want %d", len(deployments), tt.wantCount)
			}
		})
	}
}

func TestListAllHostedConfigurationVersions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		appID     string
		profileID string
		setupMock func() *mock.MockAppConfigClient
		wantCount int
		wantErr   bool
	}{
		{
			name:      "single page",
			appID:     "app-123",
			profileID: "prof-456",
			setupMock: func() *mock.MockAppConfigClient {
				return &mock.MockAppConfigClient{
					ListHostedConfigurationVersionsFunc: func(ctx context.Context, params *appconfig.ListHostedConfigurationVersionsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListHostedConfigurationVersionsOutput, error) {
						return &appconfig.ListHostedConfigurationVersionsOutput{
							Items: []types.HostedConfigurationVersionSummary{
								{VersionNumber: 1},
								{VersionNumber: 2},
							},
						}, nil
					},
				}
			},
			wantCount: 2,
			wantErr:   false,
		},
		{
			name:      "multiple pages",
			appID:     "app-123",
			profileID: "prof-456",
			setupMock: func() *mock.MockAppConfigClient {
				callCount := 0
				return &mock.MockAppConfigClient{
					ListHostedConfigurationVersionsFunc: func(ctx context.Context, params *appconfig.ListHostedConfigurationVersionsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListHostedConfigurationVersionsOutput, error) {
						callCount++
						if callCount == 1 {
							return &appconfig.ListHostedConfigurationVersionsOutput{
								Items: []types.HostedConfigurationVersionSummary{
									{VersionNumber: 1},
								},
								NextToken: aws.String("page2"),
							}, nil
						}
						return &appconfig.ListHostedConfigurationVersionsOutput{
							Items: []types.HostedConfigurationVersionSummary{
								{VersionNumber: 2},
							},
						}, nil
					},
				}
			},
			wantCount: 2,
			wantErr:   false,
		},
		{
			name:      "API error",
			appID:     "app-123",
			profileID: "prof-456",
			setupMock: func() *mock.MockAppConfigClient {
				return &mock.MockAppConfigClient{
					ListHostedConfigurationVersionsFunc: func(ctx context.Context, params *appconfig.ListHostedConfigurationVersionsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListHostedConfigurationVersionsOutput, error) {
						return nil, errors.New("API error")
					},
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			client := &Client{
				appConfig: tt.setupMock(),
			}

			ctx := context.Background()
			versions, err := client.ListAllHostedConfigurationVersions(ctx, tt.appID, tt.profileID)

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

			if len(versions) != tt.wantCount {
				t.Errorf("got %d versions, want %d", len(versions), tt.wantCount)
			}
		})
	}
}
