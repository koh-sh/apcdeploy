package run

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/koh-sh/apcdeploy/internal/batch"
	"github.com/koh-sh/apcdeploy/internal/config"
	reportertest "github.com/koh-sh/apcdeploy/internal/reporter/testing"
)

// TestRunOnTarget_DeployerFactoryError covers the early-return path of
// RunOnTarget when the deployer factory fails (e.g. AWS client creation
// error). The TargetReporter must be moved to Fail before the helper
// returns so the orchestrator's row never stays in "running".
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

// TestRunOnTarget_ValidatesOptsBeforeAWS covers RunOnTarget's
// validateOpts call: invalid options must be rejected without making
// any AWS / file calls.
func TestRunOnTarget_ValidatesOptsBeforeAWS(t *testing.T) {
	t.Parallel()

	executor := NewExecutorWithFactory(&reportertest.MockReporter{},
		func(ctx context.Context, cfg *config.Config) (*Deployer, error) {
			t.Fatal("deployerFactory must not be called when opts are invalid")
			return nil, nil
		},
	)
	target := &batch.Target{
		Identifier: "us-east-1/app/profile/env",
		Config:     &config.Config{Region: "us-east-1"},
	}
	rep := &reportertest.MockReporter{}
	tg := rep.Targets([]string{target.Identifier})
	defer tg.Close()
	tr := batch.NewTargetReporter(tg, target.Identifier)

	err := executor.RunOnTarget(context.Background(), target, tr, &Options{
		WaitDeploy: true,
		WaitBake:   true,
	})
	if err == nil {
		t.Fatal("expected error for conflicting wait flags")
	}
}
