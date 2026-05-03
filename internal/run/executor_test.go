package run

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/appconfig"
	"github.com/aws/aws-sdk-go-v2/service/appconfig/types"
	awsInternal "github.com/koh-sh/apcdeploy/internal/aws"
	"github.com/koh-sh/apcdeploy/internal/aws/mock"
	"github.com/koh-sh/apcdeploy/internal/config"
	reportertest "github.com/koh-sh/apcdeploy/internal/reporter/testing"
)

func TestNewExecutor(t *testing.T) {
	reporter := &reportertest.MockReporter{}
	executor := NewExecutor(reporter)

	if executor == nil {
		t.Fatal("expected executor to be non-nil")
		return
	}

	if executor.reporter != reporter {
		t.Error("expected executor to have the provided reporter")
	}
}

func TestExecutorValidateTimeout(t *testing.T) {
	tests := []struct {
		name        string
		timeout     int
		wantErr     bool
		expectedMsg string
	}{
		{
			name:        "negative timeout is invalid",
			timeout:     -1,
			wantErr:     true,
			expectedMsg: "timeout must be a non-negative value",
		},
		{
			name:    "zero timeout is valid",
			timeout: 0,
			wantErr: false,
		},
		{
			name:    "positive timeout is valid",
			timeout: 300,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reporter := &reportertest.MockReporter{}
			executor := NewExecutor(reporter)

			opts := &Options{
				ConfigFile: "nonexistent.yml",
				WaitDeploy: false,
				WaitBake:   false,
				Timeout:    tt.timeout,
			}

			err := executor.Execute(context.Background(), opts)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error for negative timeout, got nil")
				} else if !strings.Contains(err.Error(), tt.expectedMsg) {
					t.Errorf("expected error containing %q, got %q", tt.expectedMsg, err.Error())
				}
			} else {
				// We expect an error here because the config file doesn't exist
				// but it should not be a timeout validation error
				if err != nil && strings.Contains(err.Error(), "timeout must be a non-negative value") {
					t.Errorf("unexpected timeout validation error: %v", err)
				}
			}
		})
	}
}

func TestExecutorValidateWaitFlags(t *testing.T) {
	reporter := &reportertest.MockReporter{}
	executor := NewExecutor(reporter)

	opts := &Options{
		ConfigFile: "nonexistent.yml",
		WaitDeploy: true,
		WaitBake:   true,
		Timeout:    300,
	}

	err := executor.Execute(context.Background(), opts)

	if err == nil {
		t.Error("expected error when both --wait-deploy and --wait-bake are specified")
		return
	}

	if !strings.Contains(err.Error(), "--wait-deploy and --wait-bake cannot be used together") {
		t.Errorf("expected error about mutually exclusive flags, got: %v", err)
	}
}

func TestExecutorLoadConfigurationError(t *testing.T) {
	reporter := &reportertest.MockReporter{}
	executor := NewExecutor(reporter)

	opts := &Options{
		ConfigFile: "nonexistent.yml",
		WaitDeploy: false,
		WaitBake:   false,
		Timeout:    300,
	}

	err := executor.Execute(context.Background(), opts)

	if err == nil {
		t.Error("expected error when loading non-existent config file")
	}

	if !strings.Contains(err.Error(), "failed to load configuration") {
		t.Errorf("expected 'failed to load configuration' error, got: %v", err)
	}

	// Config loading is an instant operation: per the output contract it does
	// not produce any reporter output on failure — the returned error is the
	// signal. The reporter should be untouched.
	if len(reporter.Messages) != 0 {
		t.Errorf("expected no reporter messages for config-load failure, got: %v", reporter.Messages)
	}
}

