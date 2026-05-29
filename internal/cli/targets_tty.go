package cli

import (
	"fmt"
	"io"
	"os"
	"time"

	bspinner "charm.land/bubbles/v2/spinner"
)

// ttyTargets renders the multi-row Targets block in place. It pre-prints
// every row as pending, then redraws the whole block on every state change
// or animation tick. Pre-printing once and then moving the cursor up by
// len(rows) for each redraw keeps the cursor math trivial — the same trick
// ttyChecklist uses.
//
// cols is the terminal column width captured at construction; 0 means
// "unknown — do not truncate". When non-zero, formatLine truncates the
// row's variable-length payload (errMsg / summary / detail / reason) so
// each rendered row fits within one terminal line. The redraw assumes
// one visual line per row (`\033[NA` moves up N logical rows); a wrapped
// row would shift the cursor and cause earlier output to bleed into
// subsequent frames.
type ttyTargets struct {
	targetsBase
	w        io.Writer
	cols     int
	frames   []string
	fps      time.Duration
	frameIdx int

	stop chan struct{}
	done chan struct{}
}

// newTTYTargets prints the initial pending block and starts the animation
// goroutine. Empty id lists short-circuit to a closed handle so tests / dry
// runs do not leak the goroutine.
func newTTYTargets(r *Reporter, ids []string) *ttyTargets {
	fps := bspinner.MiniDot.FPS
	if fps <= 0 {
		fps = 120 * time.Millisecond
	}
	t := &ttyTargets{
		targetsBase: newTargetsBase(ids),
		w:           r.errW,
		cols:        terminalColsOf(r.errW),
		frames:      bspinner.MiniDot.Frames,
		fps:         fps,
		stop:        make(chan struct{}),
		done:        make(chan struct{}),
	}
	if len(ids) == 0 {
		close(t.done)
		return t
	}
	t.renderInitial()
	go t.animate()
	return t
}

// terminalColsOf returns the column width of w when w is an *os.File
// connected to a TTY; otherwise 0 (used by callers as "do not truncate").
// Imported into a thin helper rather than inlined so tests can pass an
// io.Writer (e.g. *bytes.Buffer) without provoking a nil-deref.
func terminalColsOf(w io.Writer) int {
	if f, ok := w.(*os.File); ok {
		return TerminalCols(f)
	}
	return 0
}

// renderInitial prints every row once. After this the cursor sits on the
// line directly below the last row; subsequent redraws move up by len(order)
// to reach the top of the block.
func (t *ttyTargets) renderInitial() {
	for _, id := range t.order {
		fmt.Fprintln(t.w, t.formatLine(t.rows[id], t.frames[0]))
	}
}

// redraw rewrites the entire block in place. Caller MUST hold t.mu.
//
// `\033[%dA` moves the cursor up by len(order) logical rows. formatLine
// truncates each row to one terminal line so this stays in sync; on top
// of that, `\033[J` (clear from cursor to end of screen) runs once after
// the cursor returns to the top of the block as a safety net for any
// previous frame that may have overflowed (e.g. a frame written before
// the terminal width was knowable, or after a SIGWINCH narrowed it).
func (t *ttyTargets) redraw() {
	frame := t.frames[t.frameIdx%len(t.frames)]
	fmt.Fprintf(t.w, "\033[%dA\r\033[J", len(t.order))
	for _, id := range t.order {
		fmt.Fprintln(t.w, t.formatLine(t.rows[id], frame))
	}
}

// formatLine builds one rendered row, truncating long payload fields so
// the result fits within one terminal line. The fields fed into
// renderRow are the only variable-length inputs — the icon, "failed: "
// prefix, and progress bar are fixed-width, so a per-field rune budget
// is sufficient and avoids ANSI-aware string truncation.
func (t *ttyTargets) formatLine(row *targetsRow, frame string) string {
	r := *row
	if t.cols > 0 {
		// Reserve fixed budget for: identifier column, gap, state icon,
		// space, common prefixes ("failed: " etc.), and a 1-col safety
		// margin so the cursor never sits exactly at the right edge
		// (some terminals wrap on column-0 of the next line in that
		// case). Floor at 10 so very narrow terminals still get
		// something visible rather than a string of just ellipses.
		budget := max(t.cols-t.idWidth-12, 10)
		r.errMsg = truncateCells(r.errMsg, budget)
		r.summary = truncateCells(r.summary, budget)
		r.detail = truncateCells(r.detail, budget)
		r.reason = truncateCells(r.reason, budget)
	}
	return padID(r.id, t.idWidth) + renderRow(&r, frame)
}

func (t *ttyTargets) animate() {
	defer close(t.done)
	ticker := time.NewTicker(t.fps)
	defer ticker.Stop()
	for {
		select {
		case <-t.stop:
			return
		case <-ticker.C:
			t.mu.Lock()
			if t.closed {
				t.mu.Unlock()
				return
			}
			t.frameIdx++
			t.redraw()
			t.mu.Unlock()
		}
	}
}

// SetPhase advances a row to running with the given phase + detail. If the
// row was previously in deploying with a real progress bar and the new phase
// is something else, the progress flag is cleared.
func (t *ttyTargets) SetPhase(id, phase, detail string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	row, ok := t.rows[sanitizeIdentifier(id)]
	if !ok || t.closed || isTerminalState(row.state) {
		return
	}
	row.state = rowRunning
	if row.phase != phase {
		row.hasProgress = false
		row.percent = 0
	}
	row.phase = phase
	row.detail = detail
	t.redraw()
}

func (t *ttyTargets) SetProgress(id string, percent float64, eta time.Duration) {
	t.mu.Lock()
	defer t.mu.Unlock()
	row, ok := t.rows[sanitizeIdentifier(id)]
	if !ok || t.closed || isTerminalState(row.state) {
		return
	}
	row.state = rowRunning
	row.hasProgress = true
	row.percent = percent
	row.eta = eta
	t.redraw()
}

func (t *ttyTargets) Done(id, summary string) {
	t.transition(id, rowDone, func(row *targetsRow) { row.summary = summary })
}

func (t *ttyTargets) Fail(id string, err error) {
	msg := ""
	if err != nil {
		msg = err.Error()
	}
	t.transition(id, rowFail, func(row *targetsRow) { row.errMsg = msg })
}

func (t *ttyTargets) Skip(id, reason string) {
	t.transition(id, rowSkip, func(row *targetsRow) { row.reason = reason })
}

func (t *ttyTargets) transition(id string, state targetsRowState, mutate func(*targetsRow)) {
	t.mu.Lock()
	defer t.mu.Unlock()
	row, ok := t.rows[sanitizeIdentifier(id)]
	if !ok || t.closed || isTerminalState(row.state) {
		return
	}
	row.state = state
	mutate(row)
	t.redraw()
}

func (t *ttyTargets) Close() {
	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return
	}
	t.closed = true
	t.mu.Unlock()
	if len(t.order) > 0 {
		// Order matters: setting t.closed under mu prevents new mutations
		// from scheduling redraws, close(t.stop) tells animate() to exit,
		// and <-t.done blocks until that goroutine has actually returned.
		// Only after that fence is the final redraw safe without extra
		// synchronisation — the animate ticker can no longer fire and no
		// other writer can race with us.
		close(t.stop)
		<-t.done
		// Final redraw with the resting frame so any lingering spinner
		// glyph isn't left mid-animation.
		t.mu.Lock()
		t.redraw()
		t.mu.Unlock()
	}
}
