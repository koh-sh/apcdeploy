package edit

import (
	"context"
	"fmt"
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
	promptTesting "github.com/koh-sh/apcdeploy/internal/prompt/testing"
	reporterTesting "github.com/koh-sh/apcdeploy/internal/reporter/testing"
)

// fakeEditorScript writes a fake editor to a temp script that replaces the
// file contents with newContent. It also sets $EDITOR via t.Setenv.
func fakeEditorScript(t *testing.T, newContent string) {
	t.Helper()
	dir := t.TempDir()
	script := filepath.Join(dir, "fake-editor.sh")
	body := fmt.Sprintf(`#!/bin/sh
cat > "$1" <<'APCDEPLOY_EOF'
%s
APCDEPLOY_EOF
`, newContent)
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatalf("failed to write fake editor: %v", err)
	}
	t.Setenv("EDITOR", script)
}

// noChangeEditorScript configures an editor that leaves the file untouched.
func noChangeEditorScript(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	script := filepath.Join(dir, "noop-editor.sh")
	if err := os.WriteFile(script, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("failed to write noop editor: %v", err)
	}
	t.Setenv("EDITOR", script)
}

// baseMockClient returns a MockAppConfigClient pre-wired for a standard
// single-deployment happy path (Freeform JSON profile).
func baseMockClient(deployedContent []byte, contentType string) *mock.MockAppConfigClient {
	return &mock.MockAppConfigClient{
		ListApplicationsFunc: func(ctx context.Context, params *appconfig.ListApplicationsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListApplicationsOutput, error) {
			return &appconfig.ListApplicationsOutput{
				Items: []types.Application{{Id: aws.String("app-1"), Name: aws.String("test-app")}},
			}, nil
		},
		ListConfigurationProfilesFunc: func(ctx context.Context, params *appconfig.ListConfigurationProfilesInput, optFns ...func(*appconfig.Options)) (*appconfig.ListConfigurationProfilesOutput, error) {
			return &appconfig.ListConfigurationProfilesOutput{
				Items: []types.ConfigurationProfileSummary{
					{Id: aws.String("prof-1"), Name: aws.String("test-profile"), Type: aws.String("AWS.Freeform")},
				},
			}, nil
		},
		GetConfigurationProfileFunc: func(ctx context.Context, params *appconfig.GetConfigurationProfileInput, optFns ...func(*appconfig.Options)) (*appconfig.GetConfigurationProfileOutput, error) {
			return &appconfig.GetConfigurationProfileOutput{Id: aws.String("prof-1"), Name: aws.String("test-profile"), Type: aws.String("AWS.Freeform")}, nil
		},
		ListEnvironmentsFunc: func(ctx context.Context, params *appconfig.ListEnvironmentsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListEnvironmentsOutput, error) {
			return &appconfig.ListEnvironmentsOutput{
				Items: []types.Environment{{Id: aws.String("env-1"), Name: aws.String("test-env")}},
			}, nil
		},
		ListDeploymentStrategiesFunc: func(ctx context.Context, params *appconfig.ListDeploymentStrategiesInput, optFns ...func(*appconfig.Options)) (*appconfig.ListDeploymentStrategiesOutput, error) {
			return &appconfig.ListDeploymentStrategiesOutput{
				Items: []types.DeploymentStrategy{{Id: aws.String("strategy-1"), Name: aws.String("AppConfig.AllAtOnce")}},
			}, nil
		},
		ListDeploymentsFunc: func(ctx context.Context, params *appconfig.ListDeploymentsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListDeploymentsOutput, error) {
			return &appconfig.ListDeploymentsOutput{
				Items: []types.DeploymentSummary{
					{DeploymentNumber: 7, ConfigurationVersion: aws.String("3"), State: types.DeploymentStateComplete},
				},
			}, nil
		},
		GetDeploymentFunc: func(ctx context.Context, params *appconfig.GetDeploymentInput, optFns ...func(*appconfig.Options)) (*appconfig.GetDeploymentOutput, error) {
			return &appconfig.GetDeploymentOutput{
				DeploymentNumber:       7,
				ConfigurationProfileId: aws.String("prof-1"),
				ConfigurationVersion:   aws.String("3"),
				DeploymentStrategyId:   aws.String("strategy-inherited"),
				State:                  types.DeploymentStateComplete,
			}, nil
		},
		GetHostedConfigurationVersionFunc: func(ctx context.Context, params *appconfig.GetHostedConfigurationVersionInput, optFns ...func(*appconfig.Options)) (*appconfig.GetHostedConfigurationVersionOutput, error) {
			return &appconfig.GetHostedConfigurationVersionOutput{
				Content:     deployedContent,
				ContentType: aws.String(contentType),
			}, nil
		},
		CreateHostedConfigurationVersionFunc: func(ctx context.Context, params *appconfig.CreateHostedConfigurationVersionInput, optFns ...func(*appconfig.Options)) (*appconfig.CreateHostedConfigurationVersionOutput, error) {
			return &appconfig.CreateHostedConfigurationVersionOutput{VersionNumber: 4}, nil
		},
		StartDeploymentFunc: func(ctx context.Context, params *appconfig.StartDeploymentInput, optFns ...func(*appconfig.Options)) (*appconfig.StartDeploymentOutput, error) {
			return &appconfig.StartDeploymentOutput{DeploymentNumber: 8}, nil
		},
	}
}

