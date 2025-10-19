package aws

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/appconfig"
	appconfigTypes "github.com/aws/aws-sdk-go-v2/service/appconfig/types"
	awsMock "github.com/koh-sh/apcdeploy/internal/aws/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_ListApplications(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		setupMock   func(*awsMock.MockAppConfigClient)
		checkResult func(*testing.T, *appconfig.ListApplicationsOutput, error)
	}{
		{
			name: "successful list with data validation",
			setupMock: func(m *awsMock.MockAppConfigClient) {
				m.ListApplicationsFunc = func(ctx context.Context, params *appconfig.ListApplicationsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListApplicationsOutput, error) {
					return &appconfig.ListApplicationsOutput{
						Items: []appconfigTypes.Application{
							{Name: aws.String("app1"), Id: aws.String("id1")},
							{Name: aws.String("app2"), Id: aws.String("id2")},
						},
					}, nil
				}
			},
			checkResult: func(t *testing.T, output *appconfig.ListApplicationsOutput, err error) {
				require.NoError(t, err)
				require.NotNil(t, output)
				assert.Len(t, output.Items, 2)
				assert.Equal(t, "app1", *output.Items[0].Name)
				assert.Equal(t, "id1", *output.Items[0].Id)
				assert.Equal(t, "app2", *output.Items[1].Name)
				assert.Equal(t, "id2", *output.Items[1].Id)
			},
		},
		{
			name: "error from SDK",
			setupMock: func(m *awsMock.MockAppConfigClient) {
				m.ListApplicationsFunc = func(ctx context.Context, params *appconfig.ListApplicationsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListApplicationsOutput, error) {
					return nil, errors.New("API error")
				}
			},
			checkResult: func(t *testing.T, output *appconfig.ListApplicationsOutput, err error) {
				require.Error(t, err)
				assert.EqualError(t, err, "API error")
				assert.Nil(t, output)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockClient := &awsMock.MockAppConfigClient{}
			tt.setupMock(mockClient)

			client := NewTestClient(mockClient)

			output, err := client.ListApplications(context.Background(), &appconfig.ListApplicationsInput{})

			tt.checkResult(t, output, err)
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
		name        string
		setupMock   func(*awsMock.MockAppConfigClient)
		checkResult func(*testing.T, *appconfig.GetConfigurationProfileOutput, error)
	}{
		{
			name: "successful get with data validation",
			setupMock: func(m *awsMock.MockAppConfigClient) {
				m.GetConfigurationProfileFunc = func(ctx context.Context, params *appconfig.GetConfigurationProfileInput, optFns ...func(*appconfig.Options)) (*appconfig.GetConfigurationProfileOutput, error) {
					return &appconfig.GetConfigurationProfileOutput{
						Name:          aws.String("profile1"),
						Id:            aws.String("id1"),
						ApplicationId: aws.String("app-id"),
						Type:          aws.String("AWS.AppConfig.FeatureFlags"),
					}, nil
				}
			},
			checkResult: func(t *testing.T, output *appconfig.GetConfigurationProfileOutput, err error) {
				require.NoError(t, err)
				require.NotNil(t, output)
				assert.Equal(t, "profile1", *output.Name)
				assert.Equal(t, "id1", *output.Id)
				assert.Equal(t, "app-id", *output.ApplicationId)
				assert.Equal(t, "AWS.AppConfig.FeatureFlags", *output.Type)
			},
		},
		{
			name: "error from SDK",
			setupMock: func(m *awsMock.MockAppConfigClient) {
				m.GetConfigurationProfileFunc = func(ctx context.Context, params *appconfig.GetConfigurationProfileInput, optFns ...func(*appconfig.Options)) (*appconfig.GetConfigurationProfileOutput, error) {
					return nil, errors.New("API error")
				}
			},
			checkResult: func(t *testing.T, output *appconfig.GetConfigurationProfileOutput, err error) {
				require.Error(t, err)
				assert.EqualError(t, err, "API error")
				assert.Nil(t, output)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockClient := &awsMock.MockAppConfigClient{}
			tt.setupMock(mockClient)

			client := NewTestClient(mockClient)

			output, err := client.GetConfigurationProfile(context.Background(), &appconfig.GetConfigurationProfileInput{
				ApplicationId:          aws.String("app-id"),
				ConfigurationProfileId: aws.String("profile-id"),
			})

			tt.checkResult(t, output, err)
		})
	}
}

func TestClient_GetHostedConfigurationVersion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		setupMock   func(*awsMock.MockAppConfigClient)
		checkResult func(*testing.T, *appconfig.GetHostedConfigurationVersionOutput, error)
	}{
		{
			name: "successful get with data validation",
			setupMock: func(m *awsMock.MockAppConfigClient) {
				m.GetHostedConfigurationVersionFunc = func(ctx context.Context, params *appconfig.GetHostedConfigurationVersionInput, optFns ...func(*appconfig.Options)) (*appconfig.GetHostedConfigurationVersionOutput, error) {
					return &appconfig.GetHostedConfigurationVersionOutput{
						VersionNumber:          1,
						ApplicationId:          aws.String("app-id"),
						ConfigurationProfileId: aws.String("profile-id"),
						ContentType:            aws.String("application/json"),
						Content:                []byte(`{"key":"value"}`),
					}, nil
				}
			},
			checkResult: func(t *testing.T, output *appconfig.GetHostedConfigurationVersionOutput, err error) {
				require.NoError(t, err)
				require.NotNil(t, output)
				assert.Equal(t, int32(1), output.VersionNumber)
				assert.Equal(t, "app-id", *output.ApplicationId)
				assert.Equal(t, "profile-id", *output.ConfigurationProfileId)
				assert.Equal(t, "application/json", *output.ContentType)
				assert.Equal(t, []byte(`{"key":"value"}`), output.Content)
			},
		},
		{
			name: "error from SDK",
			setupMock: func(m *awsMock.MockAppConfigClient) {
				m.GetHostedConfigurationVersionFunc = func(ctx context.Context, params *appconfig.GetHostedConfigurationVersionInput, optFns ...func(*appconfig.Options)) (*appconfig.GetHostedConfigurationVersionOutput, error) {
					return nil, errors.New("API error")
				}
			},
			checkResult: func(t *testing.T, output *appconfig.GetHostedConfigurationVersionOutput, err error) {
				require.Error(t, err)
				assert.EqualError(t, err, "API error")
				assert.Nil(t, output)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockClient := &awsMock.MockAppConfigClient{}
			tt.setupMock(mockClient)

			client := NewTestClient(mockClient)

			output, err := client.GetHostedConfigurationVersion(context.Background(), &appconfig.GetHostedConfigurationVersionInput{
				ApplicationId:          aws.String("app-id"),
				ConfigurationProfileId: aws.String("profile-id"),
				VersionNumber:          aws.Int32(1),
			})

			tt.checkResult(t, output, err)
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