// TestExecutorFullWorkflowWithMock tests the complete deployment workflow with mocked AWS
func TestExecutorFullWorkflowWithMock(t *testing.T) {
	// Create temporary test files
	tempDir, err := os.MkdirTemp("", "executor-full-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create config file
	configPath := filepath.Join(tempDir, "apcdeploy.yml")
	configContent := `application: test-app
configuration_profile: test-profile
environment: test-env
deployment_strategy: AppConfig.AllAtOnce
data_file: data.json
region: us-east-1
`
	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	// Create data file
	dataPath := filepath.Join(tempDir, "data.json")
	if err := os.WriteFile(dataPath, []byte(`{"key": "value"}`), 0o644); err != nil {
		t.Fatalf("Failed to write data: %v", err)
	}

	// Create mock AWS client
	mockClient := &mock.MockAppConfigClient{
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
						Type: aws.String("AWS.Freeform"),
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
		ListDeploymentStrategiesFunc: func(ctx context.Context, params *appconfig.ListDeploymentStrategiesInput, optFns ...func(*appconfig.Options)) (*appconfig.ListDeploymentStrategiesOutput, error) {
			return &appconfig.ListDeploymentStrategiesOutput{
				Items: []types.DeploymentStrategy{
					{
						Id:   aws.String("strategy-123"),
						Name: aws.String("AppConfig.AllAtOnce"),
					},
				},
			}, nil
		},
		ListDeploymentsFunc: func(ctx context.Context, params *appconfig.ListDeploymentsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListDeploymentsOutput, error) {
			return &appconfig.ListDeploymentsOutput{
				Items: []types.DeploymentSummary{},
			}, nil
		},
		CreateHostedConfigurationVersionFunc: func(ctx context.Context, params *appconfig.CreateHostedConfigurationVersionInput, optFns ...func(*appconfig.Options)) (*appconfig.CreateHostedConfigurationVersionOutput, error) {
			return &appconfig.CreateHostedConfigurationVersionOutput{
				VersionNumber: 1,
			}, nil
		},
		StartDeploymentFunc: func(ctx context.Context, params *appconfig.StartDeploymentInput, optFns ...func(*appconfig.Options)) (*appconfig.StartDeploymentOutput, error) {
			return &appconfig.StartDeploymentOutput{
				DeploymentNumber: 1,
			}, nil
		},
		GetDeploymentFunc: func(ctx context.Context, params *appconfig.GetDeploymentInput, optFns ...func(*appconfig.Options)) (*appconfig.GetDeploymentOutput, error) {
			return &appconfig.GetDeploymentOutput{
				State: types.DeploymentStateComplete,
			}, nil
		},
	}

	// Create deployer factory that uses the mock client
	deployerFactory := func(ctx context.Context, cfg *config.Config) (*Deployer, error) {
		awsClient := awsInternal.NewTestClient(mockClient)
		return NewWithClient(cfg, awsClient), nil
	}

	reporter := &reportertest.MockReporter{}
	executor := NewExecutorWithFactory(reporter, deployerFactory)

	opts := &Options{
		ConfigFile: configPath,
		WaitDeploy: false,
		WaitBake:   false,
		Timeout:    300,
	}

	err = executor.Execute(context.Background(), opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Run now reports phases through a single Targets row that progresses
	// through preparing → comparing → creating-version → deploying, then
	// finalises with Done("started — vN, Strategy, deployment #N") when no
	// wait flag is set (output.md §7.1).
	if len(reporter.TargetsCalls) != 1 {
		t.Fatalf("expected exactly 1 Targets call, got %d", len(reporter.TargetsCalls))
	}
	tc := reporter.TargetsCalls[0]
	if !tc.Closed {
		t.Errorf("expected Targets to be Close()d")
	}
	wantPhases := []string{"preparing", "comparing", "creating-version", "deploying"}
	gotPhases := []string{}
	for _, tr := range tc.Transitions {
		if tr.Kind == "phase" {
			gotPhases = append(gotPhases, tr.Phase)
		}
	}
	for _, want := range wantPhases {
		seen := false
		for _, got := range gotPhases {
			if got == want {
				seen = true
			}
		}
		if !seen {
			t.Errorf("expected phase %q in: %v", want, gotPhases)
		}
	}
	foundStarted := false
	for _, tr := range tc.Transitions {
		if tr.Kind == "done" && strings.Contains(tr.Summary, "started") && strings.Contains(tr.Summary, "v1") {
			foundStarted = true
		}
	}
	if !foundStarted {
		t.Errorf("expected Targets.Done summary mentioning 'started' and 'v1'; got: %+v", tc.Transitions)
	}
}

// TestExecutorFullWorkflowWithWait tests deployment with wait options.
// The deploy sub-phase drives Targets.SetProgress (AppConfig-reported %),
// and the bake sub-phase uses Targets.SetPhase("baking", "(~N min left)")
// — bake is a monitoring wait so a spinner with countdown is more honest
// than a progress bar that would imply quantified rollout work.
func TestExecutorFullWorkflowWithWait(t *testing.T) {
	tests := []struct {
		name            string
		waitDeploy      bool
		waitBake        bool
		mockStates      []types.DeploymentState
		wantVerb        string // verb expected in the Targets.Done summary
		wantBakingPhase bool   // whether SetPhase("baking", ...) is expected
	}{
		{
			name:            "wait for bake: immediate completion",
			waitDeploy:      false,
			waitBake:        true,
			mockStates:      []types.DeploymentState{types.DeploymentStateComplete},
			wantVerb:        "complete",
			wantBakingPhase: true,
		},
		{
			name:            "wait for bake: completion after polling",
			waitDeploy:      false,
			waitBake:        true,
			mockStates:      []types.DeploymentState{types.DeploymentStateDeploying, types.DeploymentStateBaking, types.DeploymentStateComplete},
			wantVerb:        "complete",
			wantBakingPhase: true,
		},
		{
			name:            "wait for deploy: stops at baking",
			waitDeploy:      true,
			waitBake:        false,
			mockStates:      []types.DeploymentState{types.DeploymentStateDeploying, types.DeploymentStateBaking},
			wantVerb:        "deployed",
			wantBakingPhase: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary test files
			tempDir, err := os.MkdirTemp("", "executor-wait-*")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tempDir)

			configPath := filepath.Join(tempDir, "apcdeploy.yml")
			configContent := `application: test-app
configuration_profile: test-profile
environment: test-env
deployment_strategy: AppConfig.AllAtOnce
data_file: data.json
region: us-east-1
`
			if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
				t.Fatalf("Failed to write config: %v", err)
			}

			dataPath := filepath.Join(tempDir, "data.json")
			if err := os.WriteFile(dataPath, []byte(`{"key": "value"}`), 0o644); err != nil {
				t.Fatalf("Failed to write data: %v", err)
			}

			callCount := 0
			mockClient := &mock.MockAppConfigClient{
				ListApplicationsFunc: func(ctx context.Context, params *appconfig.ListApplicationsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListApplicationsOutput, error) {
					return &appconfig.ListApplicationsOutput{
						Items: []types.Application{{Id: aws.String("app-123"), Name: aws.String("test-app")}},
					}, nil
				},
				ListConfigurationProfilesFunc: func(ctx context.Context, params *appconfig.ListConfigurationProfilesInput, optFns ...func(*appconfig.Options)) (*appconfig.ListConfigurationProfilesOutput, error) {
					return &appconfig.ListConfigurationProfilesOutput{
						Items: []types.ConfigurationProfileSummary{{Id: aws.String("profile-123"), Name: aws.String("test-profile"), Type: aws.String("AWS.Freeform")}},
					}, nil
				},
				GetConfigurationProfileFunc: func(ctx context.Context, params *appconfig.GetConfigurationProfileInput, optFns ...func(*appconfig.Options)) (*appconfig.GetConfigurationProfileOutput, error) {
					return &appconfig.GetConfigurationProfileOutput{Id: aws.String("profile-123"), Type: aws.String("AWS.Freeform")}, nil
				},
				ListEnvironmentsFunc: func(ctx context.Context, params *appconfig.ListEnvironmentsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListEnvironmentsOutput, error) {
					return &appconfig.ListEnvironmentsOutput{
						Items: []types.Environment{{Id: aws.String("env-123"), Name: aws.String("test-env")}},
					}, nil
				},
				ListDeploymentStrategiesFunc: func(ctx context.Context, params *appconfig.ListDeploymentStrategiesInput, optFns ...func(*appconfig.Options)) (*appconfig.ListDeploymentStrategiesOutput, error) {
					return &appconfig.ListDeploymentStrategiesOutput{
						Items: []types.DeploymentStrategy{{Id: aws.String("strategy-123"), Name: aws.String("AppConfig.AllAtOnce")}},
					}, nil
				},
				ListDeploymentsFunc: func(ctx context.Context, params *appconfig.ListDeploymentsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListDeploymentsOutput, error) {
					return &appconfig.ListDeploymentsOutput{Items: []types.DeploymentSummary{}}, nil
				},
				CreateHostedConfigurationVersionFunc: func(ctx context.Context, params *appconfig.CreateHostedConfigurationVersionInput, optFns ...func(*appconfig.Options)) (*appconfig.CreateHostedConfigurationVersionOutput, error) {
					return &appconfig.CreateHostedConfigurationVersionOutput{VersionNumber: 1}, nil
				},
				StartDeploymentFunc: func(ctx context.Context, params *appconfig.StartDeploymentInput, optFns ...func(*appconfig.Options)) (*appconfig.StartDeploymentOutput, error) {
					return &appconfig.StartDeploymentOutput{DeploymentNumber: 1}, nil
				},
				GetDeploymentFunc: func(ctx context.Context, params *appconfig.GetDeploymentInput, optFns ...func(*appconfig.Options)) (*appconfig.GetDeploymentOutput, error) {
					var state types.DeploymentState
					if callCount < len(tt.mockStates) {
						state = tt.mockStates[callCount]
					} else {
						state = tt.mockStates[len(tt.mockStates)-1]
					}
					callCount++
					return &appconfig.GetDeploymentOutput{State: state}, nil
				},
			}

			deployerFactory := func(ctx context.Context, cfg *config.Config) (*Deployer, error) {
				awsClient := awsInternal.NewTestClient(mockClient)
				awsClient.PollingInterval = 100 * time.Millisecond // Fast polling for tests
				return NewWithClient(cfg, awsClient), nil
			}

			reporter := &reportertest.MockReporter{}
			executor := NewExecutorWithFactory(reporter, deployerFactory)

			opts := &Options{
				ConfigFile: configPath,
				WaitDeploy: tt.waitDeploy,
				WaitBake:   tt.waitBake,
				Timeout:    30,
			}

			err = executor.Execute(context.Background(), opts)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(reporter.TargetsCalls) != 1 {
				t.Fatalf("expected exactly 1 Targets call, got %d", len(reporter.TargetsCalls))
			}
			tc := reporter.TargetsCalls[0]
			if !tc.Closed {
				t.Errorf("expected Targets to be Close()d")
			}

			// SetProgress fires whenever AppConfig reports percentage; the
			// deploy sub-phase always emits at least one progress transition
			// even when AppConfig returns COMPLETE on the first poll
			// (MakeTargetsDeployTick pins to 1.0 in that case).
			progressCount := 0
			for _, tr := range tc.Transitions {
				if tr.Kind == "progress" {
					progressCount++
				}
			}
			if progressCount == 0 {
				t.Errorf("expected at least one progress transition; got: %+v", tc.Transitions)
			}

			// The done transition's verb encodes how far we waited.
			foundDone := false
			for _, tr := range tc.Transitions {
				if tr.Kind == "done" && strings.Contains(tr.Summary, tt.wantVerb) {
					foundDone = true
				}
			}
			if !foundDone {
				t.Errorf("expected Targets.Done with verb %q; got: %+v", tt.wantVerb, tc.Transitions)
			}

			// --wait-bake additionally swaps the row to a baking sub-phase
			// after the deploy reaches BAKING.
			seenBakingPhase := false
			for _, tr := range tc.Transitions {
				if tr.Kind == "phase" && tr.Phase == "baking" {
					seenBakingPhase = true
				}
			}
			if seenBakingPhase != tt.wantBakingPhase {
				t.Errorf("baking sub-phase seen = %v, want %v (transitions: %+v)", seenBakingPhase, tt.wantBakingPhase, tc.Transitions)
			}
		})
	}
}

