package lsresources

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/appconfig"
	appconfigTypes "github.com/aws/aws-sdk-go-v2/service/appconfig/types"
	awsInternal "github.com/koh-sh/apcdeploy/internal/aws"
	awsMock "github.com/koh-sh/apcdeploy/internal/aws/mock"
	reporterTesting "github.com/koh-sh/apcdeploy/internal/reporter/testing"
)

func TestNewExecutor(t *testing.T) {
	t.Parallel()

	mockReporter := &reporterTesting.MockReporter{}
	executor := NewExecutor(mockReporter)

	if executor.reporter != mockReporter {
		t.Error("expected reporter to be set correctly")
	}
	if executor.clientFactory == nil {
		t.Error("expected clientFactory to be set")
	}
}

func TestExecutor_Execute(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		opts           *Options
		setupMock      func(*awsMock.MockAppConfigClient)
		validateOutput func(*testing.T, string)
		expectError    bool
		errorContains  string
	}{
		{
			name: "successful execution with JSON output",
			opts: &Options{
				Region: "us-east-1",
				JSON:   true,
				Silent: false,
			},
			setupMock: func(m *awsMock.MockAppConfigClient) {
				m.ListDeploymentStrategiesFunc = func(ctx context.Context, params *appconfig.ListDeploymentStrategiesInput, optFns ...func(*appconfig.Options)) (*appconfig.ListDeploymentStrategiesOutput, error) {
					return &appconfig.ListDeploymentStrategiesOutput{
						Items: []appconfigTypes.DeploymentStrategy{},
					}, nil
				}
				m.ListApplicationsFunc = func(ctx context.Context, params *appconfig.ListApplicationsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListApplicationsOutput, error) {
					return &appconfig.ListApplicationsOutput{
						Items: []appconfigTypes.Application{
							{Name: aws.String("test-app"), Id: aws.String("app-id-1")},
						},
					}, nil
				}
				m.ListConfigurationProfilesFunc = func(ctx context.Context, params *appconfig.ListConfigurationProfilesInput, optFns ...func(*appconfig.Options)) (*appconfig.ListConfigurationProfilesOutput, error) {
					return &appconfig.ListConfigurationProfilesOutput{
						Items: []appconfigTypes.ConfigurationProfileSummary{
							{Name: aws.String("test-profile"), Id: aws.String("prof-id-1")},
						},
					}, nil
				}
				m.ListEnvironmentsFunc = func(ctx context.Context, params *appconfig.ListEnvironmentsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListEnvironmentsOutput, error) {
					return &appconfig.ListEnvironmentsOutput{
						Items: []appconfigTypes.Environment{
							{Name: aws.String("test-env"), Id: aws.String("env-id-1")},
						},
					}, nil
				}
			},
			validateOutput: func(t *testing.T, output string) {
				if !strings.Contains(output, "test-app") {
					t.Errorf("expected output to contain 'test-app', got: %s", output)
				}
				if !strings.Contains(output, "test-profile") {
					t.Errorf("expected output to contain 'test-profile', got: %s", output)
				}
				if !strings.Contains(output, "test-env") {
					t.Errorf("expected output to contain 'test-env', got: %s", output)
				}
			},
			expectError: false,
		},
		{
			name: "successful execution with human-readable output",
			opts: &Options{
				Region: "us-west-2",
				JSON:   false,
				Silent: false,
			},
			setupMock: func(m *awsMock.MockAppConfigClient) {
				m.ListDeploymentStrategiesFunc = func(ctx context.Context, params *appconfig.ListDeploymentStrategiesInput, optFns ...func(*appconfig.Options)) (*appconfig.ListDeploymentStrategiesOutput, error) {
					return &appconfig.ListDeploymentStrategiesOutput{
						Items: []appconfigTypes.DeploymentStrategy{},
					}, nil
				}
				m.ListApplicationsFunc = func(ctx context.Context, params *appconfig.ListApplicationsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListApplicationsOutput, error) {
					return &appconfig.ListApplicationsOutput{
						Items: []appconfigTypes.Application{
							{Name: aws.String("my-app"), Id: aws.String("app-123")},
						},
					}, nil
				}
				m.ListConfigurationProfilesFunc = func(ctx context.Context, params *appconfig.ListConfigurationProfilesInput, optFns ...func(*appconfig.Options)) (*appconfig.ListConfigurationProfilesOutput, error) {
					return &appconfig.ListConfigurationProfilesOutput{
						Items: []appconfigTypes.ConfigurationProfileSummary{},
					}, nil
				}
				m.ListEnvironmentsFunc = func(ctx context.Context, params *appconfig.ListEnvironmentsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListEnvironmentsOutput, error) {
					return &appconfig.ListEnvironmentsOutput{
						Items: []appconfigTypes.Environment{},
					}, nil
				}
			},
			validateOutput: func(t *testing.T, output string) {
				if !strings.Contains(output, "Region: us-west-2") {
					t.Errorf("expected output to contain 'Region: us-west-2', got: %s", output)
				}
				if !strings.Contains(output, "my-app") {
					t.Errorf("expected output to contain 'my-app', got: %s", output)
				}
			},
			expectError: false,
		},
		{
			name: "error listing applications",
			opts: &Options{
				Region: "eu-west-1",
				JSON:   false,
				Silent: false,
			},
			setupMock: func(m *awsMock.MockAppConfigClient) {
				m.ListDeploymentStrategiesFunc = func(ctx context.Context, params *appconfig.ListDeploymentStrategiesInput, optFns ...func(*appconfig.Options)) (*appconfig.ListDeploymentStrategiesOutput, error) {
					return &appconfig.ListDeploymentStrategiesOutput{
						Items: []appconfigTypes.DeploymentStrategy{},
					}, nil
				}
				m.ListApplicationsFunc = func(ctx context.Context, params *appconfig.ListApplicationsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListApplicationsOutput, error) {
					return nil, errors.New("API error")
				}
			},
			validateOutput: func(t *testing.T, output string) {
				// No output expected on error
			},
			expectError:   true,
			errorContains: "failed to list resources",
		},
		{
			name: "error listing configuration profiles",
			opts: &Options{
				Region: "us-east-1",
				JSON:   false,
				Silent: false,
			},
			setupMock: func(m *awsMock.MockAppConfigClient) {
				m.ListDeploymentStrategiesFunc = func(ctx context.Context, params *appconfig.ListDeploymentStrategiesInput, optFns ...func(*appconfig.Options)) (*appconfig.ListDeploymentStrategiesOutput, error) {
					return &appconfig.ListDeploymentStrategiesOutput{
						Items: []appconfigTypes.DeploymentStrategy{},
					}, nil
				}
				m.ListApplicationsFunc = func(ctx context.Context, params *appconfig.ListApplicationsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListApplicationsOutput, error) {
					return &appconfig.ListApplicationsOutput{
						Items: []appconfigTypes.Application{
							{Name: aws.String("test-app"), Id: aws.String("app-id-1")},
						},
					}, nil
				}
				m.ListConfigurationProfilesFunc = func(ctx context.Context, params *appconfig.ListConfigurationProfilesInput, optFns ...func(*appconfig.Options)) (*appconfig.ListConfigurationProfilesOutput, error) {
					return nil, errors.New("profile API error")
				}
			},
			validateOutput: func(t *testing.T, output string) {
				// No output expected on error
			},
			expectError:   true,
			errorContains: "failed to list resources",
		},
		{
			name: "error listing environments",
			opts: &Options{
				Region: "us-east-1",
				JSON:   false,
				Silent: false,
			},
			setupMock: func(m *awsMock.MockAppConfigClient) {
				m.ListDeploymentStrategiesFunc = func(ctx context.Context, params *appconfig.ListDeploymentStrategiesInput, optFns ...func(*appconfig.Options)) (*appconfig.ListDeploymentStrategiesOutput, error) {
					return &appconfig.ListDeploymentStrategiesOutput{
						Items: []appconfigTypes.DeploymentStrategy{},
					}, nil
				}
				m.ListApplicationsFunc = func(ctx context.Context, params *appconfig.ListApplicationsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListApplicationsOutput, error) {
					return &appconfig.ListApplicationsOutput{
						Items: []appconfigTypes.Application{
							{Name: aws.String("test-app"), Id: aws.String("app-id-1")},
						},
					}, nil
				}
				m.ListConfigurationProfilesFunc = func(ctx context.Context, params *appconfig.ListConfigurationProfilesInput, optFns ...func(*appconfig.Options)) (*appconfig.ListConfigurationProfilesOutput, error) {
					return &appconfig.ListConfigurationProfilesOutput{
						Items: []appconfigTypes.ConfigurationProfileSummary{},
					}, nil
				}
				m.ListEnvironmentsFunc = func(ctx context.Context, params *appconfig.ListEnvironmentsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListEnvironmentsOutput, error) {
					return nil, errors.New("environment API error")
				}
			},
			validateOutput: func(t *testing.T, output string) {
				// No output expected on error
			},
			expectError:   true,
			errorContains: "failed to list resources",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Setup mock AWS client
			mockAppConfig := &awsMock.MockAppConfigClient{}
			tt.setupMock(mockAppConfig)

			// Create factory function that returns our mock client
			factory := func(ctx context.Context, region string) (*awsInternal.Client, error) {
				// Use provided region, or default to us-east-1 if empty
				actualRegion := region
				if actualRegion == "" {
					actualRegion = "us-east-1"
				}
				return &awsInternal.Client{
					AppConfig: mockAppConfig,
					Region:    actualRegion,
				}, nil
			}

			// Setup reporter
			mockReporter := &reporterTesting.MockReporter{}

			// Create executor with factory
			executor := NewExecutorWithFactory(mockReporter, factory)

			// Capture output
			var output bytes.Buffer

			// Execute
			err := executor.ExecuteWithWriter(context.Background(), tt.opts, &output)

			// Assertions
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got nil")
				}
				if tt.errorContains != "" && err != nil {
					if !strings.Contains(err.Error(), tt.errorContains) {
						t.Errorf("expected error to contain %q, got %q", tt.errorContains, err.Error())
					}
				}
			} else {
				if err != nil {
					t.Errorf("expected no error but got: %v", err)
				}
				tt.validateOutput(t, output.String())
			}
		})
	}
}