func TestWorkflowHappyPath(t *testing.T) {
	fakeEditorScript(t, `{"key":"updated"}`)

	deployedContent := []byte(`{"key":"value"}`)
	client := baseMockClient(deployedContent, "application/json")

	var startedWithStrategy string
	client.StartDeploymentFunc = func(ctx context.Context, params *appconfig.StartDeploymentInput, optFns ...func(*appconfig.Options)) (*appconfig.StartDeploymentOutput, error) {
		startedWithStrategy = aws.ToString(params.DeploymentStrategyId)
		return &appconfig.StartDeploymentOutput{DeploymentNumber: 8}, nil
	}

	rep := &reporterTesting.MockReporter{}
	awsClient := awsInternal.NewTestClient(client)
	wf := newWorkflowWithClient(awsClient, &promptTesting.MockPrompter{}, rep)

	opts := &Options{
		Region:      "us-east-1",
		Application: "test-app",
		Profile:     "test-profile",
		Environment: "test-env",
		Timeout:     300,
	}

	if err := wf.Run(context.Background(), opts); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Inherited strategy should be used when --deployment-strategy is not provided.
	if startedWithStrategy != "strategy-inherited" {
		t.Errorf("expected inherited strategy, got %q", startedWithStrategy)
	}

	assertContainsMessage(t, rep.Messages, "Created configuration version 4")
	assertContainsMessage(t, rep.Messages, "Deployment #8 started")
}

func TestWorkflowErrorsWhenNoDeployment(t *testing.T) {
	noChangeEditorScript(t)

	client := baseMockClient([]byte(`{}`), "application/json")
	client.ListDeploymentsFunc = func(ctx context.Context, params *appconfig.ListDeploymentsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListDeploymentsOutput, error) {
		return &appconfig.ListDeploymentsOutput{Items: []types.DeploymentSummary{}}, nil
	}

	awsClient := awsInternal.NewTestClient(client)
	wf := newWorkflowWithClient(awsClient, &promptTesting.MockPrompter{}, &reporterTesting.MockReporter{})

	opts := &Options{
		Region:      "us-east-1",
		Application: "test-app",
		Profile:     "test-profile",
		Environment: "test-env",
		Timeout:     300,
	}

	err := wf.Run(context.Background(), opts)
	if err == nil {
		t.Fatal("expected error when no deployment exists")
	}
	if !strings.Contains(err.Error(), "no deployment found") {
		t.Errorf("expected 'no deployment found' error, got: %v", err)
	}
}

