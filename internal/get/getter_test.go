package get

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/appconfig"
	"github.com/aws/aws-sdk-go-v2/service/appconfig/types"
	"github.com/aws/aws-sdk-go-v2/service/appconfigdata"
	awsInternal "github.com/koh-sh/apcdeploy/internal/aws"
	"github.com/koh-sh/apcdeploy/internal/aws/mock"
	"github.com/koh-sh/apcdeploy/internal/config"
)

func TestNew(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		region  string
		wantErr bool
	}{
		{
			name:    "valid region",
			region:  "us-east-1",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			cfg := &config.Config{
				Application:          "test-app",
				ConfigurationProfile: "test-profile",
				Environment:          "test-env",
				Region:               tt.region,
				DataFile:             "data.json",
			}

			getter, err := New(ctx, cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && getter == nil {
				t.Error("Expected getter to be non-nil")
			}
			if !tt.wantErr && getter.awsClient == nil {
				t.Error("Expected awsClient to be non-nil")
			}
		})
	}
}

func TestNewWithClient(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Application:          "test-app",
		ConfigurationProfile: "test-profile",
		Environment:          "test-env",
	}

	mockAppConfigClient := &mock.MockAppConfigClient{}
	mockAppConfigDataClient := &mock.MockAppConfigDataClient{}

	awsClient := awsInternal.NewTestClientWithData(mockAppConfigClient, mockAppConfigDataClient)

	getter := NewWithClient(cfg, awsClient)

	if getter.cfg != cfg {
		t.Error("expected getter to have the provided config")
	}

	if getter.awsClient != awsClient {
		t.Error("expected getter to have the provided AWS client")
	}
}

func TestResolveResourcesSuccess(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Application:          "test-app",
		ConfigurationProfile: "test-profile",
		Environment:          "test-env",
	}

	mockAppConfigClient := &mock.MockAppConfigClient{
		ListApplicationsFunc: func(ctx context.Context, params *appconfig.ListApplicationsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListApplicationsOutput, error) {
			return &appconfig.ListApplicationsOutput{
				Items: []types.Application{
					{
						Id:   aws.String("app-123"),
						Name: aws.String("test-app"),
					},
				},
			}, nil
		},
		ListConfigurationProfilesFunc: func(ctx context.Context, params *appconfig.ListConfigurationProfilesInput, optFns ...func(*appconfig.Options)) (*appconfig.ListConfigurationProfilesOutput, error) {
			return &appconfig.ListConfigurationProfilesOutput{
				Items: []types.ConfigurationProfileSummary{
					{
						Id:   aws.String("profile-123"),
						Name: aws.String("test-profile"),
					},
				},
			}, nil
		},
		GetConfigurationProfileFunc: func(ctx context.Context, params *appconfig.GetConfigurationProfileInput, optFns ...func(*appconfig.Options)) (*appconfig.GetConfigurationProfileOutput, error) {
			return &appconfig.GetConfigurationProfileOutput{
				Id:   aws.String("profile-123"),
				Name: aws.String("test-profile"),
				Type: aws.String("AWS.Freeform"),
			}, nil
		},
		ListEnvironmentsFunc: func(ctx context.Context, params *appconfig.ListEnvironmentsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListEnvironmentsOutput, error) {
			return &appconfig.ListEnvironmentsOutput{
				Items: []types.Environment{
					{
						Id:   aws.String("env-123"),
						Name: aws.String("test-env"),
					},
				},
			}, nil
		},
	}

	awsClient := awsInternal.NewTestClient(mockAppConfigClient)

	getter := NewWithClient(cfg, awsClient)
	resolved, err := getter.ResolveResources(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resolved.ApplicationID != "app-123" {
		t.Errorf("expected app ID 'app-123', got '%s'", resolved.ApplicationID)
	}

	if resolved.Profile.ID != "profile-123" {
		t.Errorf("expected profile ID 'profile-123', got '%s'", resolved.Profile.ID)
	}

	if resolved.EnvironmentID != "env-123" {
		t.Errorf("expected environment ID 'env-123', got '%s'", resolved.EnvironmentID)
	}

	if resolved.DeploymentStrategyID != "" {
		t.Errorf("expected empty deployment strategy ID, got '%s'", resolved.DeploymentStrategyID)
	}
}

func TestResolveResourcesApplicationError(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Application:          "test-app",
		ConfigurationProfile: "test-profile",
		Environment:          "test-env",
	}

	mockAppConfigClient := &mock.MockAppConfigClient{
		ListApplicationsFunc: func(ctx context.Context, params *appconfig.ListApplicationsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListApplicationsOutput, error) {
			return nil, errors.New("API error")
		},
	}

	awsClient := awsInternal.NewTestClient(mockAppConfigClient)

	getter := NewWithClient(cfg, awsClient)
	_, err := getter.ResolveResources(context.Background())

	if err == nil {
		t.Fatal("expected error")
	}

	if !strings.Contains(err.Error(), "failed to list applications") {
		t.Errorf("expected 'failed to list applications' in error, got: %v", err)
	}
}

