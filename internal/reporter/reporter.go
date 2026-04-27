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
//   - Step / Success / Info / Warn / Header / Box / Table / Spin → stderr,
//     suppressed in silent mode.
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
	// Spinner. On non-TTY output, Spin emits a Step line and the spinner is
	// a no-op until Done/Fail.
	Spin(msg string) Spinner

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
// one of Done or Fail to terminate the spinner.
type Spinner interface {
	// Done stops the spinner and reports a success line with the given message.
	Done(msg string)
	// Fail stops the spinner and reports an error line with the given message.
	Fail(msg string)
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