func TestWorkflowSkipsWhenNoChanges(t *testing.T) {
	deployedContent := []byte(`{"key":"value"}`)
	fakeEditorScript(t, `{"key":"value"}`)

	client := baseMockClient(deployedContent, "application/json")
	createCalled := false
	client.CreateHostedConfigurationVersionFunc = func(ctx context.Context, params *appconfig.CreateHostedConfigurationVersionInput, optFns ...func(*appconfig.Options)) (*appconfig.CreateHostedConfigurationVersionOutput, error) {
		createCalled = true
		return &appconfig.CreateHostedConfigurationVersionOutput{VersionNumber: 4}, nil
	}

	rep := &reporterTesting.MockReporter{}
	awsClient := awsInternal.NewTestClient(client)
	wf := newWorkflowWithClient(awsClient, &promptTesting.MockPrompter{}, rep)

	opts := &Options{
		Region:      "us-east-1",
		Application: "test-app",
		Profile:     "test-profile",
		Environment: "test-env",
		Timeout:     300,
	}

	if err := wf.Run(context.Background(), opts); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if createCalled {
		t.Error("CreateHostedConfigurationVersion should not be called when no changes")
	}
	assertContainsMessage(t, rep.Messages, "No changes detected")
}

func TestWorkflowRejectsInvalidJSON(t *testing.T) {
	fakeEditorScript(t, `{not valid json`)

	client := baseMockClient([]byte(`{"key":"value"}`), "application/json")
	awsClient := awsInternal.NewTestClient(client)
	wf := newWorkflowWithClient(awsClient, &promptTesting.MockPrompter{}, &reporterTesting.MockReporter{})

	opts := &Options{
		Region:      "us-east-1",
		Application: "test-app",
		Profile:     "test-profile",
		Environment: "test-env",
		Timeout:     300,
	}

	err := wf.Run(context.Background(), opts)
	if err == nil {
		t.Fatal("expected validation error for invalid JSON")
	}
	if !strings.Contains(err.Error(), "invalid JSON syntax") {
		t.Errorf("expected 'invalid JSON syntax' error, got: %v", err)
	}
}

func TestWorkflowFailsWhenOngoingDeployment(t *testing.T) {
	fakeEditorScript(t, `{"key":"updated"}`)

	client := baseMockClient([]byte(`{"key":"value"}`), "application/json")
	client.ListDeploymentsFunc = func(ctx context.Context, params *appconfig.ListDeploymentsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListDeploymentsOutput, error) {
		return &appconfig.ListDeploymentsOutput{
			Items: []types.DeploymentSummary{
				{DeploymentNumber: 7, ConfigurationVersion: aws.String("3"), State: types.DeploymentStateDeploying},
			},
		}, nil
	}

	awsClient := awsInternal.NewTestClient(client)
	wf := newWorkflowWithClient(awsClient, &promptTesting.MockPrompter{}, &reporterTesting.MockReporter{})

	opts := &Options{
		Region:      "us-east-1",
		Application: "test-app",
		Profile:     "test-profile",
		Environment: "test-env",
		Timeout:     300,
	}

	err := wf.Run(context.Background(), opts)
	if err == nil {
		t.Fatal("expected error when deployment in progress")
	}
	if !strings.Contains(err.Error(), "deployment already in progress") {
		t.Errorf("expected 'deployment already in progress' error, got: %v", err)
	}
}

func TestWorkflowUsesProvidedStrategyFlag(t *testing.T) {
	fakeEditorScript(t, `{"key":"updated"}`)

	client := baseMockClient([]byte(`{"key":"value"}`), "application/json")

	var startedWithStrategy string
	client.StartDeploymentFunc = func(ctx context.Context, params *appconfig.StartDeploymentInput, optFns ...func(*appconfig.Options)) (*appconfig.StartDeploymentOutput, error) {
		startedWithStrategy = aws.ToString(params.DeploymentStrategyId)
		return &appconfig.StartDeploymentOutput{DeploymentNumber: 8}, nil
	}

	awsClient := awsInternal.NewTestClient(client)
	wf := newWorkflowWithClient(awsClient, &promptTesting.MockPrompter{}, &reporterTesting.MockReporter{})

	opts := &Options{
		Region:             "us-east-1",
		Application:        "test-app",
		Profile:            "test-profile",
		Environment:        "test-env",
		DeploymentStrategy: "AppConfig.AllAtOnce",
		Timeout:            300,
	}

	if err := wf.Run(context.Background(), opts); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if startedWithStrategy != "strategy-1" {
		t.Errorf("expected resolved strategy 'strategy-1', got %q", startedWithStrategy)
	}
}