func TestResolveResourcesProfileError(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Application:          "test-app",
		ConfigurationProfile: "test-profile",
		Environment:          "test-env",
	}

	mockAppConfigClient := &mock.MockAppConfigClient{
		ListApplicationsFunc: func(ctx context.Context, params *appconfig.ListApplicationsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListApplicationsOutput, error) {
			return &appconfig.ListApplicationsOutput{
				Items: []types.Application{{Id: aws.String("app-123"), Name: aws.String("test-app")}},
			}, nil
		},
		ListConfigurationProfilesFunc: func(ctx context.Context, params *appconfig.ListConfigurationProfilesInput, optFns ...func(*appconfig.Options)) (*appconfig.ListConfigurationProfilesOutput, error) {
			return nil, errors.New("API error")
		},
	}

	awsClient := awsInternal.NewTestClient(mockAppConfigClient)

	getter := NewWithClient(cfg, awsClient)
	_, err := getter.ResolveResources(context.Background())

	if err == nil {
		t.Fatal("expected error")
	}

	if !strings.Contains(err.Error(), "failed to list configuration profiles") {
		t.Errorf("expected 'failed to list configuration profiles' in error, got: %v", err)
	}
}

func TestResolveResourcesEnvironmentError(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Application:          "test-app",
		ConfigurationProfile: "test-profile",
		Environment:          "test-env",
	}

	mockAppConfigClient := &mock.MockAppConfigClient{
		ListApplicationsFunc: func(ctx context.Context, params *appconfig.ListApplicationsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListApplicationsOutput, error) {
			return &appconfig.ListApplicationsOutput{
				Items: []types.Application{{Id: aws.String("app-123"), Name: aws.String("test-app")}},
			}, nil
		},
		ListConfigurationProfilesFunc: func(ctx context.Context, params *appconfig.ListConfigurationProfilesInput, optFns ...func(*appconfig.Options)) (*appconfig.ListConfigurationProfilesOutput, error) {
			return &appconfig.ListConfigurationProfilesOutput{
				Items: []types.ConfigurationProfileSummary{{Id: aws.String("profile-123"), Name: aws.String("test-profile")}},
			}, nil
		},
		GetConfigurationProfileFunc: func(ctx context.Context, params *appconfig.GetConfigurationProfileInput, optFns ...func(*appconfig.Options)) (*appconfig.GetConfigurationProfileOutput, error) {
			return &appconfig.GetConfigurationProfileOutput{
				Id:   aws.String("profile-123"),
				Type: aws.String("AWS.Freeform"),
			}, nil
		},
		ListEnvironmentsFunc: func(ctx context.Context, params *appconfig.ListEnvironmentsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListEnvironmentsOutput, error) {
			return nil, errors.New("API error")
		},
	}

	awsClient := awsInternal.NewTestClient(mockAppConfigClient)

	getter := NewWithClient(cfg, awsClient)
	_, err := getter.ResolveResources(context.Background())

	if err == nil {
		t.Fatal("expected error")
	}

	if !strings.Contains(err.Error(), "failed to list environments") {
		t.Errorf("expected 'failed to list environments' in error, got: %v", err)
	}
}

func TestGetConfigurationSuccess(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Application:          "test-app",
		ConfigurationProfile: "test-profile",
		Environment:          "test-env",
	}

	configData := []byte(`{"key": "value"}`)

	mockAppConfigDataClient := &mock.MockAppConfigDataClient{
		StartConfigurationSessionFunc: func(ctx context.Context, params *appconfigdata.StartConfigurationSessionInput, optFns ...func(*appconfigdata.Options)) (*appconfigdata.StartConfigurationSessionOutput, error) {
			return &appconfigdata.StartConfigurationSessionOutput{
				InitialConfigurationToken: aws.String("token-123"),
			}, nil
		},
		GetLatestConfigurationFunc: func(ctx context.Context, params *appconfigdata.GetLatestConfigurationInput, optFns ...func(*appconfigdata.Options)) (*appconfigdata.GetLatestConfigurationOutput, error) {
			return &appconfigdata.GetLatestConfigurationOutput{
				Configuration: configData,
			}, nil
		},
	}

	awsClient := &awsInternal.Client{
		AppConfigData: mockAppConfigDataClient,
	}

	getter := NewWithClient(cfg, awsClient)

	resolved := &awsInternal.ResolvedResources{
		ApplicationID: "app-123",
		Profile: &awsInternal.ProfileInfo{
			ID:   "profile-123",
			Name: "test-profile",
			Type: "AWS.Freeform",
		},
		EnvironmentID: "env-123",
	}

	result, err := getter.GetConfiguration(context.Background(), resolved)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if string(result) != string(configData) {
		t.Errorf("expected config data %q, got %q", configData, result)
	}
}
