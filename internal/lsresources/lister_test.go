package lsresources

import (
	"context"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/appconfig"
	appconfigTypes "github.com/aws/aws-sdk-go-v2/service/appconfig/types"
	awsInternal "github.com/koh-sh/apcdeploy/internal/aws"
	awsMock "github.com/koh-sh/apcdeploy/internal/aws/mock"
)

func TestLister_ListResources(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		region         string
		setupMock      func(*awsMock.MockAppConfigClient)
		expectedResult *ResourcesTree
		expectError    bool
		errorContains  string
	}{
		{
			name:   "successful listing with multiple applications",
			region: "us-east-1",
			setupMock: func(m *awsMock.MockAppConfigClient) {
				// Mock ListDeploymentStrategies
				m.ListDeploymentStrategiesFunc = func(ctx context.Context, params *appconfig.ListDeploymentStrategiesInput, optFns ...func(*appconfig.Options)) (*appconfig.ListDeploymentStrategiesOutput, error) {
					return &appconfig.ListDeploymentStrategiesOutput{
						Items: []appconfigTypes.DeploymentStrategy{
							{
								Name:                        aws.String("AppConfig.AllAtOnce"),
								Id:                          aws.String("strategy-1"),
								Description:                 aws.String("Quick deployment"),
								DeploymentDurationInMinutes: 0,
								FinalBakeTimeInMinutes:      0,
								GrowthFactor:                aws.Float32(100),
								GrowthType:                  appconfigTypes.GrowthTypeLinear,
							},
							{
								Name:                        aws.String("AppConfig.Linear"),
								Id:                          aws.String("strategy-2"),
								Description:                 aws.String("Linear deployment"),
								DeploymentDurationInMinutes: 30,
								FinalBakeTimeInMinutes:      10,
								GrowthFactor:                aws.Float32(20),
								GrowthType:                  appconfigTypes.GrowthTypeLinear,
							},
						},
					}, nil
				}

				// Mock ListApplications
				m.ListApplicationsFunc = func(ctx context.Context, params *appconfig.ListApplicationsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListApplicationsOutput, error) {
					return &appconfig.ListApplicationsOutput{
						Items: []appconfigTypes.Application{
							{Name: aws.String("app1"), Id: aws.String("app-id-1")},
							{Name: aws.String("app2"), Id: aws.String("app-id-2")},
						},
					}, nil
				}

				// Mock ListConfigurationProfiles
				m.ListConfigurationProfilesFunc = func(ctx context.Context, params *appconfig.ListConfigurationProfilesInput, optFns ...func(*appconfig.Options)) (*appconfig.ListConfigurationProfilesOutput, error) {
					switch *params.ApplicationId {
					case "app-id-1":
						return &appconfig.ListConfigurationProfilesOutput{
							Items: []appconfigTypes.ConfigurationProfileSummary{
								{Name: aws.String("profile1"), Id: aws.String("prof-id-1")},
								{Name: aws.String("profile2"), Id: aws.String("prof-id-2")},
							},
						}, nil
					case "app-id-2":
						return &appconfig.ListConfigurationProfilesOutput{
							Items: []appconfigTypes.ConfigurationProfileSummary{
								{Name: aws.String("profile3"), Id: aws.String("prof-id-3")},
							},
						}, nil
					}
					return &appconfig.ListConfigurationProfilesOutput{}, nil
				}

				// Mock ListEnvironments
				m.ListEnvironmentsFunc = func(ctx context.Context, params *appconfig.ListEnvironmentsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListEnvironmentsOutput, error) {
					switch *params.ApplicationId {
					case "app-id-1":
						return &appconfig.ListEnvironmentsOutput{
							Items: []appconfigTypes.Environment{
								{Name: aws.String("dev"), Id: aws.String("env-id-1")},
								{Name: aws.String("prod"), Id: aws.String("env-id-2")},
							},
						}, nil
					case "app-id-2":
						return &appconfig.ListEnvironmentsOutput{
							Items: []appconfigTypes.Environment{
								{Name: aws.String("staging"), Id: aws.String("env-id-3")},
							},
						}, nil
					}
					return &appconfig.ListEnvironmentsOutput{}, nil
				}
			},
			expectedResult: &ResourcesTree{
				Region: "us-east-1",
				DeploymentStrategies: []DeploymentStrategy{
					{
						Name:                        "AppConfig.AllAtOnce",
						ID:                          "strategy-1",
						Description:                 "Quick deployment",
						DeploymentDurationInMinutes: 0,
						FinalBakeTimeInMinutes:      0,
						GrowthFactor:                100,
						GrowthType:                  "LINEAR",
					},
					{
						Name:                        "AppConfig.Linear",
						ID:                          "strategy-2",
						Description:                 "Linear deployment",
						DeploymentDurationInMinutes: 30,
						FinalBakeTimeInMinutes:      10,
						GrowthFactor:                20,
						GrowthType:                  "LINEAR",
					},
				},
				Applications: []Application{
					{
						Name: "app1",
						ID:   "app-id-1",
						Profiles: []ConfigurationProfile{
							{Name: "profile1", ID: "prof-id-1"},
							{Name: "profile2", ID: "prof-id-2"},
						},
						Environments: []Environment{
							{Name: "dev", ID: "env-id-1"},
							{Name: "prod", ID: "env-id-2"},
						},
					},
					{
						Name: "app2",
						ID:   "app-id-2",
						Profiles: []ConfigurationProfile{
							{Name: "profile3", ID: "prof-id-3"},
						},
						Environments: []Environment{
							{Name: "staging", ID: "env-id-3"},
						},
					},
				},
			},
			expectError: false,
		},
		{
			name:   "no applications found",
			region: "us-west-2",
			setupMock: func(m *awsMock.MockAppConfigClient) {
				m.ListDeploymentStrategiesFunc = func(ctx context.Context, params *appconfig.ListDeploymentStrategiesInput, optFns ...func(*appconfig.Options)) (*appconfig.ListDeploymentStrategiesOutput, error) {
					return &appconfig.ListDeploymentStrategiesOutput{
						Items: []appconfigTypes.DeploymentStrategy{},
					}, nil
				}
				m.ListApplicationsFunc = func(ctx context.Context, params *appconfig.ListApplicationsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListApplicationsOutput, error) {
					return &appconfig.ListApplicationsOutput{
						Items: []appconfigTypes.Application{},
					}, nil
				}
			},
			expectedResult: &ResourcesTree{
				Region:               "us-west-2",
				Applications:         []Application{},
				DeploymentStrategies: []DeploymentStrategy{},
			},
			expectError: false,
		},
		{
			name:   "application with no profiles or environments",
			region: "eu-west-1",
			setupMock: func(m *awsMock.MockAppConfigClient) {
				m.ListDeploymentStrategiesFunc = func(ctx context.Context, params *appconfig.ListDeploymentStrategiesInput, optFns ...func(*appconfig.Options)) (*appconfig.ListDeploymentStrategiesOutput, error) {
					return &appconfig.ListDeploymentStrategiesOutput{
						Items: []appconfigTypes.DeploymentStrategy{},
					}, nil
				}

				m.ListApplicationsFunc = func(ctx context.Context, params *appconfig.ListApplicationsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListApplicationsOutput, error) {
					return &appconfig.ListApplicationsOutput{
						Items: []appconfigTypes.Application{
							{Name: aws.String("empty-app"), Id: aws.String("app-empty")},
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
			expectedResult: &ResourcesTree{
				Region:               "eu-west-1",
				DeploymentStrategies: []DeploymentStrategy{},
				Applications: []Application{
					{
						Name:         "empty-app",
						ID:           "app-empty",
						Profiles:     []ConfigurationProfile{},
						Environments: []Environment{},
					},
				},
			},
			expectError: false,
		},
		{
			name:   "applications with nil names or IDs are skipped",
			region: "us-west-2",
			setupMock: func(m *awsMock.MockAppConfigClient) {
				m.ListDeploymentStrategiesFunc = func(ctx context.Context, params *appconfig.ListDeploymentStrategiesInput, optFns ...func(*appconfig.Options)) (*appconfig.ListDeploymentStrategiesOutput, error) {
					return &appconfig.ListDeploymentStrategiesOutput{
						Items: []appconfigTypes.DeploymentStrategy{},
					}, nil
				}

				m.ListApplicationsFunc = func(ctx context.Context, params *appconfig.ListApplicationsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListApplicationsOutput, error) {
					return &appconfig.ListApplicationsOutput{
						Items: []appconfigTypes.Application{
							{Name: nil, Id: aws.String("app-1")},                     // Missing name
							{Name: aws.String("valid-app"), Id: aws.String("app-2")}, // Valid
							{Name: aws.String("no-id-app"), Id: nil},                 // Missing ID
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
			expectedResult: &ResourcesTree{
				Region:               "us-west-2",
				DeploymentStrategies: []DeploymentStrategy{},
				Applications: []Application{
					{
						Name:         "valid-app",
						ID:           "app-2",
						Profiles:     []ConfigurationProfile{},
						Environments: []Environment{},
					},
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Setup
			mockAPI := &awsMock.MockAppConfigClient{}
			tt.setupMock(mockAPI)

			client := &awsInternal.Client{
				AppConfig: mockAPI,
			}

			lister := New(client, tt.region)

			// Execute
			result, err := lister.ListResources(context.Background())

			// Assert
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
				if !compareResourcesTree(tt.expectedResult, result) {
					t.Errorf("expected result %+v, got %+v", tt.expectedResult, result)
				}
			}
		})
	}
}

// Helper functions for testing
func compareResourcesTree(expected, actual *ResourcesTree) bool {
	if expected == nil && actual == nil {
		return true
	}
	if expected == nil || actual == nil {
		return false
	}
	if expected.Region != actual.Region {
		return false
	}
	if len(expected.Applications) != len(actual.Applications) {
		return false
	}
	for i := range expected.Applications {
		if !compareApplication(&expected.Applications[i], &actual.Applications[i]) {
			return false
		}
	}
	if len(expected.DeploymentStrategies) != len(actual.DeploymentStrategies) {
		return false
	}
	for i := range expected.DeploymentStrategies {
		if !compareDeploymentStrategy(&expected.DeploymentStrategies[i], &actual.DeploymentStrategies[i]) {
			return false
		}
	}
	return true
}

func compareApplication(expected, actual *Application) bool {
	if expected.Name != actual.Name || expected.ID != actual.ID {
		return false
	}
	if len(expected.Profiles) != len(actual.Profiles) {
		return false
	}
	if len(expected.Environments) != len(actual.Environments) {
		return false
	}
	for i := range expected.Profiles {
		if expected.Profiles[i] != actual.Profiles[i] {
			return false
		}
	}
	for i := range expected.Environments {
		if expected.Environments[i] != actual.Environments[i] {
			return false
		}
	}
	return true
}

func compareDeploymentStrategy(expected, actual *DeploymentStrategy) bool {
	if expected.Name != actual.Name || expected.ID != actual.ID {
		return false
	}
	if expected.Description != actual.Description {
		return false
	}
	if expected.DeploymentDurationInMinutes != actual.DeploymentDurationInMinutes {
		return false
	}
	if expected.FinalBakeTimeInMinutes != actual.FinalBakeTimeInMinutes {
		return false
	}
	if expected.GrowthFactor != actual.GrowthFactor {
		return false
	}
	if expected.GrowthType != actual.GrowthType {
		return false
	}
	if expected.ReplicateTo != actual.ReplicateTo {
		return false
	}
	return true
}
