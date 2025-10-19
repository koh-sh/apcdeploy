package aws

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/appconfig"
	appconfigTypes "github.com/aws/aws-sdk-go-v2/service/appconfig/types"
	awsMock "github.com/koh-sh/apcdeploy/internal/aws/mock"
)

func TestClient_ListApplications(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		setupMock func(*awsMock.MockAppConfigClient)
		wantErr   bool
	}{
		{
			name: "successful list",
			setupMock: func(m *awsMock.MockAppConfigClient) {
				m.ListApplicationsFunc = func(ctx context.Context, params *appconfig.ListApplicationsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListApplicationsOutput, error) {
					return &appconfig.ListApplicationsOutput{
						Items: []appconfigTypes.Application{
							{Name: aws.String("app1"), Id: aws.String("id1")},
						},
					}, nil
				}
			},
			wantErr: false,
		},
		{
			name: "error from SDK",
			setupMock: func(m *awsMock.MockAppConfigClient) {
				m.ListApplicationsFunc = func(ctx context.Context, params *appconfig.ListApplicationsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListApplicationsOutput, error) {
					return nil, errors.New("API error")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockClient := &awsMock.MockAppConfigClient{}
			tt.setupMock(mockClient)

			client := NewTestClient(mockClient)

			_, err := client.ListApplications(context.Background(), &appconfig.ListApplicationsInput{})

			if (err != nil) != tt.wantErr {
				t.Errorf("ListApplications() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestClient_ListConfigurationProfiles(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		setupMock func(*awsMock.MockAppConfigClient)
		wantErr   bool
	}{
		{
			name: "successful list",
			setupMock: func(m *awsMock.MockAppConfigClient) {
				m.ListConfigurationProfilesFunc = func(ctx context.Context, params *appconfig.ListConfigurationProfilesInput, optFns ...func(*appconfig.Options)) (*appconfig.ListConfigurationProfilesOutput, error) {
					return &appconfig.ListConfigurationProfilesOutput{
						Items: []appconfigTypes.ConfigurationProfileSummary{
							{Name: aws.String("profile1"), Id: aws.String("id1")},
						},
					}, nil
				}
			},
			wantErr: false,
		},
		{
			name: "error from SDK",
			setupMock: func(m *awsMock.MockAppConfigClient) {
				m.ListConfigurationProfilesFunc = func(ctx context.Context, params *appconfig.ListConfigurationProfilesInput, optFns ...func(*appconfig.Options)) (*appconfig.ListConfigurationProfilesOutput, error) {
					return nil, errors.New("API error")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockClient := &awsMock.MockAppConfigClient{}
			tt.setupMock(mockClient)

			client := NewTestClient(mockClient)

			_, err := client.ListConfigurationProfiles(context.Background(), &appconfig.ListConfigurationProfilesInput{
				ApplicationId: aws.String("app-id"),
			})

			if (err != nil) != tt.wantErr {
				t.Errorf("ListConfigurationProfiles() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestClient_ListEnvironments(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		setupMock func(*awsMock.MockAppConfigClient)
		wantErr   bool
	}{
		{
			name: "successful list",
			setupMock: func(m *awsMock.MockAppConfigClient) {
				m.ListEnvironmentsFunc = func(ctx context.Context, params *appconfig.ListEnvironmentsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListEnvironmentsOutput, error) {
					return &appconfig.ListEnvironmentsOutput{
						Items: []appconfigTypes.Environment{
							{Name: aws.String("env1"), Id: aws.String("id1")},
						},
					}, nil
				}
			},
			wantErr: false,
		},
		{
			name: "error from SDK",
			setupMock: func(m *awsMock.MockAppConfigClient) {
				m.ListEnvironmentsFunc = func(ctx context.Context, params *appconfig.ListEnvironmentsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListEnvironmentsOutput, error) {
					return nil, errors.New("API error")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockClient := &awsMock.MockAppConfigClient{}
			tt.setupMock(mockClient)

			client := NewTestClient(mockClient)

			_, err := client.ListEnvironments(context.Background(), &appconfig.ListEnvironmentsInput{
				ApplicationId: aws.String("app-id"),
			})

			if (err != nil) != tt.wantErr {
				t.Errorf("ListEnvironments() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestClient_ListDeploymentStrategies(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		setupMock func(*awsMock.MockAppConfigClient)
		wantErr   bool
	}{
		{
			name: "successful list",
			setupMock: func(m *awsMock.MockAppConfigClient) {
				m.ListDeploymentStrategiesFunc = func(ctx context.Context, params *appconfig.ListDeploymentStrategiesInput, optFns ...func(*appconfig.Options)) (*appconfig.ListDeploymentStrategiesOutput, error) {
					return &appconfig.ListDeploymentStrategiesOutput{
						Items: []appconfigTypes.DeploymentStrategy{
							{Name: aws.String("strategy1"), Id: aws.String("id1")},
						},
					}, nil
				}
			},
			wantErr: false,
		},
		{
			name: "error from SDK",
			setupMock: func(m *awsMock.MockAppConfigClient) {
				m.ListDeploymentStrategiesFunc = func(ctx context.Context, params *appconfig.ListDeploymentStrategiesInput, optFns ...func(*appconfig.Options)) (*appconfig.ListDeploymentStrategiesOutput, error) {
					return nil, errors.New("API error")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockClient := &awsMock.MockAppConfigClient{}
			tt.setupMock(mockClient)

			client := NewTestClient(mockClient)

			_, err := client.ListDeploymentStrategies(context.Background(), &appconfig.ListDeploymentStrategiesInput{})

			if (err != nil) != tt.wantErr {
				t.Errorf("ListDeploymentStrategies() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestClient_ListHostedConfigurationVersions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		setupMock func(*awsMock.MockAppConfigClient)
		wantErr   bool
	}{
		{
			name: "successful list",
			setupMock: func(m *awsMock.MockAppConfigClient) {
				m.ListHostedConfigurationVersionsFunc = func(ctx context.Context, params *appconfig.ListHostedConfigurationVersionsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListHostedConfigurationVersionsOutput, error) {
					return &appconfig.ListHostedConfigurationVersionsOutput{
						Items: []appconfigTypes.HostedConfigurationVersionSummary{
							{VersionNumber: 1},
						},
					}, nil
				}
			},
			wantErr: false,
		},
		{
			name: "error from SDK",
			setupMock: func(m *awsMock.MockAppConfigClient) {
				m.ListHostedConfigurationVersionsFunc = func(ctx context.Context, params *appconfig.ListHostedConfigurationVersionsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListHostedConfigurationVersionsOutput, error) {
					return nil, errors.New("API error")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockClient := &awsMock.MockAppConfigClient{}
			tt.setupMock(mockClient)

			client := NewTestClient(mockClient)

			_, err := client.ListHostedConfigurationVersions(context.Background(), &appconfig.ListHostedConfigurationVersionsInput{
				ApplicationId:          aws.String("app-id"),
				ConfigurationProfileId: aws.String("profile-id"),
			})

			if (err != nil) != tt.wantErr {
				t.Errorf("ListHostedConfigurationVersions() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestClient_ListDeployments(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		setupMock func(*awsMock.MockAppConfigClient)
		wantErr   bool
	}{
		{
			name: "successful list",
			setupMock: func(m *awsMock.MockAppConfigClient) {
				m.ListDeploymentsFunc = func(ctx context.Context, params *appconfig.ListDeploymentsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListDeploymentsOutput, error) {
					return &appconfig.ListDeploymentsOutput{
						Items: []appconfigTypes.DeploymentSummary{
							{DeploymentNumber: 1},
						},
					}, nil
				}
			},
			wantErr: false,
		},
		{
			name: "error from SDK",
			setupMock: func(m *awsMock.MockAppConfigClient) {
				m.ListDeploymentsFunc = func(ctx context.Context, params *appconfig.ListDeploymentsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListDeploymentsOutput, error) {
					return nil, errors.New("API error")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockClient := &awsMock.MockAppConfigClient{}
			tt.setupMock(mockClient)

			client := NewTestClient(mockClient)

			_, err := client.ListDeployments(context.Background(), &appconfig.ListDeploymentsInput{
				ApplicationId: aws.String("app-id"),
				EnvironmentId: aws.String("env-id"),
			})

			if (err != nil) != tt.wantErr {
				t.Errorf("ListDeployments() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestClient_GetConfigurationProfile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		setupMock func(*awsMock.MockAppConfigClient)
		wantErr   bool
	}{
		{
			name: "successful get",
			setupMock: func(m *awsMock.MockAppConfigClient) {
				m.GetConfigurationProfileFunc = func(ctx context.Context, params *appconfig.GetConfigurationProfileInput, optFns ...func(*appconfig.Options)) (*appconfig.GetConfigurationProfileOutput, error) {
					return &appconfig.GetConfigurationProfileOutput{
						Name: aws.String("profile1"),
						Id:   aws.String("id1"),
					}, nil
				}
			},
			wantErr: false,
		},
		{
			name: "error from SDK",
			setupMock: func(m *awsMock.MockAppConfigClient) {
				m.GetConfigurationProfileFunc = func(ctx context.Context, params *appconfig.GetConfigurationProfileInput, optFns ...func(*appconfig.Options)) (*appconfig.GetConfigurationProfileOutput, error) {
					return nil, errors.New("API error")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockClient := &awsMock.MockAppConfigClient{}
			tt.setupMock(mockClient)

			client := NewTestClient(mockClient)

			_, err := client.GetConfigurationProfile(context.Background(), &appconfig.GetConfigurationProfileInput{
				ApplicationId:          aws.String("app-id"),
				ConfigurationProfileId: aws.String("profile-id"),
			})

			if (err != nil) != tt.wantErr {
				t.Errorf("GetConfigurationProfile() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestClient_GetHostedConfigurationVersion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		setupMock func(*awsMock.MockAppConfigClient)
		wantErr   bool
	}{
		{
			name: "successful get",
			setupMock: func(m *awsMock.MockAppConfigClient) {
				m.GetHostedConfigurationVersionFunc = func(ctx context.Context, params *appconfig.GetHostedConfigurationVersionInput, optFns ...func(*appconfig.Options)) (*appconfig.GetHostedConfigurationVersionOutput, error) {
					return &appconfig.GetHostedConfigurationVersionOutput{
						VersionNumber: 1,
					}, nil
				}
			},
			wantErr: false,
		},
		{
			name: "error from SDK",
			setupMock: func(m *awsMock.MockAppConfigClient) {
				m.GetHostedConfigurationVersionFunc = func(ctx context.Context, params *appconfig.GetHostedConfigurationVersionInput, optFns ...func(*appconfig.Options)) (*appconfig.GetHostedConfigurationVersionOutput, error) {
					return nil, errors.New("API error")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockClient := &awsMock.MockAppConfigClient{}
			tt.setupMock(mockClient)

			client := NewTestClient(mockClient)

			_, err := client.GetHostedConfigurationVersion(context.Background(), &appconfig.GetHostedConfigurationVersionInput{
				ApplicationId:          aws.String("app-id"),
				ConfigurationProfileId: aws.String("profile-id"),
				VersionNumber:          aws.Int32(1),
			})

			if (err != nil) != tt.wantErr {
				t.Errorf("GetHostedConfigurationVersion() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestClient_GetDeployment(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		setupMock func(*awsMock.MockAppConfigClient)
		wantErr   bool
	}{
		{
			name: "successful get",
			setupMock: func(m *awsMock.MockAppConfigClient) {
				m.GetDeploymentFunc = func(ctx context.Context, params *appconfig.GetDeploymentInput, optFns ...func(*appconfig.Options)) (*appconfig.GetDeploymentOutput, error) {
					return &appconfig.GetDeploymentOutput{
						DeploymentNumber: 1,
					}, nil
				}
			},
			wantErr: false,
		},
		{
			name: "error from SDK",
			setupMock: func(m *awsMock.MockAppConfigClient) {
				m.GetDeploymentFunc = func(ctx context.Context, params *appconfig.GetDeploymentInput, optFns ...func(*appconfig.Options)) (*appconfig.GetDeploymentOutput, error) {
					return nil, errors.New("API error")
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockClient := &awsMock.MockAppConfigClient{}
			tt.setupMock(mockClient)

			client := NewTestClient(mockClient)

			_, err := client.GetDeployment(context.Background(), &appconfig.GetDeploymentInput{
				ApplicationId:    aws.String("app-id"),
				EnvironmentId:    aws.String("env-id"),
				DeploymentNumber: aws.Int32(1),
			})

			if (err != nil) != tt.wantErr {
				t.Errorf("GetDeployment() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
