package cli

import (
	"fmt"
	"io"
	"time"
)

// plainTargets is the non-TTY Targets implementation. Without in-place
// updates, each phase transition emits a new line in `<id>: <body>` form
// (output.md §6.2). Progress is decimated to 25/50/75/100% so CI logs
// stay clean.
type plainTargets struct {
	targetsBase
	w io.Writer

	// lastPhase[id] is the phase already announced; transitions repeating
	// the same phase (e.g. successive SetPhase("preparing")) are dropped.
	lastPhase map[string]string

	// progressThreshold[id] is the highest deploying-progress threshold
	// already announced (0/25/50/75/100). Once 100 is announced no further
	// progress lines fire for that row.
	progressThreshold map[string]int
}

func newPlainTargets(r *Reporter, ids []string) *plainTargets {
	return &plainTargets{
		targetsBase:       newTargetsBase(ids),
		w:                 r.errW,
		lastPhase:         make(map[string]string, len(ids)),
		progressThreshold: make(map[string]int, len(ids)),
	}
}

// SetPhase emits one `<id>: <phase> [<detail>]` line per genuine phase
// transition. Repeating the same phase is silent.
func (t *plainTargets) SetPhase(id, phase, detail string) {
	clean := sanitizeIdentifier(id)
	t.mu.Lock()
	defer t.mu.Unlock()
	row, ok := t.rows[clean]
	if !ok || t.closed || isTerminalState(row.state) {
		return
	}
	row.state = rowRunning
	row.phase = phase
	row.detail = detail
	if t.lastPhase[clean] == phase {
		return
	}
	t.lastPhase[clean] = phase
	body := phase
	if detail != "" {
		body += " " + detail
	}
	fmt.Fprintf(t.w, "%s: %s\n", clean, body)
}

// SetProgress emits `<id>: <phase> NN%` only when the percent crosses a new
// 25-step threshold. Calling SetProgress also pins the row's phase to
// "deploying" (the only sub-phase that reports a real percent).
func (t *plainTargets) SetProgress(id string, percent float64, _ time.Duration) {
	clean := sanitizeIdentifier(id)
	t.mu.Lock()
	defer t.mu.Unlock()
	row, ok := t.rows[clean]
	if !ok || t.closed || isTerminalState(row.state) {
		return
	}
	row.state = rowRunning
	row.hasProgress = true
	row.percent = percent
	if row.phase == "" {
		row.phase = "deploying"
	}
	threshold := percentThreshold(percent)
	if threshold <= t.progressThreshold[clean] {
		return
	}
	t.progressThreshold[clean] = threshold
	fmt.Fprintf(t.w, "%s: %s %d%%\n", clean, row.phase, threshold)
}

// Done emits a single success line.
func (t *plainTargets) Done(id, summary string) {
	clean := sanitizeIdentifier(id)
	t.terminal(clean, rowDone, func() {
		fmt.Fprintf(t.w, "%s: %s %s\n", clean, symSuccess, summary)
	})
}

// Fail emits a single failure line. The error message is rendered raw; the
// surrounding command is responsible for any Errors: section.
func (t *plainTargets) Fail(id string, err error) {
	clean := sanitizeIdentifier(id)
	msg := ""
	if err != nil {
		msg = err.Error()
	}
	t.terminal(clean, rowFail, func() {
		fmt.Fprintf(t.w, "%s: %s failed: %s\n", clean, symError, msg)
	})
}

// Skip emits a single skip line.
func (t *plainTargets) Skip(id, reason string) {
	clean := sanitizeIdentifier(id)
	t.terminal(clean, rowSkip, func() {
		fmt.Fprintf(t.w, "%s: %s %s\n", clean, symSkip, reason)
	})
}

// terminal flips a row to a terminal state and emits the matching line.
// Repeated calls against the same id are dropped. id is expected to already
// be sanitized by the caller (the public methods sanitize at entry).
func (t *plainTargets) terminal(id string, state targetsRowState, emit func()) {
	t.mu.Lock()
	defer t.mu.Unlock()
	row, ok := t.rows[id]
	if !ok || t.closed || isTerminalState(row.state) {
		return
	}
	row.state = state
	emit()
}

func (t *plainTargets) Close() {
	t.mu.Lock()
	t.closed = true
	t.mu.Unlock()
}

// percentThreshold maps a [0, 1] percent to the highest crossed 25-step
// threshold (0/25/50/75/100). Values below 25% return 0 so newPlainTargets's
// zero-valued map matches "no thresholds announced yet".
func percentThreshold(percent float64) int {
	switch {
	case percent >= 1.0:
		return 100
	case percent >= 0.75:
		return 75
	case percent >= 0.50:
		return 50
	case percent >= 0.25:
		return 25
	default:
		return 0
	}
}
