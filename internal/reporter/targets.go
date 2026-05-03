package reporter

import "time"

// Targets is the single Reporter primitive used to render the lifecycle of
// one or more deployment targets (1 target = 1 line). It replaces the older
// Step / Success / Spin / Checklist / Progress kinds for command-level
// output — see docs/design/output.md §4.1.
//
// Identifiers passed via NewTargets fix the row order; subsequent calls
// reference each row by that identifier. The implementation MUST tolerate
// concurrent calls (the multi-config orchestrator drives several targets in
// parallel from separate goroutines).
//
// Phase strings are short verbs from the documented set
// (preparing / comparing / creating-version / deploying / baking) and are
// rendered after the state icon. Detail is free-form context appended after
// the phase (e.g. "(~5 min left)") and may be empty.
//
// Done / Fail / Skip each terminate the row's lifecycle. Subsequent calls
// against the same id are ignored (last-writer-wins is intentionally not
// supported — terminal states are sticky).
//
// Close MUST be called exactly once after every target has reached a
// terminal state. Forgetting Close leaks the renderer goroutine — defer it
// at the call site.
type Targets interface {
	// SetPhase advances the running row to a sub-phase with optional detail.
	// detail is shown verbatim after the phase label.
	SetPhase(id, phase, detail string)

	// SetProgress advances the row's progress bar (0.0-1.0). eta is shown
	// as "(~N min left)" when non-zero. Only meaningful for the deploying
	// sub-phase; other phases ignore the percent and render a spinner.
	SetProgress(id string, percent float64, eta time.Duration)

	// Done finalizes the row as successful. summary is the post-icon text
	// (e.g. "deployed (12s) — v42, AllAtOnce"); the implementation prefixes
	// the success icon and identifier alignment.
	Done(id, summary string)

	// Fail finalizes the row as failed. err's message is rendered after
	// "✗ failed:" and may be expanded into the Errors: section by the
	// surrounding command.
	Fail(id string, err error)

	// Skip finalizes the row as skipped (fail-fast, no-changes, etc.).
	// reason is the post-icon text (e.g. "skipped (no changes)").
	Skip(id, reason string)

	// Close releases the rendering goroutine. Idempotent. MUST be called
	// exactly once; defer it after construction.
	Close()
}
