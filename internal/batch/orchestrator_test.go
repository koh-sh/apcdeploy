package batch

import (
	"context"
	"errors"
	"fmt"
	"strings"
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
	bFailed := make(chan struct{})
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
				close(bFailed)
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

	// Wait for "b" to actually finish failing — ensures the orchestrator
	// has set the fail-fast flag before we release "a".
	select {
	case <-bFailed:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for target b to fail")
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

	var startBarrier sync.WaitGroup
	releaseGate := make(chan struct{})
	startBarrier.Add(len(targets))

	o := &Orchestrator{
		Targets:  targets,
		Reporter: rep,
		// Parallel left as 0 → defaults to len(Targets).
		Execute: func(ctx context.Context, tgt *Target, tr reporter.TargetReporter) error {
			// Each goroutine signals it has started, then blocks on a shared
			// gate. The gate is only released after all len(targets) goroutines
			// have signalled — so if the orchestrator failed to launch all of
			// them concurrently, the test deadlocks (and times out via the
			// watchdog below) instead of passing on a weak `>= 2` threshold.
			startBarrier.Done()
			<-releaseGate
			tr.Done("ok")
			return nil
		},
	}

	runDone := make(chan struct {
		summary Summary
		err     error
	}, 1)
	go func() {
		summary, err := o.Run(context.Background())
		runDone <- struct {
			summary Summary
			err     error
		}{summary, err}
	}()

	// Wait until every target has reached Execute. The watchdog catches
	// the case where the orchestrator only fanned out to a subset.
	allStarted := make(chan struct{})
	go func() {
		startBarrier.Wait()
		close(allStarted)
	}()
	select {
	case <-allStarted:
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for all targets to start concurrently — default Parallel did not fan out to len(Targets)")
	}
	close(releaseGate)

	res := <-runDone
	if res.err != nil {
		t.Fatalf("Run: %v", res.err)
	}
	if res.summary.OK != len(targets) {
		t.Errorf("summary.OK = %d, want %d", res.summary.OK, len(targets))
	}
}

// errSentinel is a package-local sentinel used to assert that
// Orchestrator.Run preserves errors.Is matching through whatever wrapper
// it returns. The actual production sentinel (aws.ErrNoDeployment) lives
// in a higher layer; using a local sentinel keeps this test free of that
// dependency.
var errSentinel = errors.New("sentinel")

// TestOrchestrator_PreservesSentinelSingleFailure asserts errors.Is finds
// a wrapped sentinel when exactly one target fails. This is the most
// load-bearing case in production: cmd/root.go maps
// aws.ErrNoDeployment to exit code 2 via errors.Is on the orchestrator's
// return value.
func TestOrchestrator_PreservesSentinelSingleFailure(t *testing.T) {
	t.Parallel()

	rep := &reportertesting.MockReporter{}
	targets := makeTargets("a")
	o := &Orchestrator{
		Targets:  targets,
		Reporter: rep,
		Execute: func(_ context.Context, _ *Target, tr reporter.TargetReporter) error {
			wrapped := fmt.Errorf("wrapped: %w", errSentinel)
			tr.Fail(wrapped)
			return wrapped
		},
	}
	_, err := o.Run(context.Background())
	if err == nil {
		t.Fatal("Run should return non-nil error")
	}
	if !errors.Is(err, errSentinel) {
		t.Errorf("errors.Is(err, sentinel) = false; err = %v", err)
	}
}

// TestOrchestrator_PreservesSentinelAmongMultipleFailures asserts a
// sentinel buried in any one of several failing targets is still found
// by errors.Is. Without Unwrap() []error this would silently regress to
// "only the first error matches".
func TestOrchestrator_PreservesSentinelAmongMultipleFailures(t *testing.T) {
	t.Parallel()

	rep := &reportertesting.MockReporter{}
	targets := makeTargets("a", "b", "c")
	o := &Orchestrator{
		Targets:         targets,
		ContinueOnError: true,
		Reporter:        rep,
		Execute: func(_ context.Context, tgt *Target, tr reporter.TargetReporter) error {
			var wrapped error
			switch tgt.Identifier {
			case "a":
				wrapped = errors.New("a failed")
			case "b":
				wrapped = fmt.Errorf("b: %w", errSentinel)
			case "c":
				wrapped = errors.New("c failed")
			}
			tr.Fail(wrapped)
			return wrapped
		},
	}
	_, err := o.Run(context.Background())
	if err == nil {
		t.Fatal("Run should return non-nil error")
	}
	if !errors.Is(err, errSentinel) {
		t.Errorf("errors.Is found no sentinel among multiple failures; err = %v", err)
	}
}

// TestOrchestrator_AggregateErrorErrorIsSingleLine asserts the top-level
// Error string is a one-line summary so cmd/root.go's `✗` line does not
// duplicate the per-target Errors: section that cmd/batch_render.go
// already emits. If this test fails, the user is seeing the same
// failure text twice in stderr.
func TestOrchestrator_AggregateErrorErrorIsSingleLine(t *testing.T) {
	t.Parallel()

	rep := &reportertesting.MockReporter{}
	targets := makeTargets("a", "b", "c")
	o := &Orchestrator{
		Targets:         targets,
		ContinueOnError: true,
		Reporter:        rep,
		Execute: func(_ context.Context, _ *Target, tr reporter.TargetReporter) error {
			err := errors.New("boom")
			tr.Fail(err)
			return err
		},
	}
	_, err := o.Run(context.Background())
	if err == nil {
		t.Fatal("Run should return non-nil error")
	}
	msg := err.Error()
	if strings.Contains(msg, "\n") {
		t.Errorf("AggregateError.Error() must be one line; got:\n%s", msg)
	}
	if !strings.Contains(msg, "3 of 3") {
		t.Errorf("expected summary to mention '3 of 3'; got %q", msg)
	}

	var aggregate *AggregateError
	if !errors.As(err, &aggregate) {
		t.Fatalf("errors.As must extract *AggregateError; err = %v", err)
	}
	if aggregate.Total != 3 || len(aggregate.Errs) != 3 {
		t.Errorf("AggregateError = %+v, want Total=3 Errs=3", aggregate)
	}
}