func TestWorkflowInteractiveSelection(t *testing.T) {
	fakeEditorScript(t, `{"key":"updated"}`)

	client := baseMockClient([]byte(`{"key":"value"}`), "application/json")
	awsClient := awsInternal.NewTestClient(client)

	selectCalls := 0
	prompter := &promptTesting.MockPrompter{
		SelectFunc: func(message string, options []string) (string, error) {
			selectCalls++
			if len(options) == 0 {
				return "", nil
			}
			return options[0], nil
		},
	}
	rep := &reporterTesting.MockReporter{}

	wf := newWorkflowWithClient(awsClient, prompter, rep)

	opts := &Options{
		Region:  "us-east-1",
		Timeout: 300,
	}

	if err := wf.Run(context.Background(), opts); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if selectCalls != 3 {
		t.Errorf("expected 3 interactive selections (app, profile, env), got %d", selectCalls)
	}
}

// assertContainsMessage fails the test if no reporter message contains substr.
func assertContainsMessage(t *testing.T, messages []string, substr string) {
	t.Helper()
	for _, m := range messages {
		if strings.Contains(m, substr) {
			return
		}
	}
	t.Errorf("expected message containing %q, got: %v", substr, messages)
}

// writeExec writes an executable script for use as a fake editor.
func writeExec(path, body string) error {
	return os.WriteFile(path, []byte(body), 0o755)
}

