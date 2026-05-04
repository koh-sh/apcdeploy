package batch

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/koh-sh/apcdeploy/internal/config"
	"github.com/koh-sh/apcdeploy/internal/reporter"
	reportertesting "github.com/koh-sh/apcdeploy/internal/reporter/testing"
)

func makeTargets(ids ...string) []*Target {
	out := make([]*Target, len(ids))
	for i, id := range ids {
		out[i] = &Target{
			Path:       id + ".yml",
			Identifier: id,
			Config:     &config.Config{},
		}
	}
	return out
}

func TestOrchestrator_AllSucceed(t *testing.T) {
	t.Parallel()

	rep := &reportertesting.MockReporter{}
	targets := makeTargets("a", "b", "c")

	o := &Orchestrator{
		Targets:  targets,
		Reporter: rep,
		Execute: func(ctx context.Context, tgt *Target, tr reporter.TargetReporter) error {
			tr.Done("done " + tgt.Identifier)
			return nil
		},
	}
	summary, err := o.Run(context.Background())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if summary.OK != 3 || summary.NoOp != 0 || summary.Failed != 0 {
		t.Errorf("summary = %+v, want OK=3", summary)
	}
	if len(rep.TargetsCalls) != 1 || !rep.TargetsCalls[0].Closed {
		t.Errorf("Targets must be opened once and Closed; got %+v", rep.TargetsCalls)
	}
}

func TestOrchestrator_NoOpCounted(t *testing.T) {
	t.Parallel()

	rep := &reportertesting.MockReporter{}
	targets := makeTargets("a", "b")

	o := &Orchestrator{
		Targets:  targets,
		Reporter: rep,
		Execute: func(ctx context.Context, tgt *Target, tr reporter.TargetReporter) error {
			tr.Skip("no changes")
			return nil
		},
	}
	summary, err := o.Run(context.Background())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if summary.OK != 0 || summary.NoOp != 2 || summary.Failed != 0 {
		t.Errorf("summary = %+v, want NoOp=2", summary)
	}
}

func TestOrchestrator_FailFastSkipsRemaining(t *testing.T) {
	t.Parallel()

	rep := &reportertesting.MockReporter{}
	targets := makeTargets("a", "b", "c", "d")

	// Block "a" so the failure (from "b") propagates while "c"/"d" are
	// still queued. Parallel=2 means at most a+b run concurrently;
	// c/d should never start.
	gateA := make(chan struct{})
	var executed atomic.Int32

	o := &Orchestrator{
		Targets:  targets,
		Parallel: 2,
		Reporter: rep,
		Execute: func(ctx context.Context, tgt *Target, tr reporter.TargetReporter) error {
			executed.Add(1)
			switch tgt.Identifier {
			case "a":
				<-gateA
				tr.Done("ok")
				return nil
			case "b":
				err := errors.New("boom")
				tr.Fail(err)
				return err
			default:
				t.Errorf("target %s should not have executed under fail-fast", tgt.Identifier)
				tr.Done("unexpected")
				return nil
			}
		},
	}

	doneCh := make(chan struct{})
	var summary Summary
	var runErr error
	go func() {
		summary, runErr = o.Run(context.Background())
		close(doneCh)
	}()

	// Wait for "b" to fail.
	deadline := time.Now().Add(2 * time.Second)
	for executed.Load() < 2 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	// Release "a" so the orchestrator can finish.
	close(gateA)
	<-doneCh

	if runErr == nil {
		t.Fatal("Run should return non-nil error when a target fails")
	}
	if summary.Failed != 1 {
		t.Errorf("summary.Failed = %d, want 1", summary.Failed)
	}
	if summary.Skipped != 2 {
		t.Errorf("summary.Skipped = %d, want 2 (c,d skipped under fail-fast)", summary.Skipped)
	}
	if executed.Load() != 2 {
		t.Errorf("executed = %d, want 2 (only a,b executed)", executed.Load())
	}
}