// TestExecutorSkipsDeploymentWhenNoDiff tests that deployment is skipped when there are no changes
func TestExecutorSkipsDeploymentWhenNoDiff(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "executor-nodiff-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	configPath := filepath.Join(tempDir, "apcdeploy.yml")
	configContent := `application: test-app
configuration_profile: test-profile
environment: test-env
deployment_strategy: AppConfig.AllAtOnce
data_file: data.json
region: us-east-1
`
	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	// Create data file with same content as deployed version
	dataPath := filepath.Join(tempDir, "data.json")
	dataContent := []byte(`{"key": "value"}`)
	if err := os.WriteFile(dataPath, dataContent, 0o644); err != nil {
		t.Fatalf("Failed to write data: %v", err)
	}

	createVersionCalled := false
	startDeploymentCalled := false

	mockClient := &mock.MockAppConfigClient{
		ListApplicationsFunc: func(ctx context.Context, params *appconfig.ListApplicationsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListApplicationsOutput, error) {
			return &appconfig.ListApplicationsOutput{
				Items: []types.Application{{Id: aws.String("app-123"), Name: aws.String("test-app")}},
			}, nil
		},
		ListConfigurationProfilesFunc: func(ctx context.Context, params *appconfig.ListConfigurationProfilesInput, optFns ...func(*appconfig.Options)) (*appconfig.ListConfigurationProfilesOutput, error) {
			return &appconfig.ListConfigurationProfilesOutput{
				Items: []types.ConfigurationProfileSummary{{Id: aws.String("profile-123"), Name: aws.String("test-profile"), Type: aws.String("AWS.Freeform")}},
			}, nil
		},
		GetConfigurationProfileFunc: func(ctx context.Context, params *appconfig.GetConfigurationProfileInput, optFns ...func(*appconfig.Options)) (*appconfig.GetConfigurationProfileOutput, error) {
			return &appconfig.GetConfigurationProfileOutput{Id: aws.String("profile-123"), Type: aws.String("AWS.Freeform")}, nil
		},
		ListEnvironmentsFunc: func(ctx context.Context, params *appconfig.ListEnvironmentsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListEnvironmentsOutput, error) {
			return &appconfig.ListEnvironmentsOutput{
				Items: []types.Environment{{Id: aws.String("env-123"), Name: aws.String("test-env")}},
			}, nil
		},
		ListDeploymentStrategiesFunc: func(ctx context.Context, params *appconfig.ListDeploymentStrategiesInput, optFns ...func(*appconfig.Options)) (*appconfig.ListDeploymentStrategiesOutput, error) {
			return &appconfig.ListDeploymentStrategiesOutput{
				Items: []types.DeploymentStrategy{{Id: aws.String("strategy-123"), Name: aws.String("AppConfig.AllAtOnce")}},
			}, nil
		},
		ListDeploymentsFunc: func(ctx context.Context, params *appconfig.ListDeploymentsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListDeploymentsOutput, error) {
			// Return a completed deployment
			return &appconfig.ListDeploymentsOutput{
				Items: []types.DeploymentSummary{
					{
						DeploymentNumber:     1,
						State:                types.DeploymentStateComplete,
						ConfigurationVersion: aws.String("1"),
					},
				},
			}, nil
		},
		GetDeploymentFunc: func(ctx context.Context, params *appconfig.GetDeploymentInput, optFns ...func(*appconfig.Options)) (*appconfig.GetDeploymentOutput, error) {
			return &appconfig.GetDeploymentOutput{
				State:                  types.DeploymentStateComplete,
				ConfigurationProfileId: aws.String("profile-123"),
				ConfigurationVersion:   aws.String("1"),
			}, nil
		},
		GetHostedConfigurationVersionFunc: func(ctx context.Context, params *appconfig.GetHostedConfigurationVersionInput, optFns ...func(*appconfig.Options)) (*appconfig.GetHostedConfigurationVersionOutput, error) {
			// Return same content as local file (no diff)
			return &appconfig.GetHostedConfigurationVersionOutput{
				Content: dataContent,
			}, nil
		},
		CreateHostedConfigurationVersionFunc: func(ctx context.Context, params *appconfig.CreateHostedConfigurationVersionInput, optFns ...func(*appconfig.Options)) (*appconfig.CreateHostedConfigurationVersionOutput, error) {
			createVersionCalled = true
			return &appconfig.CreateHostedConfigurationVersionOutput{VersionNumber: 2}, nil
		},
		StartDeploymentFunc: func(ctx context.Context, params *appconfig.StartDeploymentInput, optFns ...func(*appconfig.Options)) (*appconfig.StartDeploymentOutput, error) {
			startDeploymentCalled = true
			return &appconfig.StartDeploymentOutput{DeploymentNumber: 2}, nil
		},
	}

	deployerFactory := func(ctx context.Context, cfg *config.Config) (*Deployer, error) {
		return NewWithClient(cfg, awsInternal.NewTestClient(mockClient)), nil
	}

	reporter := &reportertest.MockReporter{}
	executor := NewExecutorWithFactory(reporter, deployerFactory)

	opts := &Options{
		ConfigFile: configPath,
		WaitDeploy: false,
		WaitBake:   false,
		Timeout:    300,
		Force:      false,
	}

	err = executor.Execute(context.Background(), opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify that deployment was NOT started
	if createVersionCalled {
		t.Error("CreateHostedConfigurationVersion should not have been called when there are no changes")
	}
	if startDeploymentCalled {
		t.Error("StartDeployment should not have been called when there are no changes")
	}

	// The no-change branch finalises the Targets row via Skip("skipped (no
	// changes)") so the user sees the early-exit reason inline with the
	// target identifier. Asserting on the skip transition (not any line
	// containing the text) catches regressions where the early-exit gets
	// reported as Done or Fail.
	found := false
	for _, call := range reporter.TargetsCalls {
		for _, tr := range call.Transitions {
			if tr.Kind == "skip" && strings.Contains(tr.Reason, "no changes") {
				found = true
			}
		}
	}
	if !found {
		t.Errorf("expected Targets.Skip with 'no changes'; got: %+v", reporter.TargetsCalls)
	}
}

// TestExecutorForceDeploymentWithNoDiff tests that --force flag bypasses diff check
func TestExecutorForceDeploymentWithNoDiff(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "executor-force-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	configPath := filepath.Join(tempDir, "apcdeploy.yml")
	configContent := `application: test-app
configuration_profile: test-profile
environment: test-env
deployment_strategy: AppConfig.AllAtOnce
data_file: data.json
region: us-east-1
`
	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	// Create data file with same content as deployed version
	dataPath := filepath.Join(tempDir, "data.json")
	dataContent := []byte(`{"key": "value"}`)
	if err := os.WriteFile(dataPath, dataContent, 0o644); err != nil {
		t.Fatalf("Failed to write data: %v", err)
	}

	createVersionCalled := false
	startDeploymentCalled := false

	mockClient := &mock.MockAppConfigClient{
		ListApplicationsFunc: func(ctx context.Context, params *appconfig.ListApplicationsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListApplicationsOutput, error) {
			return &appconfig.ListApplicationsOutput{
				Items: []types.Application{{Id: aws.String("app-123"), Name: aws.String("test-app")}},
			}, nil
		},
		ListConfigurationProfilesFunc: func(ctx context.Context, params *appconfig.ListConfigurationProfilesInput, optFns ...func(*appconfig.Options)) (*appconfig.ListConfigurationProfilesOutput, error) {
			return &appconfig.ListConfigurationProfilesOutput{
				Items: []types.ConfigurationProfileSummary{{Id: aws.String("profile-123"), Name: aws.String("test-profile"), Type: aws.String("AWS.Freeform")}},
			}, nil
		},
		GetConfigurationProfileFunc: func(ctx context.Context, params *appconfig.GetConfigurationProfileInput, optFns ...func(*appconfig.Options)) (*appconfig.GetConfigurationProfileOutput, error) {
			return &appconfig.GetConfigurationProfileOutput{Id: aws.String("profile-123"), Type: aws.String("AWS.Freeform")}, nil
		},
		ListEnvironmentsFunc: func(ctx context.Context, params *appconfig.ListEnvironmentsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListEnvironmentsOutput, error) {
			return &appconfig.ListEnvironmentsOutput{
				Items: []types.Environment{{Id: aws.String("env-123"), Name: aws.String("test-env")}},
			}, nil
		},
		ListDeploymentStrategiesFunc: func(ctx context.Context, params *appconfig.ListDeploymentStrategiesInput, optFns ...func(*appconfig.Options)) (*appconfig.ListDeploymentStrategiesOutput, error) {
			return &appconfig.ListDeploymentStrategiesOutput{
				Items: []types.DeploymentStrategy{{Id: aws.String("strategy-123"), Name: aws.String("AppConfig.AllAtOnce")}},
			}, nil
		},
		ListDeploymentsFunc: func(ctx context.Context, params *appconfig.ListDeploymentsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListDeploymentsOutput, error) {
			// Return a completed deployment
			return &appconfig.ListDeploymentsOutput{
				Items: []types.DeploymentSummary{
					{
						DeploymentNumber:     1,
						State:                types.DeploymentStateComplete,
						ConfigurationVersion: aws.String("1"),
					},
				},
			}, nil
		},
		GetDeploymentFunc: func(ctx context.Context, params *appconfig.GetDeploymentInput, optFns ...func(*appconfig.Options)) (*appconfig.GetDeploymentOutput, error) {
			return &appconfig.GetDeploymentOutput{
				State:                  types.DeploymentStateComplete,
				ConfigurationProfileId: aws.String("profile-123"),
				ConfigurationVersion:   aws.String("1"),
			}, nil
		},
		GetHostedConfigurationVersionFunc: func(ctx context.Context, params *appconfig.GetHostedConfigurationVersionInput, optFns ...func(*appconfig.Options)) (*appconfig.GetHostedConfigurationVersionOutput, error) {
			// Return same content as local file (no diff)
			return &appconfig.GetHostedConfigurationVersionOutput{
				Content: dataContent,
			}, nil
		},
		CreateHostedConfigurationVersionFunc: func(ctx context.Context, params *appconfig.CreateHostedConfigurationVersionInput, optFns ...func(*appconfig.Options)) (*appconfig.CreateHostedConfigurationVersionOutput, error) {
			createVersionCalled = true
			return &appconfig.CreateHostedConfigurationVersionOutput{VersionNumber: 2}, nil
		},
		StartDeploymentFunc: func(ctx context.Context, params *appconfig.StartDeploymentInput, optFns ...func(*appconfig.Options)) (*appconfig.StartDeploymentOutput, error) {
			startDeploymentCalled = true
			return &appconfig.StartDeploymentOutput{DeploymentNumber: 2}, nil
		},
	}

	deployerFactory := func(ctx context.Context, cfg *config.Config) (*Deployer, error) {
		return NewWithClient(cfg, awsInternal.NewTestClient(mockClient)), nil
	}

	reporter := &reportertest.MockReporter{}
	executor := NewExecutorWithFactory(reporter, deployerFactory)

	opts := &Options{
		ConfigFile: configPath,
		WaitDeploy: false,
		WaitBake:   false,
		Timeout:    300,
		Force:      true, // Force deployment even without changes
	}

	err = executor.Execute(context.Background(), opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify that deployment WAS started even without changes
	if !createVersionCalled {
		t.Error("CreateHostedConfigurationVersion should have been called with --force flag")
	}
	if !startDeploymentCalled {
		t.Error("StartDeployment should have been called with --force flag")
	}

	// Verify deployment-started outcome on the Targets row. With --force we
	// skip the comparing phase but still finalise as Done("started …
	// deployment #2").
	found := false
	for _, call := range reporter.TargetsCalls {
		for _, tr := range call.Transitions {
			if tr.Kind == "done" && strings.Contains(tr.Summary, "deployment #2") {
				found = true
			}
		}
	}
	if !found {
		t.Errorf("expected Targets.Done summary mentioning 'deployment #2'; got: %+v", reporter.TargetsCalls)
	}
}

// TestExecutorWithOngoingDeployment tests error when deployment is in progress
func TestExecutorWithOngoingDeployment(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "executor-ongoing-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	configPath := filepath.Join(tempDir, "apcdeploy.yml")
	if err := os.WriteFile(configPath, []byte(`application: test-app
configuration_profile: test-profile
environment: test-env
data_file: data.json
region: us-east-1
`), 0o644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	dataPath := filepath.Join(tempDir, "data.json")
	if err := os.WriteFile(dataPath, []byte(`{"key": "value"}`), 0o644); err != nil {
		t.Fatalf("Failed to write data: %v", err)
	}

	mockClient := &mock.MockAppConfigClient{
		ListApplicationsFunc: func(ctx context.Context, params *appconfig.ListApplicationsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListApplicationsOutput, error) {
			return &appconfig.ListApplicationsOutput{
				Items: []types.Application{{Id: aws.String("app-123"), Name: aws.String("test-app")}},
			}, nil
		},
		ListConfigurationProfilesFunc: func(ctx context.Context, params *appconfig.ListConfigurationProfilesInput, optFns ...func(*appconfig.Options)) (*appconfig.ListConfigurationProfilesOutput, error) {
			return &appconfig.ListConfigurationProfilesOutput{
				Items: []types.ConfigurationProfileSummary{{Id: aws.String("profile-123"), Name: aws.String("test-profile"), Type: aws.String("AWS.Freeform")}},
			}, nil
		},
		GetConfigurationProfileFunc: func(ctx context.Context, params *appconfig.GetConfigurationProfileInput, optFns ...func(*appconfig.Options)) (*appconfig.GetConfigurationProfileOutput, error) {
			return &appconfig.GetConfigurationProfileOutput{Id: aws.String("profile-123"), Type: aws.String("AWS.Freeform")}, nil
		},
		ListEnvironmentsFunc: func(ctx context.Context, params *appconfig.ListEnvironmentsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListEnvironmentsOutput, error) {
			return &appconfig.ListEnvironmentsOutput{
				Items: []types.Environment{{Id: aws.String("env-123"), Name: aws.String("test-env")}},
			}, nil
		},
		ListDeploymentStrategiesFunc: func(ctx context.Context, params *appconfig.ListDeploymentStrategiesInput, optFns ...func(*appconfig.Options)) (*appconfig.ListDeploymentStrategiesOutput, error) {
			return &appconfig.ListDeploymentStrategiesOutput{
				Items: []types.DeploymentStrategy{{Id: aws.String("strategy-123"), Name: aws.String("AppConfig.AllAtOnce")}},
			}, nil
		},
		ListDeploymentsFunc: func(ctx context.Context, params *appconfig.ListDeploymentsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListDeploymentsOutput, error) {
			// Return an ongoing deployment
			return &appconfig.ListDeploymentsOutput{
				Items: []types.DeploymentSummary{
					{
						DeploymentNumber: 1,
						State:            types.DeploymentStateDeploying,
					},
				},
			}, nil
		},
	}

	deployerFactory := func(ctx context.Context, cfg *config.Config) (*Deployer, error) {
		awsClient := awsInternal.NewTestClient(mockClient)
		awsClient.PollingInterval = 100 * time.Millisecond // Fast polling for tests
		return NewWithClient(cfg, awsClient), nil
	}

	reporter := &reportertest.MockReporter{}
	executor := NewExecutorWithFactory(reporter, deployerFactory)

	opts := &Options{
		ConfigFile: configPath,
		WaitDeploy: false,
		WaitBake:   false,
		Timeout:    300,
	}

	err = executor.Execute(context.Background(), opts)

	if err == nil {
		t.Fatal("expected error for ongoing deployment")
	}

	if !strings.Contains(err.Error(), "deployment already in progress") {
		t.Errorf("expected 'deployment already in progress' error, got: %v", err)
	}
}
