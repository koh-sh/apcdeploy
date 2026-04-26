package aws

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/appconfig"
	awsMock "github.com/koh-sh/apcdeploy/internal/aws/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
