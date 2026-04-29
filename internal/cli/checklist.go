package cli

import (
	"fmt"
	"io"
	"sync"
	"time"

	bspinner "github.com/charmbracelet/bubbles/spinner"
	"github.com/koh-sh/apcdeploy/internal/reporter"
)

// itemState is the lifecycle state of a single checklist item.
type itemState int

const (
	statePending itemState = iota
	stateActive
	stateDone
	stateFail
	stateSkip
)

// checklistItem holds the rendered label and current state.
type checklistItem struct {
	label string
	msg   string // overrides label once state != pending
	state itemState
}

// ttyChecklist is the TTY implementation of reporter.Checklist. It pre-prints
// every item as pending (○), then redraws the whole block in place when state
// changes. A goroutine ticks at the spinner FPS to animate the active item's
// frame; redrawing the entire block on each tick (rather than only the active
// line) keeps the cursor math trivial.
type ttyChecklist struct {
	w        io.Writer
	items    []checklistItem
	frames   []string
	fps      time.Duration
	frameIdx int

	mu     sync.Mutex
	closed bool
	stop   chan struct{}
	done   chan struct{}
}

// newTTYChecklist constructs and renders the initial pending block.
func newTTYChecklist(w io.Writer, labels []string) *ttyChecklist {
	items := make([]checklistItem, len(labels))
	for i, l := range labels {
		items[i] = checklistItem{label: l, state: statePending}
	}
	fps := bspinner.MiniDot.FPS
	if fps <= 0 {
		fps = time.Second / 12
	}
	c := &ttyChecklist{
		w:      w,
		items:  items,
		frames: bspinner.MiniDot.Frames,
		fps:    fps,
		stop:   make(chan struct{}),
		done:   make(chan struct{}),
	}
	c.renderInitial()
	go c.animate()
	return c
}

// renderInitial prints every item line followed by a newline. After this the
// cursor sits on the line directly below the last item; subsequent redraws
// move up by len(items) to reach the top of the block.
func (c *ttyChecklist) renderInitial() {
	for _, it := range c.items {
		fmt.Fprintln(c.w, c.formatLine(it, c.frames[0]))
	}
}

// redraw rewrites the entire block in place. Called both on state changes and
// from the animation ticker for the active spinner frame.
func (c *ttyChecklist) redraw() {
	frame := c.frames[c.frameIdx%len(c.frames)]
	// Move cursor up to the first item's line.
	fmt.Fprintf(c.w, "\033[%dA", len(c.items))
	for _, it := range c.items {
		// \r moves to col 0; \033[K clears to end of line; \n advances.
		fmt.Fprintf(c.w, "\r\033[K%s\n", c.formatLine(it, frame))
	}
}

// formatLine renders a single item with its symbol + (label or msg).
func (c *ttyChecklist) formatLine(it checklistItem, activeFrame string) string {
	text := it.label
	if it.msg != "" {
		text = it.msg
	}
	switch it.state {
	case stateActive:
		return styles.step.Render(activeFrame) + " " + text
	case stateDone:
		return styles.success.Render(symSuccess) + " " + text
	case stateFail:
		return styles.errorS.Render(symError) + " " + text
	case stateSkip:
		return styles.subtle.Render(symSkip) + " " + styles.subtle.Render(text)
	default:
		return styles.subtle.Render(symPending) + " " + styles.subtle.Render(text)
	}
}

func (c *ttyChecklist) animate() {
	defer close(c.done)
	ticker := time.NewTicker(c.fps)
	defer ticker.Stop()
	for {
		select {
		case <-c.stop:
			return
		case <-ticker.C:
			c.mu.Lock()
			if c.closed {
				c.mu.Unlock()
				return
			}
			c.frameIdx++
			c.redraw()
			c.mu.Unlock()
		}
	}
}

// Start transitions an item to the active state.
func (c *ttyChecklist) Start(idx int) { c.transition(idx, stateActive, "") }

// Done transitions an item to the done state with an optional new message.
func (c *ttyChecklist) Done(idx int, msg string) { c.transition(idx, stateDone, msg) }

// Fail transitions an item to the failed state with an optional new message.
func (c *ttyChecklist) Fail(idx int, msg string) { c.transition(idx, stateFail, msg) }

