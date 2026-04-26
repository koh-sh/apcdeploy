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
