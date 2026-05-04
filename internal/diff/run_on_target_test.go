package diff

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

	payload, hasChanges, err := executor.RunOnTarget(context.Background(), target, tr)
	if err == nil {
		t.Fatal("expected error from RunOnTarget when clientFactory fails")
	}
	if !errors.Is(err, failure) {
		t.Errorf("err = %v, want wrapping %v", err, failure)
	}
	if payload != nil {
		t.Errorf("payload = %q, want nil on factory failure", payload)
	}
	if hasChanges {
		t.Errorf("hasChanges = true, want false on factory failure")
	}
}
