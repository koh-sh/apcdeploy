package run

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/koh-sh/apcdeploy/internal/batch"
	"github.com/koh-sh/apcdeploy/internal/config"
	reportertest "github.com/koh-sh/apcdeploy/internal/reporter/testing"
)

// TestRunOnTarget_DeployerFactoryError covers the early-return path of
// RunOnTarget when the deployer factory fails (e.g. AWS client creation
// error). The TargetReporter must be moved to Fail before the helper
// returns so the orchestrator's row never stays in "running", and the
// error must be wrapped with "failed to create deployer:" so callers
// can pattern-match on the wrapper text.
func TestRunOnTarget_DeployerFactoryError(t *testing.T) {
	t.Parallel()

	failure := errors.New("creds boom")
	factory := func(ctx context.Context, cfg *config.Config) (*Deployer, error) {
		return nil, failure
	}

	rep := &reportertest.MockReporter{}
	executor := NewExecutorWithFactory(rep, factory)

	// Need an actual data file so RunOnTarget gets past the os.ReadFile
	// step before reaching deployerFactory.
	dataPath := t.TempDir() + "/data.json"
	if err := os.WriteFile(dataPath, []byte(`{}`), 0o644); err != nil {
		t.Fatalf("write data: %v", err)
	}

	target := &batch.Target{
		Path: "test.yml",
		Config: &config.Config{
			Region:               "us-east-1",
			Application:          "app",
			ConfigurationProfile: "profile",
			Environment:          "env",
			DataFile:             dataPath,
		},
		Identifier: "us-east-1/app/profile/env",
	}

	tg := rep.Targets([]string{target.Identifier})
	defer tg.Close()
	tr := batch.NewTargetReporter(tg, target.Identifier)

	err := executor.RunOnTarget(context.Background(), target, tr, &Options{Timeout: 30})
	if err == nil {
		t.Fatal("expected error from RunOnTarget when deployerFactory fails")
	}
	if !errors.Is(err, failure) {
		t.Errorf("err = %v, want wrapping %v", err, failure)
	}
	if !strings.Contains(err.Error(), "failed to create deployer") {
		t.Errorf("expected 'failed to create deployer' wrapper, got: %v", err)
	}

	var sawFail bool
	for _, call := range rep.TargetsCalls {
		for _, tr := range call.Transitions {
			if tr.Kind == "fail" && tr.ID == target.Identifier {
				sawFail = true
			}
		}
	}
	if !sawFail {
		t.Errorf("expected Targets.Fail for the row; transitions: %+v", rep.TargetsCalls)
	}
}

// TestRunOnTarget_DataFileReadError covers the early-return path when
// the local data file cannot be read. The error must wrap with
// "failed to read data file" and finalise the Targets row to Fail.
func TestRunOnTarget_DataFileReadError(t *testing.T) {
	t.Parallel()

	rep := &reportertest.MockReporter{}
	executor := NewExecutorWithFactory(rep,
		func(ctx context.Context, cfg *config.Config) (*Deployer, error) {
			t.Fatal("deployerFactory must not be called when data-file load fails")
			return nil, nil
		},
	)

	target := &batch.Target{
		Path: "test.yml",
		Config: &config.Config{
			Region:               "us-east-1",
			Application:          "app",
			ConfigurationProfile: "profile",
			Environment:          "env",
			DataFile:             filepath.Join(t.TempDir(), "does-not-exist.json"),
		},
		Identifier: "us-east-1/app/profile/env",
	}
	tg := rep.Targets([]string{target.Identifier})
	defer tg.Close()
	tr := batch.NewTargetReporter(tg, target.Identifier)

	err := executor.RunOnTarget(context.Background(), target, tr, &Options{Timeout: 30})
	if err == nil {
		t.Fatal("expected error from RunOnTarget when data file is missing")
	}
	if !strings.Contains(err.Error(), "failed to read data file") {
		t.Errorf("expected 'failed to read data file' wrapper, got: %v", err)
	}

	var sawFail bool
	for _, call := range rep.TargetsCalls {
		for _, tr := range call.Transitions {
			if tr.Kind == "fail" && tr.ID == target.Identifier {
				sawFail = true
			}
		}
	}
	if !sawFail {
		t.Errorf("expected Targets.Fail for the row; transitions: %+v", rep.TargetsCalls)
	}
}
