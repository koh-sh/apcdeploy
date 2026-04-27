package cli

import (
	"fmt"
	"io"
	"sync"
	"time"

	bspinner "github.com/charmbracelet/bubbles/spinner"
)

// spinner is the concrete reporter.Spinner implementation. In TTY mode it
// animates bubbles/spinner frames on a goroutine; in non-TTY mode it degrades
// to a single Step-style line written when the spinner is constructed.
type spinner struct {
	w        io.Writer
	tty      bool
	stop     chan struct{}
	done     chan struct{}
	mu       sync.Mutex
	finished bool
	r        *Reporter
}

// newSpinner constructs a spinner. In TTY mode it kicks off the animation
// goroutine immediately; in non-TTY mode it writes a single Step line.
func newSpinner(r *Reporter, msg string) *spinner {
	s := &spinner{
		w:    r.errW,
		tty:  r.errTTY,
		stop: make(chan struct{}),
		done: make(chan struct{}),
		r:    r,
	}
	if !s.tty {
		// Non-TTY: emit a Step line and let Done/Fail emit Success/Error.
		r.Step(msg)
		close(s.done)
		return s
	}
	go s.animate(msg)
	return s
}

func (s *spinner) animate(msg string) {
	defer close(s.done)
	frames := bspinner.MiniDot.Frames
	fps := bspinner.MiniDot.FPS
	if fps <= 0 {
		fps = time.Second / 12
	}
	ticker := time.NewTicker(fps)
	defer ticker.Stop()
	idx := 0
	render := func() {
		frame := styles.step.Render(frames[idx%len(frames)])
		// \r returns to line start; \033[K clears to end of line.
		fmt.Fprintf(s.w, "\r\033[K%s %s", frame, msg)
	}
	render()
	for {
		select {
		case <-s.stop:
			// Clear the line so Done/Fail can print a clean final message.
			fmt.Fprint(s.w, "\r\033[K")
			return
		case <-ticker.C:
			idx++
			render()
		}
	}
}

// Done stops the spinner and prints a success line.
func (s *spinner) Done(msg string) {
	if !s.markFinished() {
		return
	}
	s.terminate()
	s.r.Success(msg)
}

// Fail stops the spinner and prints an error line.
func (s *spinner) Fail(msg string) {
	if !s.markFinished() {
		return
	}
	s.terminate()
	s.r.Error(msg)
}

func (s *spinner) markFinished() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.finished {
		return false
	}
	s.finished = true
	return true
}

func (s *spinner) terminate() {
	if s.tty {
		close(s.stop)
	}
	<-s.done
}
