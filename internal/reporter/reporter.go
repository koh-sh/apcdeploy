// Package reporter defines the standardized output interface used by every
// command. Implementations live in internal/cli (real and silent variants) and
// internal/reporter/testing (mock).
//
// The full contract — channels, kinds, --silent semantics, TTY degradation —
// is documented in .claude/rules/output-contract.md.
package reporter

// Reporter is the single output abstraction for all commands. Executors must
// not call fmt.Fprint* directly; everything flows through this interface.
//
// Channel and silent-mode behavior are fixed per kind:
//   - Step / Success / Info / Warn / Header / Box / Table / Spin / Checklist
//     → stderr, suppressed in silent mode.
//   - Error → stderr, always shown.
//   - Data / Diff → stdout, always shown.
type Reporter interface {
	// Step announces the start of a long-running step.
	Step(msg string)
	// Success marks a step as successfully completed.
	Success(msg string)
	// Info reports neutral information (e.g. "no deployment found, creating without data").
	Info(msg string)
	// Warn reports a non-fatal anomaly the user should notice.
	Warn(msg string)
	// Error reports a fatal error. Always shown, even in silent mode.
	// Used by cmd/root.go for the top-level error message.
	Error(msg string)

	// Header renders a section heading.
	Header(title string)
	// Box renders a multi-line panel with a title (e.g. init's "Next steps").
	Box(title string, lines []string)
	// Table renders a structured table with column headers.
	Table(headers []string, rows [][]string)

	// Spin starts an animated indicator wrapping a long-running call. The
	// caller MUST eventually invoke either Done or Fail on the returned
	// Spinner. On non-TTY output, Spin is silent until Done/Fail emits a
	// completion line — there is no leading Step announcement.
	Spin(msg string) Spinner

	// Checklist starts a multi-item progress block. items defines the labels
	// shown initially as pending (○). The caller drives transitions via the
	// returned Checklist handle, then closes it. In TTY mode the block updates
	// in place (active item shows an animated spinner); in non-TTY mode only
	// the completion line for each item is emitted (no pre-list, no Step).
	// Use Spin instead when the work has only a single phase.
	Checklist(items []string) Checklist

	// Progress starts a percentage-based progress indicator. In TTY mode it
	// renders a live bar that the caller updates via ProgressBar.Update; in
	// non-TTY mode it emits a Step line on construction and additional Step
	// lines at coarse percentage thresholds. The caller MUST eventually
	// invoke either Done or Fail on the returned ProgressBar.
	Progress(msg string) ProgressBar

	// Data writes a machine-readable payload to stdout. Always emitted.
	Data(p []byte)
	// Diff writes a unified diff payload to stdout. Always emitted; colorized
	// when stdout is a TTY.
	Diff(p []byte)
}

// Spinner is the handle returned by Reporter.Spin. Callers MUST call exactly
// one of Done, Fail, or Stop to terminate the spinner.
type Spinner interface {
	// Update replaces the spinner's animated label without changing its
	// running state. In TTY mode the next animation frame renders the new
	// message in place. In non-TTY mode Update is silent (matching the
	// "no narration mid-flight" rule for spinners). Update is safe to call
	// from a different goroutine than the one running the animation.
	Update(msg string)
	// Done stops the spinner and reports a success line with the given message.
	Done(msg string)
	// Fail stops the spinner and reports an error line with the given message.
	// Use Stop instead when the caller will propagate the error and rely on
	// cmd/root.go to format it, so the user sees a single error line.
	Fail(msg string)
	// Stop terminates the spinner without emitting any line. Use when the
	// caller is about to return an error that will be reported by the
	// top-level error handler.
	Stop()
}

// Checklist is the handle returned by Reporter.Checklist. Items are referenced
// by their zero-based index in the slice passed to Checklist. Each item moves
// through the states: pending (○) → active (⠋) → done (✓) / fail (✗) / skip
// (→). Callers MUST call Close exactly once to release any background
// rendering goroutine; Close is idempotent.
//
// Items left in pending or active state when Close is called are finalized as
// pending (○). For deterministic output, finish or skip every item explicitly
// before Close.
type Checklist interface {
	// Start marks the item as in-progress (animated spinner in TTY mode).
	// Calling Start on the same index twice is a no-op.
	Start(idx int)
	// Done marks the item as successfully completed; msg replaces the label
	// when non-empty.
	Done(idx int, msg string)
	// Fail marks the item as failed; msg replaces the label when non-empty.
	// In TTY mode the item renders with ✗. In non-TTY mode Fail is silent —
	// the caller is expected to return an error that cmd/root.go formats, so
	// the user sees a single error line. The silent Reporter forwards Fail to
	// Error so that fatal failures still surface in scripts.
	Fail(idx int, msg string)
	// Skip marks the item as skipped (e.g. an early-exit branch); msg
	// replaces the label when non-empty.
	Skip(idx int, msg string)
	// Close finalizes the checklist. After Close, further Start/Done/Fail/Skip
	// calls are no-ops.
	Close()
}

// ProgressBar is the handle returned by Reporter.Progress. Callers stream
// updates through Update and MUST eventually call exactly one of Done, Fail,
// or Stop to terminate the bar.
type ProgressBar interface {
	// Update sets the current completion percentage (0-100) and the label.
	// Values outside that range are clamped.
	Update(percent float64, msg string)
	// Done stops the bar and reports a success line with the given message.
	Done(msg string)
	// Fail stops the bar and reports an error line with the given message.
	// Use Stop instead when the caller will propagate the error and rely on
	// cmd/root.go to format it, so the user sees a single error line.
	Fail(msg string)
	// Stop terminates the bar without emitting any line. Use when the caller
	// is about to return an error that will be reported by the top-level
	// error handler.
	Stop()
}