func TestOrchestrator_ContinueOnErrorRunsAll(t *testing.T) {
	t.Parallel()

	rep := &reportertesting.MockReporter{}
	targets := makeTargets("a", "b", "c")

	o := &Orchestrator{
		Targets:         targets,
		Parallel:        1,
		ContinueOnError: true,
		Reporter:        rep,
		Execute: func(ctx context.Context, tgt *Target, tr reporter.TargetReporter) error {
			if tgt.Identifier == "b" {
				err := errors.New("only b fails")
				tr.Fail(err)
				return err
			}
			tr.Done("ok")
			return nil
		},
	}
	summary, err := o.Run(context.Background())
	if err == nil {
		t.Fatal("Run should still return error so exit code is non-zero")
	}
	if summary.OK != 2 || summary.Failed != 1 || summary.Skipped != 0 {
		t.Errorf("summary = %+v, want OK=2 Failed=1 Skipped=0", summary)
	}
	if len(summary.Errors) != 1 || summary.Errors[0].Identifier != "b" {
		t.Errorf("summary.Errors = %+v, want one entry for b", summary.Errors)
	}
}

func TestOrchestrator_PreservesArgumentStartOrder(t *testing.T) {
	t.Parallel()

	rep := &reportertesting.MockReporter{}
	targets := makeTargets("a", "b", "c", "d", "e")

	var (
		mu      sync.Mutex
		started []string
	)
	o := &Orchestrator{
		Targets:  targets,
		Parallel: 1, // serial — start order == finish order
		Reporter: rep,
		Execute: func(ctx context.Context, tgt *Target, tr reporter.TargetReporter) error {
			mu.Lock()
			started = append(started, tgt.Identifier)
			mu.Unlock()
			tr.Done("ok")
			return nil
		},
	}
	if _, err := o.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}
	want := []string{"a", "b", "c", "d", "e"}
	if len(started) != len(want) {
		t.Fatalf("started = %v, want %v", started, want)
	}
	for i := range want {
		if started[i] != want[i] {
			t.Errorf("started[%d] = %q, want %q", i, started[i], want[i])
		}
	}
}

func TestOrchestrator_FailureWithoutFailCallStillCounted(t *testing.T) {
	t.Parallel()

	// An executor that returns an error without calling tr.Fail is a
	// contract violation, but the orchestrator must still record the
	// failure and surface the error so the row doesn't render forever.
	rep := &reportertesting.MockReporter{}
	targets := makeTargets("a")

	o := &Orchestrator{
		Targets:  targets,
		Reporter: rep,
		Execute: func(ctx context.Context, tgt *Target, tr reporter.TargetReporter) error {
			return errors.New("naked error")
		},
	}
	summary, err := o.Run(context.Background())
	if err == nil {
		t.Fatal("Run should propagate executor error")
	}
	if summary.Failed != 1 {
		t.Errorf("summary.Failed = %d, want 1", summary.Failed)
	}
}

func TestOrchestrator_DefaultParallelIsAll(t *testing.T) {
	t.Parallel()

	rep := &reportertesting.MockReporter{}
	targets := makeTargets("a", "b", "c", "d")

	var concurrent atomic.Int32
	var maxSeen atomic.Int32

	o := &Orchestrator{
		Targets:  targets,
		Reporter: rep,
		// Parallel left as 0 → defaults to len(Targets).
		Execute: func(ctx context.Context, tgt *Target, tr reporter.TargetReporter) error {
			cur := concurrent.Add(1)
			defer concurrent.Add(-1)
			for {
				prev := maxSeen.Load()
				if cur <= prev || maxSeen.CompareAndSwap(prev, cur) {
					break
				}
			}
			time.Sleep(20 * time.Millisecond)
			tr.Done("ok")
			return nil
		},
	}
	if _, err := o.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if maxSeen.Load() < 2 {
		t.Errorf("maxSeen = %d, want >= 2 (default Parallel should run targets concurrently)", maxSeen.Load())
	}
}
