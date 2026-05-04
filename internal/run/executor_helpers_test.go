package run

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/koh-sh/apcdeploy/internal/config"
	reportertest "github.com/koh-sh/apcdeploy/internal/reporter/testing"
)

// TestExecutorDeployerFactoryError exercises the early-return path where the
// deployer factory itself fails before any AWS interaction. This isolates the
// error wrapping (`failed to create deployer: ...`) from the resource-resolution
// path so a regression in the wrapper text fails this test rather than the
// catch-all happy-path test further downstream.
func TestExecutorDeployerFactoryError(t *testing.T) {
	t.Parallel()

	tempDir, err := os.MkdirTemp("", "executor-factory-err-*")
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
	if err := os.WriteFile(filepath.Join(tempDir, "data.json"), []byte(`{}`), 0o644); err != nil {
		t.Fatalf("Failed to write data: %v", err)
	}

	factoryErr := errors.New("simulated factory failure")
	factory := func(_ context.Context, _ *config.Config) (*Deployer, error) {
		return nil, factoryErr
	}

	rep := &reportertest.MockReporter{}
	executor := NewExecutorWithFactory(rep, factory)

	err = executor.Execute(context.Background(), &Options{
		ConfigFile: configPath,
		Timeout:    300,
	})
	if err == nil {
		t.Fatal("expected factory error to propagate")
	}
	if !strings.Contains(err.Error(), "failed to create deployer") {
		t.Errorf("expected 'failed to create deployer' wrapper, got: %v", err)
	}
	if !errors.Is(err, factoryErr) {
		t.Errorf("expected wrapped factory error, got: %v", err)
	}
}