// Skip transitions an item to the skipped state with an optional new message.
func (c *ttyChecklist) Skip(idx int, msg string) { c.transition(idx, stateSkip, msg) }

func (c *ttyChecklist) transition(idx int, state itemState, msg string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed || idx < 0 || idx >= len(c.items) {
		return
	}
	c.items[idx].state = state
	if msg != "" {
		c.items[idx].msg = msg
	}
	c.redraw()
}

// Close stops the animation goroutine. The final block remains on screen.
func (c *ttyChecklist) Close() {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return
	}
	c.closed = true
	c.mu.Unlock()
	close(c.stop)
	<-c.done
	// Final redraw with the resting frame so the spinner glyph isn't left
	// frozen mid-animation if an item was active at Close time.
	c.mu.Lock()
	c.redraw()
	c.mu.Unlock()
}

// nonTTYChecklist emits one Success/Error/Info line per Done/Fail/Skip
// transition. Pending and Start are silent, matching the user-facing
// requirement that non-TTY logs carry only terminal states.
type nonTTYChecklist struct {
	r        *Reporter
	items    []checklistItem
	mu       sync.Mutex
	closed   bool
	finished []bool
}

func newNonTTYChecklist(r *Reporter, labels []string) *nonTTYChecklist {
	items := make([]checklistItem, len(labels))
	for i, l := range labels {
		items[i] = checklistItem{label: l, state: statePending}
	}
	return &nonTTYChecklist{
		r:        r,
		items:    items,
		finished: make([]bool, len(labels)),
	}
}

func (c *nonTTYChecklist) Start(int) {}

func (c *nonTTYChecklist) Done(idx int, msg string) {
	if !c.markFinished(idx) {
		return
	}
	c.r.Success(c.label(idx, msg))
}

// Fail in non-TTY mode is silent: callers are expected to return the error to
// cmd/root.go which emits the user-facing failure line. Suppressing here keeps
// non-TTY output to one error line per failure.
//
// The msg argument is intentionally unused but kept for interface parity with
// Done/Skip and to match the TTY implementation's signature.
func (c *nonTTYChecklist) Fail(idx int, msg string) {
	_ = msg
	c.markFinished(idx)
}

// Skip in non-TTY mode emits an Info line, distinguishing the early-exit
// branch (e.g. "no changes detected") from a successful completion.
func (c *nonTTYChecklist) Skip(idx int, msg string) {
	if !c.markFinished(idx) {
		return
	}
	c.r.Info(c.label(idx, msg))
}

func (c *nonTTYChecklist) Close() {
	c.mu.Lock()
	c.closed = true
	c.mu.Unlock()
}

func (c *nonTTYChecklist) markFinished(idx int) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed || idx < 0 || idx >= len(c.items) || c.finished[idx] {
		return false
	}
	c.finished[idx] = true
	return true
}

func (c *nonTTYChecklist) label(idx int, msg string) string {
	if msg != "" {
		return msg
	}
	return c.items[idx].label
}

// silentChecklist forwards only Fail to Error; Done/Skip/Start are no-ops.
// Mirrors silentSpinner so fatal failures still surface in scripts using
// --silent.
type silentChecklist struct {
	r        *SilentReporter
	count    int
	mu       sync.Mutex
	finished []bool
}

func newSilentChecklist(r *SilentReporter, labels []string) *silentChecklist {
	return &silentChecklist{
		r:        r,
		count:    len(labels),
		finished: make([]bool, len(labels)),
	}
}

func (c *silentChecklist) Start(int)        {}
func (c *silentChecklist) Done(int, string) {}
func (c *silentChecklist) Skip(int, string) {}
func (c *silentChecklist) Close()           {}

func (c *silentChecklist) Fail(idx int, msg string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if idx < 0 || idx >= c.count || c.finished[idx] {
		return
	}
	c.finished[idx] = true
	c.r.Error(msg)
}

// Compile-time interface checks.
var (
	_ reporter.Checklist = (*ttyChecklist)(nil)
	_ reporter.Checklist = (*nonTTYChecklist)(nil)
	_ reporter.Checklist = (*silentChecklist)(nil)
)
