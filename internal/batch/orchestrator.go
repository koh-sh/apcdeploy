package batch

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/koh-sh/apcdeploy/internal/reporter"
)

// ExecuteFunc is the per-target callback the Orchestrator invokes once for
// every loaded Target. Implementations MUST drive the supplied
// TargetReporter through to a terminal state (Done / Skip / Fail) before
// returning. Returning a non-nil error counts as a failure even if Fail was
// not called — the orchestrator will defensively call Fail in that case so
// the row never stays in "running" forever.
//
// The context is the parent passed to Orchestrator.Run; per-target
// timeouts (e.g. run's --timeout) are the executor's responsibility, not
// the orchestrator's (multi-config.md §7.4).
type ExecuteFunc func(ctx context.Context, t *Target, tr reporter.TargetReporter) error

// TargetError pairs a failed Target with the underlying error so the
// command layer can render the post-run "Errors:" section
// (docs/design/output.md §8.3).
type TargetError struct {
	Identifier string
	Path       string
	Err        error
}

// Summary aggregates per-target outcomes for the post-run line described
// in docs/design/output.md §8.2 (`N ok, N no-op, N failed [(elapsed)]`).
//
// Skipped is broken out from NoOp so that callers (and tests) can tell
// fail-fast skips from "no changes" no-ops. The aggregate "no-op" column
// shown to the user is NoOp + Skipped.
type Summary struct {
	OK      int
	NoOp    int
	Skipped int
	Failed  int
	Errors  []TargetError
	Elapsed time.Duration
}

// Orchestrator drives a slice of pre-loaded Targets through one shared
// Reporter.Targets handle, in parallel, with optional fail-fast or
// continue-on-error semantics (multi-config.md §7).
//
// The zero value is not usable — Targets, Reporter, and Execute MUST be
// set. Parallel <= 0 means "all at once" (i.e. len(Targets)).
type Orchestrator struct {
	Targets         []*Target
	Parallel        int
	ContinueOnError bool
	Reporter        reporter.Reporter
	Execute         ExecuteFunc
}

// Run drives every Target through Execute and returns a Summary plus an
// aggregate error (non-nil if any target failed, regardless of
// ContinueOnError — exit code 1 is the contract for any failure,
// multi-config.md §10.1).
//
// Concurrency model:
//   - One goroutine per Target, gated by a buffered semaphore of size
//     `parallel` so at most `parallel` targets execute simultaneously.
//   - Argument start order is preserved: Targets[i] is launched before
//     Targets[i+1]. With Parallel=1 this produces strict serial order
//     (multi-config.md §7.2).
//   - Fail-fast does NOT cancel running targets (§7.3). It only prevents
//     further targets from starting; queued targets are reported via
//     Targets.Skip with reason "skipped (fail-fast)".
func (o *Orchestrator) Run(ctx context.Context) (Summary, error) {
	if len(o.Targets) == 0 {
		return Summary{}, errors.New("orchestrator: no targets")
	}
	if o.Reporter == nil {
		return Summary{}, errors.New("orchestrator: Reporter is required")
	}
	if o.Execute == nil {
		return Summary{}, errors.New("orchestrator: Execute is required")
	}

	parallel := o.Parallel
	if parallel <= 0 || parallel > len(o.Targets) {
		parallel = len(o.Targets)
	}

	ids := make([]string, len(o.Targets))
	for i, t := range o.Targets {
		ids[i] = t.Identifier
	}
	tg := o.Reporter.Targets(ids)
	defer tg.Close()

	start := time.Now()

	// failed flag short-circuits scheduling under fail-fast. It is only
	// checked when ContinueOnError is false. Protected by mu along with
	// errors and the Skipped count for any target rejected on entry.
	var (
		mu     sync.Mutex
		failed bool
		errs   []TargetError
	)

	views := make([]*targetView, len(o.Targets))
	for i, t := range o.Targets {
		views[i] = &targetView{inner: tg, id: t.Identifier}
	}

	type workItem struct {
		idx    int
		target *Target
	}
	// Buffered so the producer never blocks: this lets us close the
	// channel up-front and rely on `for range work` for clean worker
	// shutdown. The channel is FIFO, so workers pull targets in
	// argument order even though completion order is undefined under
	// parallel >= 2 (multi-config.md §7.2).
	work := make(chan workItem, len(o.Targets))
	for i, t := range o.Targets {
		work <- workItem{idx: i, target: t}
	}
	close(work)

	var wg sync.WaitGroup
	for w := 0; w < parallel; w++ {
		wg.Go(func() {
			for item := range work {
				if !o.ContinueOnError {
					mu.Lock()
					skip := failed
					mu.Unlock()
					if skip {
						views[item.idx].markSkippedFailFast()
						tg.Skip(item.target.Identifier, "skipped (fail-fast)")
						continue
					}
				}

				err := o.Execute(ctx, item.target, views[item.idx])
				if err != nil {
					// If the executor returned an error but never
					// called Fail (contract violation), surface it to
					// the row so the user sees a terminal state.
					if views[item.idx].state == statePending {
						views[item.idx].Fail(err)
					}
					mu.Lock()
					failed = true
					errs = append(errs, TargetError{
						Identifier: item.target.Identifier,
						Path:       item.target.Path,
						Err:        err,
					})
					mu.Unlock()
				}
			}
		})
	}

	wg.Wait()
	elapsed := time.Since(start)

	summary := Summary{Elapsed: elapsed, Errors: errs}
	for _, v := range views {
		switch v.state {
		case stateOK:
			summary.OK++
		case stateNoOp:
			summary.NoOp++
		case stateFailed:
			summary.Failed++
		case stateSkipped:
			summary.Skipped++
		case statePending:
			// Executor returned nil without setting a terminal state. Not
			// expected, but count it as OK so the summary at least adds
			// up to len(Targets); the contract violation is its own bug.
			summary.OK++
		}
	}

	if len(errs) > 0 {
		return summary, fmt.Errorf("%d target(s) failed", len(errs))
	}
	return summary, nil
}

// targetView is a per-target wrapper around the shared reporter.Targets
// handle. Each goroutine owns one view, so no synchronisation is required
// inside the view itself.
type targetView struct {
	inner reporter.Targets
	id    string
	state targetState
}

type targetState int

const (
	statePending targetState = iota
	stateOK
	stateNoOp
	stateFailed
	stateSkipped
)

func (v *targetView) SetPhase(phase, detail string) {
	v.inner.SetPhase(v.id, phase, detail)
}

func (v *targetView) SetProgress(percent float64, eta time.Duration) {
	v.inner.SetProgress(v.id, percent, eta)
}

func (v *targetView) Done(summary string) {
	if v.state == statePending {
		v.state = stateOK
	}
	v.inner.Done(v.id, summary)
}

func (v *targetView) Skip(reason string) {
	if v.state == statePending {
		v.state = stateNoOp
	}
	v.inner.Skip(v.id, reason)
}

func (v *targetView) Fail(err error) {
	if v.state == statePending {
		v.state = stateFailed
	}
	v.inner.Fail(v.id, err)
}

// markSkippedFailFast records a fail-fast skip without round-tripping
// through Skip(). The orchestrator already called tg.Skip directly so we
// only need to update the local counter.
func (v *targetView) markSkippedFailFast() {
	v.state = stateSkipped
}
