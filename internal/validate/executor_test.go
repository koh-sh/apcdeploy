package validate

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/appconfig"
	"github.com/aws/aws-sdk-go-v2/service/appconfig/types"
	awsInternal "github.com/koh-sh/apcdeploy/internal/aws"
	"github.com/koh-sh/apcdeploy/internal/aws/mock"
	batchtest "github.com/koh-sh/apcdeploy/internal/batch/testing"
	"github.com/koh-sh/apcdeploy/internal/config"
	reportertest "github.com/koh-sh/apcdeploy/internal/reporter/testing"
)

func TestNewExecutor(t *testing.T) {
	t.Parallel()

	rep := &reportertest.MockReporter{}
	executor := NewExecutor(rep)
	if executor.reporter != rep {
		t.Error("expected executor to have the provided reporter")
	}
}

// buildMock returns a mock AppConfig client that resolves the canonical
// test-app/test-profile/test-env triple, with the given profile type and validators.
func buildMock(profileType string, validators []types.Validator) *mock.MockAppConfigClient {
	return &mock.MockAppConfigClient{
		ListApplicationsFunc: func(ctx context.Context, params *appconfig.ListApplicationsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListApplicationsOutput, error) {
			return &appconfig.ListApplicationsOutput{Items: []types.Application{{Id: aws.String("app-1"), Name: aws.String("test-app")}}}, nil
		},
		ListConfigurationProfilesFunc: func(ctx context.Context, params *appconfig.ListConfigurationProfilesInput, optFns ...func(*appconfig.Options)) (*appconfig.ListConfigurationProfilesOutput, error) {
			return &appconfig.ListConfigurationProfilesOutput{Items: []types.ConfigurationProfileSummary{{Id: aws.String("prof-1"), Name: aws.String("test-profile")}}}, nil
		},
		ListEnvironmentsFunc: func(ctx context.Context, params *appconfig.ListEnvironmentsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListEnvironmentsOutput, error) {
			return &appconfig.ListEnvironmentsOutput{Items: []types.Environment{{Id: aws.String("env-1"), Name: aws.String("test-env")}}}, nil
		},
		GetConfigurationProfileFunc: func(ctx context.Context, params *appconfig.GetConfigurationProfileInput, optFns ...func(*appconfig.Options)) (*appconfig.GetConfigurationProfileOutput, error) {
			return &appconfig.GetConfigurationProfileOutput{Id: aws.String("prof-1"), Name: aws.String("test-profile"), Type: aws.String(profileType), Validators: validators}, nil
		},
	}
}

func TestRunOnTarget(t *testing.T) {
	t.Parallel()

	const freeformSchema = `{"type":"object","properties":{"port":{"type":"integer","minimum":1}},"required":["port"]}`
	jsonSchemaValidator := []types.Validator{{Type: types.ValidatorTypeJsonSchema, Content: aws.String(freeformSchema)}}

	tests := []struct {
		name        string
		profileType string
		validators  []types.Validator
		dataName    string
		dataContent string
		wantErr     bool
	}{
		{
			name:        "featureflags valid",
			profileType: config.ProfileTypeFeatureFlags,
			dataName:    "data.json",
			dataContent: `{"version":"1","flags":{"f":{"attributes":{"c":{"constraints":{"type":"string","enum":["a"]}}}}},"values":{"f":{"enabled":true,"c":"a"}}}`,
		},
		{
			name:        "featureflags constraint violation",
			profileType: config.ProfileTypeFeatureFlags,
			dataName:    "data.json",
			dataContent: `{"version":"1","flags":{"f":{"attributes":{"c":{"constraints":{"type":"string","enum":["a"]}}}}},"values":{"f":{"enabled":true,"c":"z"}}}`,
			wantErr:     true,
		},
		{
			name:        "freeform remote schema valid",
			profileType: config.ProfileTypeFreeform,
			validators:  jsonSchemaValidator,
			dataName:    "data.json",
			dataContent: `{"port":8080}`,
		},
		{
			name:        "freeform remote schema violation",
			profileType: config.ProfileTypeFreeform,
			validators:  jsonSchemaValidator,
			dataName:    "data.json",
			dataContent: `{"port":-1}`,
			wantErr:     true,
		},
		{
			name:        "freeform no schema syntax only",
			profileType: config.ProfileTypeFreeform,
			dataName:    "data.json",
			dataContent: `{"anything":true}`,
		},
		{
			name:        "freeform yaml syntax only",
			profileType: config.ProfileTypeFreeform,
			validators:  jsonSchemaValidator,
			dataName:    "data.yaml",
			dataContent: "key: value\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			configYML := fmt.Sprintf("application: test-app\nconfiguration_profile: test-profile\nenvironment: test-env\ndata_file: %s\nregion: us-east-1\n", tt.dataName)
			configPath := filepath.Join(dir, "apcdeploy.yml")
			if err := os.WriteFile(configPath, []byte(configYML), 0o644); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, tt.dataName), []byte(tt.dataContent), 0o644); err != nil {
				t.Fatal(err)
			}

			mockClient := buildMock(tt.profileType, tt.validators)
			rep := &reportertest.MockReporter{}
			executor := NewExecutorWithFactory(rep, func(ctx context.Context, region string) (*awsInternal.Client, error) {
				return awsInternal.NewTestClient(mockClient), nil
			})

			target, tr, cleanup := batchtest.BuildTarget(t, rep, configPath)
			defer cleanup()
			err := executor.RunOnTarget(context.Background(), target, tr)

			if tt.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
		})
	}
}

