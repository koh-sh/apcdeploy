package pull

import (
	"context"
	"errors"
	"testing"

	awsInternal "github.com/koh-sh/apcdeploy/internal/aws"
	"github.com/koh-sh/apcdeploy/internal/batch"
	"github.com/koh-sh/apcdeploy/internal/config"
	reportertest "github.com/koh-sh/apcdeploy/internal/reporter/testing"
)

// TestRunOnTarget_ClientFactoryError covers the early-return path of
// RunOnTarget when the AWS client factory itself fails. The TargetReporter
// must be moved to Fail before the helper returns, so the orchestrator's
// row never stays in "running".
func TestRunOnTarget_ClientFactoryError(t *testing.T) {
	t.Parallel()

	failure := errors.New("creds boom")
	factory := func(ctx context.Context, region string) (*awsInternal.Client, error) {
		return nil, failure
	}

	rep := &reportertest.MockReporter{}
	executor := NewExecutorWithFactory(rep, factory)

	target := &batch.Target{
		Path: "test.yml",
		Config: &config.Config{
			Region:               "us-east-1",
			Application:          "app",
			ConfigurationProfile: "profile",
			Environment:          "env",
			DataFile:             "/tmp/does-not-matter",
		},
		Identifier: "us-east-1/app/profile/env",
	}

	tg := rep.Targets([]string{target.Identifier})
	defer tg.Close()
	tr := batch.NewTargetReporter(tg, target.Identifier)

	err := executor.RunOnTarget(context.Background(), target, tr)
	if err == nil {
		t.Fatal("expected error from RunOnTarget when clientFactory fails")
	}
	if !errors.Is(err, failure) {
		t.Errorf("err = %v, want wrapping %v", err, failure)
	}

	// Targets should have been advanced to a Fail terminal state.
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
