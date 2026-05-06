package batch

import "fmt"

// AggregateError wraps the per-target errors collected by Orchestrator.Run
// so cmd/root.go's top-level error renderer can show a single-line summary
// (`N of M targets failed`) instead of the multi-line text that
// `errors.Join` produces. Sentinel detection still works because the
// standard library's errors.Is / errors.As walk every error returned by
// Unwrap() []error.
//
// Why a custom type instead of errors.Join: cmd/batch_render.go already
// renders the per-target Errors: section (with Resolution: hints) before
// the top-level Error line is emitted. Using errors.Join here meant the
// joined error's Error() method duplicated that section verbatim under
// `✗`. AggregateError keeps Is/As transparent (so exit-code classifiers
// keep working) while collapsing Error() to one line.
type AggregateError struct {
	// Total is the number of targets the orchestrator dispatched.
	Total int
	// Errs is the list of per-target failures, in completion order.
	// Callers MUST NOT mutate this slice.
	Errs []TargetError
}

// Error returns a single-line summary so the top-level Reporter.Error
// line stays compact. Per-target detail lives in the Errors: section
// rendered by cmd/batch_render.go.
func (a *AggregateError) Error() string {
	return fmt.Sprintf("%d of %d targets failed", len(a.Errs), a.Total)
}

// Unwrap returns the per-target errors so errors.Is / errors.As walk
// every wrapped error and any of them matching a sentinel makes the
// whole AggregateError match. This is what keeps cmd/root.go's
// `errors.Is(err, aws.ErrNoDeployment)` exit-code branch working for
// single- and multi-target failures alike.
func (a *AggregateError) Unwrap() []error {
	out := make([]error, len(a.Errs))
	for i, e := range a.Errs {
		out[i] = e.Err
	}
	return out
}
