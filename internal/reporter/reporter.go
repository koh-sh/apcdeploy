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
//   - Step / Success / Info / Warn / Header / Box / Table / Spin / Targets
//     → stderr, suppressed in silent mode.
//   - Error → stderr, always shown.
//   - Data / Diff → stdout, always shown.
//
// Targets is the primary primitive for run/diff/pull/rollback/edit/get/status
// (docs/design/output.md §4). The older Step / Success / Info / Spin
// primitives are retained for the init command, which is fundamentally a
// sequential interactive workflow that does not fit the target-centric
// model (output.md §11 Q-1, resolved in favour of keeping these primitives).
type Reporter interface {
	// Step announces the start of a long-running step. Used by init only;
	// other commands use Targets.SetPhase.
	Step(msg string)
	// Success marks a step as successfully completed. Used by init only.
	Success(msg string)
	// Info reports neutral information (e.g. "no deployment found, creating
	// without data"). Used by init only.
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
	// completion line — there is no leading Step announcement. Used by init
	// only; other commands use Targets.
	Spin(msg string) Spinner

	// Targets opens a target-centric, multi-row block where each id is one
	// row. The implementation precomputes column widths from the full id
	// list, so all identifiers MUST be supplied up front. Callers MUST
	// invoke Close on the returned handle exactly once (defer it).
	Targets(ids []string) Targets

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
