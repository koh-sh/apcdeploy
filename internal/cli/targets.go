package cli

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/koh-sh/apcdeploy/internal/reporter"
)

// targetsBarWidth is the fixed visual width of the deploying-phase progress
// bar (output.md §5.4). Bars do not adapt to terminal width — they print at
// 20 cells regardless.
const targetsBarWidth = 20

// targetsIDGap is the minimum gap between the identifier column and the
// state icon (output.md §5.2). Implementations pad shorter identifiers with
// spaces so the icon column lines up across rows.
const targetsIDGap = 3

// targetsRowState captures the lifecycle stage of a single Targets row.
type targetsRowState int

const (
	rowPending targetsRowState = iota // initial state before SetPhase
	rowRunning                        // after SetPhase, before terminal call
	rowDone
	rowFail
	rowSkip
)

// targetsRow is one tracked target. All access is guarded by the parent
// implementation's mutex; callers MUST NOT touch fields directly from outside.
type targetsRow struct {
	id     string
	state  targetsRowState
	phase  string
	detail string

	// hasProgress is set when SetProgress has been called (deploying phase).
	hasProgress bool
	percent     float64 // clamped to [0, 1]
	eta         time.Duration

	// Terminal-state messages. Only one is populated per row.
	summary string // rowDone
	errMsg  string // rowFail
	reason  string // rowSkip
}

// Targets dispatches to the TTY or non-TTY implementation based on whether
// the Reporter's stderr is a terminal.
func (r *Reporter) Targets(ids []string) reporter.Targets {
	if r.errTTY {
		return newTTYTargets(r, ids)
	}
	return newPlainTargets(r, ids)
}

// idColumnWidth returns the rune-aware width of the longest identifier
// padded by targetsIDGap. Used by both TTY and non-TTY implementations
// (the latter pads only because the format keeps the identifier column).
func idColumnWidth(ids []string) int {
	w := 0
	for _, id := range ids {
		if n := visibleWidth(id); n > w {
			w = n
		}
	}
	return w + targetsIDGap
}

// padID returns id padded with spaces to width.
func padID(id string, width int) string {
	pad := max(width-visibleWidth(id), 0)
	return id + strings.Repeat(" ", pad)
}

// renderRow returns the post-identifier portion of a row (state icon onward).
// frame is the current spinner glyph; it is ignored when the row is not in
// an animated state.
func renderRow(row *targetsRow, frame string) string {
	switch row.state {
	case rowDone:
		return styles.success.Render(symSuccess) + " " + row.summary
	case rowFail:
		return styles.errorS.Render(symError) + " failed: " + row.errMsg
	case rowSkip:
		return styles.subtle.Render(symSkip) + " " + styles.subtle.Render(row.reason)
	case rowRunning:
		return renderRunning(row, frame)
	default:
		return styles.subtle.Render(symPending) + " " + styles.subtle.Render("pending")
	}
}

// renderRunning renders the running-state body. Deploying with a known
// percent gets a 20-cell bar; everything else gets a spinner frame.
func renderRunning(row *targetsRow, frame string) string {
	var b strings.Builder
	if row.hasProgress {
		b.WriteString(renderBar(row.percent))
		fmt.Fprintf(&b, "  %3d%% ", clampPercent(row.percent))
	} else {
		b.WriteString(styles.step.Render(frame))
		b.WriteString(" ")
	}
	b.WriteString(row.phase)
	if row.detail != "" {
		b.WriteString(" ")
		b.WriteString(styles.subtle.Render(row.detail))
	} else if row.eta > 0 {
		b.WriteString(" ")
		b.WriteString(styles.subtle.Render(formatETA(row.eta)))
	}
	return b.String()
}

// renderBar produces a 20-cell █/░ bar for percent in [0, 1].
func renderBar(percent float64) string {
	filled := max(int(percent*float64(targetsBarWidth)+0.5), 0)
	if filled > targetsBarWidth {
		filled = targetsBarWidth
	}
	full := strings.Repeat("█", filled)
	empty := strings.Repeat("░", targetsBarWidth-filled)
	return styles.success.Render(full) + styles.subtle.Render(empty)
}

func clampPercent(p float64) int {
	switch {
	case p <= 0:
		return 0
	case p >= 1:
		return 100
	default:
		return int(p*100 + 0.5)
	}
}

// formatETA renders a time.Duration as "(~N min left)" / "(~N sec left)".
// Returns "" for non-positive durations.
func formatETA(d time.Duration) string {
	if d <= 0 {
		return ""
	}
	if d < time.Minute {
		return fmt.Sprintf("(~%d sec left)", int(d.Seconds()+0.5))
	}
	return fmt.Sprintf("(~%d min left)", int(d.Minutes()+0.5))
}

// targetsBase holds the fields shared by the TTY and non-TTY implementations.
type targetsBase struct {
	mu      sync.Mutex
	rows    map[string]*targetsRow
	order   []string
	idWidth int
	closed  bool
}

func newTargetsBase(ids []string) targetsBase {
	rows := make(map[string]*targetsRow, len(ids))
	for _, id := range ids {
		rows[id] = &targetsRow{id: id, state: rowPending}
	}
	return targetsBase{
		rows:    rows,
		order:   append([]string(nil), ids...),
		idWidth: idColumnWidth(ids),
	}
}