func TestNewWorkflowWithProvidedRegion(t *testing.T) {
	// With all flags provided, newWorkflow should succeed without touching AWS,
	// using awsConfig.LoadDefaultConfig. We just need it to construct a workflow.
	t.Setenv("AWS_EC2_METADATA_DISABLED", "true")

	opts := &Options{
		Region:      "us-east-1",
		Application: "app",
		Profile:     "prof",
		Environment: "env",
		Timeout:     300,
	}
	wf, err := newWorkflow(context.Background(), opts, &promptTesting.MockPrompter{}, &reporterTesting.MockReporter{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if wf == nil || wf.awsClient == nil {
		t.Fatal("expected non-nil workflow and aws client")
	}
}

func TestNewWorkflowTTYCheckFails(t *testing.T) {
	t.Parallel()

	prompter := &promptTesting.MockPrompter{
		CheckTTYFunc: func() error { return fmt.Errorf("no tty") },
	}
	// Omit region to trigger needsInteractive path.
	opts := &Options{Application: "app", Profile: "prof", Environment: "env", Timeout: 300}
	_, err := newWorkflow(context.Background(), opts, prompter, &reporterTesting.MockReporter{})
	if err == nil {
		t.Fatal("expected TTY error")
	}
	if !strings.Contains(err.Error(), "no tty") {
		t.Errorf("expected TTY error propagated, got: %v", err)
	}
}

func TestResolveStrategyIDErrorsWhenNoneAvailable(t *testing.T) {
	t.Parallel()

	client := &mock.MockAppConfigClient{}
	awsClient := awsInternal.NewTestClient(client)
	resolver := awsInternal.NewResolver(awsClient)

	_, err := resolveStrategyID(context.Background(), resolver, "", "")
	if err == nil {
		t.Fatal("expected error when no strategy provided or inherited")
	}
	if !strings.Contains(err.Error(), "could not determine deployment strategy") {
		t.Errorf("expected informative error, got: %v", err)
	}
}

func TestResolveStrategyIDFlagResolveFails(t *testing.T) {
	t.Parallel()

	client := &mock.MockAppConfigClient{
		ListDeploymentStrategiesFunc: func(ctx context.Context, params *appconfig.ListDeploymentStrategiesInput, optFns ...func(*appconfig.Options)) (*appconfig.ListDeploymentStrategiesOutput, error) {
			return nil, fmt.Errorf("boom")
		},
	}
	awsClient := awsInternal.NewTestClient(client)
	resolver := awsInternal.NewResolver(awsClient)

	_, err := resolveStrategyID(context.Background(), resolver, "MyStrategy", "")
	if err == nil {
		t.Fatal("expected error when strategy resolution fails")
	}
	if !strings.Contains(err.Error(), "failed to resolve deployment strategy") {
		t.Errorf("expected wrapped error, got: %v", err)
	}
}

func TestWaitIfRequested(t *testing.T) {
	t.Parallel()

	makeWorkflow := func(states []types.DeploymentState) (*workflow, *reporterTesting.MockReporter) {
		callCount := 0
		client := &mock.MockAppConfigClient{
			GetDeploymentFunc: func(ctx context.Context, params *appconfig.GetDeploymentInput, optFns ...func(*appconfig.Options)) (*appconfig.GetDeploymentOutput, error) {
				state := states[min(callCount, len(states)-1)]
				callCount++
				return &appconfig.GetDeploymentOutput{State: state}, nil
			},
		}
		awsClient := awsInternal.NewTestClient(client)
		awsClient.PollingInterval = 10 * time.Millisecond
		rep := &reporterTesting.MockReporter{}
		return newWorkflowWithClient(awsClient, &promptTesting.MockPrompter{}, rep), rep
	}

	t.Run("no wait emits warning", func(t *testing.T) {
		t.Parallel()
		wf, rep := makeWorkflow([]types.DeploymentState{types.DeploymentStateComplete})
		if err := wf.waitIfRequested(context.Background(), "app", "env", 5, &Options{Timeout: 1}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		assertContainsMessage(t, rep.Messages, "Deployment #5 is in progress")
	})

	t.Run("wait-deploy stops at baking", func(t *testing.T) {
		t.Parallel()
		wf, rep := makeWorkflow([]types.DeploymentState{types.DeploymentStateBaking})
		if err := wf.waitIfRequested(context.Background(), "app", "env", 5, &Options{WaitDeploy: true, Timeout: 2}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		assertContainsMessage(t, rep.Messages, "Deployment phase completed")
	})

	t.Run("wait-bake waits for complete", func(t *testing.T) {
		t.Parallel()
		wf, rep := makeWorkflow([]types.DeploymentState{types.DeploymentStateComplete})
		if err := wf.waitIfRequested(context.Background(), "app", "env", 5, &Options{WaitBake: true, Timeout: 2}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		assertContainsMessage(t, rep.Messages, "Deployment completed successfully")
	})

	t.Run("wait-deploy propagates error", func(t *testing.T) {
		t.Parallel()
		client := &mock.MockAppConfigClient{
			GetDeploymentFunc: func(ctx context.Context, params *appconfig.GetDeploymentInput, optFns ...func(*appconfig.Options)) (*appconfig.GetDeploymentOutput, error) {
				return nil, fmt.Errorf("boom")
			},
		}
		awsClient := awsInternal.NewTestClient(client)
		awsClient.PollingInterval = 10 * time.Millisecond
		wf := newWorkflowWithClient(awsClient, &promptTesting.MockPrompter{}, &reporterTesting.MockReporter{})
		err := wf.waitIfRequested(context.Background(), "app", "env", 5, &Options{WaitDeploy: true, Timeout: 1})
		if err == nil {
			t.Fatal("expected error from wait")
		}
		if !strings.Contains(err.Error(), "deployment failed") {
			t.Errorf("expected wrapped deployment error, got: %v", err)
		}
	})
}

func TestWorkflowInvalidSizeRejected(t *testing.T) {
	dir := t.TempDir()
	script := dir + "/big-editor.sh"
	body := "#!/bin/sh\nprintf 'A%.0s' $(seq 1 2200000) > \"$1\"\n"
	if err := writeExec(script, body); err != nil {
		t.Fatalf("failed to write editor: %v", err)
	}
	t.Setenv("EDITOR", script)

	client := baseMockClient([]byte(`hello`), config.ContentTypeText)
	awsClient := awsInternal.NewTestClient(client)
	wf := newWorkflowWithClient(awsClient, &promptTesting.MockPrompter{}, &reporterTesting.MockReporter{})

	err := wf.Run(context.Background(), &Options{
		Region:      "us-east-1",
		Application: "test-app",
		Profile:     "test-profile",
		Environment: "test-env",
		Timeout:     300,
	})
	if err == nil {
		t.Fatal("expected size validation error")
	}
	if !strings.Contains(err.Error(), "exceeds maximum") {
		t.Errorf("expected size error, got: %v", err)
	}
}

func TestWorkflowResolveProfileError(t *testing.T) {
	fakeEditorScript(t, `{"key":"updated"}`)

	client := baseMockClient([]byte(`{"key":"value"}`), "application/json")
	client.GetConfigurationProfileFunc = func(ctx context.Context, params *appconfig.GetConfigurationProfileInput, optFns ...func(*appconfig.Options)) (*appconfig.GetConfigurationProfileOutput, error) {
		return nil, fmt.Errorf("boom")
	}
	awsClient := awsInternal.NewTestClient(client)
	wf := newWorkflowWithClient(awsClient, &promptTesting.MockPrompter{}, &reporterTesting.MockReporter{})

	err := wf.Run(context.Background(), &Options{
		Region: "us-east-1", Application: "test-app", Profile: "test-profile",
		Environment: "test-env", Timeout: 300,
	})
	if err == nil {
		t.Fatal("expected error from profile resolution")
	}
	if !strings.Contains(err.Error(), "failed to resolve configuration profile") {
		t.Errorf("expected wrapped profile error, got: %v", err)
	}
}

func TestWorkflowGetDeployedConfigError(t *testing.T) {
	fakeEditorScript(t, `{"key":"updated"}`)

	client := baseMockClient([]byte(`{"key":"value"}`), "application/json")
	client.GetHostedConfigurationVersionFunc = func(ctx context.Context, params *appconfig.GetHostedConfigurationVersionInput, optFns ...func(*appconfig.Options)) (*appconfig.GetHostedConfigurationVersionOutput, error) {
		return nil, fmt.Errorf("boom")
	}
	awsClient := awsInternal.NewTestClient(client)
	wf := newWorkflowWithClient(awsClient, &promptTesting.MockPrompter{}, &reporterTesting.MockReporter{})

	err := wf.Run(context.Background(), &Options{
		Region: "us-east-1", Application: "test-app", Profile: "test-profile",
		Environment: "test-env", Timeout: 300,
	})
	if err == nil {
		t.Fatal("expected error fetching deployed configuration")
	}
	if !strings.Contains(err.Error(), "failed to get latest deployed configuration") {
		t.Errorf("expected wrapped fetch error, got: %v", err)
	}
}

func TestWorkflowCreateVersionValidationError(t *testing.T) {
	fakeEditorScript(t, `{"key":"updated"}`)

	client := baseMockClient([]byte(`{"key":"value"}`), "application/json")
	msg := "JSON Schema validation failed"
	client.CreateHostedConfigurationVersionFunc = func(ctx context.Context, params *appconfig.CreateHostedConfigurationVersionInput, optFns ...func(*appconfig.Options)) (*appconfig.CreateHostedConfigurationVersionOutput, error) {
		return nil, &types.BadRequestException{Message: &msg}
	}
	awsClient := awsInternal.NewTestClient(client)
	wf := newWorkflowWithClient(awsClient, &promptTesting.MockPrompter{}, &reporterTesting.MockReporter{})

	err := wf.Run(context.Background(), &Options{
		Region: "us-east-1", Application: "test-app", Profile: "test-profile",
		Environment: "test-env", Timeout: 300,
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
	// Validation errors are returned via FormatValidationError without fmt wrapping.
	if !strings.Contains(err.Error(), "Configuration validation failed") {
		t.Errorf("expected formatted validation error, got: %v", err)
	}
}

func TestWorkflowStartDeploymentError(t *testing.T) {
	fakeEditorScript(t, `{"key":"updated"}`)

	client := baseMockClient([]byte(`{"key":"value"}`), "application/json")
	client.StartDeploymentFunc = func(ctx context.Context, params *appconfig.StartDeploymentInput, optFns ...func(*appconfig.Options)) (*appconfig.StartDeploymentOutput, error) {
		return nil, fmt.Errorf("boom")
	}
	awsClient := awsInternal.NewTestClient(client)
	wf := newWorkflowWithClient(awsClient, &promptTesting.MockPrompter{}, &reporterTesting.MockReporter{})

	err := wf.Run(context.Background(), &Options{
		Region: "us-east-1", Application: "test-app", Profile: "test-profile",
		Environment: "test-env", Timeout: 300,
	})
	if err == nil {
		t.Fatal("expected start deployment error")
	}
	if !strings.Contains(err.Error(), "failed to start deployment") {
		t.Errorf("expected wrapped start error, got: %v", err)
	}
}

// TestWorkflowFeatureFlagsIgnoresMetadata verifies profile type is propagated so
// FeatureFlags timestamp metadata is normalized away.
func TestWorkflowFeatureFlagsIgnoresMetadata(t *testing.T) {
	deployed := []byte(`{"flags":{"f":{"name":"f","_createdAt":"2024-01-01","_updatedAt":"2024-01-02"}},"version":"1"}`)
	fakeEditorScript(t, `{"flags":{"f":{"name":"f"}},"version":"1"}`)

	client := &mock.MockAppConfigClient{
		ListApplicationsFunc: func(ctx context.Context, params *appconfig.ListApplicationsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListApplicationsOutput, error) {
			return &appconfig.ListApplicationsOutput{
				Items: []types.Application{{Id: aws.String("app-1"), Name: aws.String("test-app")}},
			}, nil
		},
		ListConfigurationProfilesFunc: func(ctx context.Context, params *appconfig.ListConfigurationProfilesInput, optFns ...func(*appconfig.Options)) (*appconfig.ListConfigurationProfilesOutput, error) {
			return &appconfig.ListConfigurationProfilesOutput{
				Items: []types.ConfigurationProfileSummary{{Id: aws.String("prof-1"), Name: aws.String("test-profile"), Type: aws.String(config.ProfileTypeFeatureFlags)}},
			}, nil
		},
		GetConfigurationProfileFunc: func(ctx context.Context, params *appconfig.GetConfigurationProfileInput, optFns ...func(*appconfig.Options)) (*appconfig.GetConfigurationProfileOutput, error) {
			return &appconfig.GetConfigurationProfileOutput{Id: aws.String("prof-1"), Name: aws.String("test-profile"), Type: aws.String(config.ProfileTypeFeatureFlags)}, nil
		},
		ListEnvironmentsFunc: func(ctx context.Context, params *appconfig.ListEnvironmentsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListEnvironmentsOutput, error) {
			return &appconfig.ListEnvironmentsOutput{
				Items: []types.Environment{{Id: aws.String("env-1"), Name: aws.String("test-env")}},
			}, nil
		},
		ListDeploymentsFunc: func(ctx context.Context, params *appconfig.ListDeploymentsInput, optFns ...func(*appconfig.Options)) (*appconfig.ListDeploymentsOutput, error) {
			return &appconfig.ListDeploymentsOutput{
				Items: []types.DeploymentSummary{
					{DeploymentNumber: 1, ConfigurationVersion: aws.String("1"), State: types.DeploymentStateComplete},
				},
			}, nil
		},
		GetDeploymentFunc: func(ctx context.Context, params *appconfig.GetDeploymentInput, optFns ...func(*appconfig.Options)) (*appconfig.GetDeploymentOutput, error) {
			return &appconfig.GetDeploymentOutput{
				DeploymentNumber:       1,
				ConfigurationProfileId: aws.String("prof-1"),
				ConfigurationVersion:   aws.String("1"),
				DeploymentStrategyId:   aws.String("strategy-1"),
				State:                  types.DeploymentStateComplete,
			}, nil
		},
		GetHostedConfigurationVersionFunc: func(ctx context.Context, params *appconfig.GetHostedConfigurationVersionInput, optFns ...func(*appconfig.Options)) (*appconfig.GetHostedConfigurationVersionOutput, error) {
			return &appconfig.GetHostedConfigurationVersionOutput{
				Content:     deployed,
				ContentType: aws.String(config.ContentTypeJSON),
			}, nil
		},
	}

	awsClient := awsInternal.NewTestClient(client)
	rep := &reporterTesting.MockReporter{}
	wf := newWorkflowWithClient(awsClient, &promptTesting.MockPrompter{}, rep)

	err := wf.Run(context.Background(), &Options{
		Region:      "us-east-1",
		Application: "test-app",
		Profile:     "test-profile",
		Environment: "test-env",
		Timeout:     300,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertContainsMessage(t, rep.Messages, "No changes detected")
}