func TestResolveSchema(t *testing.T) {
	t.Parallel()

	const freeformSchema = `{"type":"object"}`
	jsonValidator := []types.Validator{{Type: types.ValidatorTypeJsonSchema, Content: aws.String(freeformSchema)}}
	lambdaValidator := []types.Validator{{Type: types.ValidatorTypeLambda, Content: aws.String("arn:aws:lambda:...")}}

	tests := []struct {
		name        string
		profile     *awsInternal.ProfileInfo
		contentType string
		wantSchema  bool
		wantSummary string
	}{
		{
			name:        "featureflags uses built-in schema",
			profile:     &awsInternal.ProfileInfo{Type: config.ProfileTypeFeatureFlags},
			contentType: config.ContentTypeJSON,
			wantSummary: "valid (feature flags schema)",
		},
		{
			name:        "freeform yaml syntax only",
			profile:     &awsInternal.ProfileInfo{Type: config.ProfileTypeFreeform},
			contentType: config.ContentTypeYAML,
			wantSummary: "valid (syntax only)",
		},
		{
			name:        "freeform json remote validator",
			profile:     &awsInternal.ProfileInfo{Type: config.ProfileTypeFreeform, Validators: jsonValidator},
			contentType: config.ContentTypeJSON,
			wantSchema:  true,
			wantSummary: "valid (remote schema)",
		},
		{
			name:        "freeform json lambda only",
			profile:     &awsInternal.ProfileInfo{Type: config.ProfileTypeFreeform, Validators: lambdaValidator},
			contentType: config.ContentTypeJSON,
			wantSummary: "valid (syntax only; lambda validator not checked)",
		},
		{
			name:        "freeform json no validator",
			profile:     &awsInternal.ProfileInfo{Type: config.ProfileTypeFreeform},
			contentType: config.ContentTypeJSON,
			wantSummary: "valid (syntax only; no schema)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			schema, summary := resolveSchema(tt.profile, tt.contentType)
			if (schema != nil) != tt.wantSchema {
				t.Fatalf("schema present = %v, want %v", schema != nil, tt.wantSchema)
			}
			if summary != tt.wantSummary {
				t.Fatalf("summary = %q, want %q", summary, tt.wantSummary)
			}
		})
	}
}

func TestRunOnTarget_ClientFactoryError(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	configPath := filepath.Join(dir, "apcdeploy.yml")
	if err := os.WriteFile(configPath, []byte("application: a\nconfiguration_profile: p\nenvironment: e\nregion: us-east-1\ndata_file: data.json\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "data.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	rep := &reportertest.MockReporter{}
	executor := NewExecutorWithFactory(rep, func(ctx context.Context, region string) (*awsInternal.Client, error) {
		return nil, fmt.Errorf("client init boom")
	})

	target, tr, cleanup := batchtest.BuildTarget(t, rep, configPath)
	defer cleanup()
	if err := executor.RunOnTarget(context.Background(), target, tr); err == nil {
		t.Fatal("expected error from client factory failure, got nil")
	}
}

func TestRunOnTarget_ErrorPaths(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		writeData  bool
		mockClient *mock.MockAppConfigClient
	}{
		{
			name:      "resolve error",
			writeData: true,
			mockClient: &mock.MockAppConfigClient{
				ListApplicationsFunc: func(ctx context.Context, params *appconfig.ListApplicationsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListApplicationsOutput, error) {
					return nil, fmt.Errorf("list applications boom")
				},
			},
		},
		{
			name:       "data file missing",
			writeData:  false,
			mockClient: buildMock(config.ProfileTypeFreeform, nil),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			configPath := filepath.Join(dir, "apcdeploy.yml")
			if err := os.WriteFile(configPath, []byte("application: test-app\nconfiguration_profile: test-profile\nenvironment: test-env\nregion: us-east-1\ndata_file: data.json\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			if tt.writeData {
				if err := os.WriteFile(filepath.Join(dir, "data.json"), []byte("{}"), 0o644); err != nil {
					t.Fatal(err)
				}
			}

			rep := &reportertest.MockReporter{}
			executor := NewExecutorWithFactory(rep, func(ctx context.Context, region string) (*awsInternal.Client, error) {
				return awsInternal.NewTestClient(tt.mockClient), nil
			})

			target, tr, cleanup := batchtest.BuildTarget(t, rep, configPath)
			defer cleanup()
			if err := executor.RunOnTarget(context.Background(), target, tr); err == nil {
				t.Fatalf("expected error, got nil")
			}
		})
	}
}
