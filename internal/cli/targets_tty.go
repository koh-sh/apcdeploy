package cli

import (
	"fmt"
	"io"
	"time"

	bspinner "github.com/charmbracelet/bubbles/spinner"
)

// ttyTargets renders the multi-row Targets block in place. It pre-prints
// every row as pending, then redraws the whole block on every state change
// or animation tick. Pre-printing once and then moving the cursor up by
// len(rows) for each redraw keeps the cursor math trivial — the same trick
// ttyChecklist uses.
type ttyTargets struct {
	targetsBase
	w        io.Writer
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

// renderInitial prints every row once. After this the cursor sits on the
// line directly below the last row; subsequent redraws move up by len(order)
// to reach the top of the block.
func (t *ttyTargets) renderInitial() {
	for _, id := range t.order {
		fmt.Fprintln(t.w, t.formatLine(t.rows[id], t.frames[0]))
	}
}

// redraw rewrites the entire block in place. Caller MUST hold t.mu.
func (t *ttyTargets) redraw() {
	frame := t.frames[t.frameIdx%len(t.frames)]
	fmt.Fprintf(t.w, "\033[%dA", len(t.order))
	for _, id := range t.order {
		fmt.Fprintf(t.w, "\r\033[K%s\n", t.formatLine(t.rows[id], frame))
	}
}

func (t *ttyTargets) formatLine(row *targetsRow, frame string) string {
	return padID(row.id, t.idWidth) + renderRow(row, frame)
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
		close(t.stop)
		<-t.done
		// Final redraw with the resting frame so any lingering spinner
		// glyph isn't left mid-animation.
		t.mu.Lock()
		t.redraw()
		t.mu.Unlock()
	}
}
