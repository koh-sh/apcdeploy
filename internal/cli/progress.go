package cli

import (
	"fmt"
	"io"
	"strings"
	"sync"
	"time"
)

// progressBarWidth is the visual width of the bar segment (excluding the
// percentage and label text).
const progressBarWidth = 24

// progressBar is the concrete reporter.ProgressBar implementation. In TTY
// mode it animates on a goroutine, redrawing on a ticker so the bar feels
// live even when Update is called less frequently than the frame rate. In
// non-TTY mode it degrades to coarse Step lines emitted at percentage
// thresholds.
type progressBar struct {
	w        io.Writer
	tty      bool
	stop     chan struct{}
	done     chan struct{}
	mu       sync.Mutex
	finished bool
	percent  float64
	msg      string
	r        *Reporter

	// Non-TTY mode only: the highest threshold already announced via Step.
	lastThreshold int
}

// progressStepThreshold is the granularity of non-TTY progress updates
// (25/50/75/100). Coarse on purpose so CI logs stay clean. The 0% line is
// covered by the Step emitted in newProgressBar.
const progressStepThreshold = 25

func newProgressBar(r *Reporter, msg string) *progressBar {
	p := &progressBar{
		w:    r.errW,
		tty:  r.errTTY,
		stop: make(chan struct{}),
		done: make(chan struct{}),
		r:    r,
		msg:  msg,
	}
	if !p.tty {
		// Non-TTY: emit an initial Step line; subsequent Update calls may
		// emit additional Steps at threshold boundaries.
		r.Step(msg)
		close(p.done)
		return p
	}
	go p.animate()
	return p
}

func (p *progressBar) animate() {
	defer close(p.done)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	p.render()
	for {
		select {
		case <-p.stop:
			fmt.Fprint(p.w, "\r\033[K")
			return
		case <-ticker.C:
			p.render()
		}
	}
}

func (p *progressBar) render() {
	p.mu.Lock()
	pct := p.percent
	msg := p.msg
	p.mu.Unlock()
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	filled := min(int(pct/100*float64(progressBarWidth)), progressBarWidth)
	bar := strings.Repeat("█", filled) + strings.Repeat("░", progressBarWidth-filled)
	styled := styles.step.Render(bar)
	fmt.Fprintf(p.w, "\r\033[K%s %5.1f%% %s", styled, pct, msg)
}

// Update advances the bar to the given percentage and label. percent is
// clamped to [0, 100] so threshold accounting can't go negative or runaway.
func (p *progressBar) Update(percent float64, msg string) {
	switch {
	case percent < 0:
		percent = 0
	case percent > 100:
		percent = 100
	}
	p.mu.Lock()
	if p.finished {
		p.mu.Unlock()
		return
	}
	p.percent = percent
	if msg != "" {
		p.msg = msg
	}
	tty := p.tty
	threshold := int(percent/progressStepThreshold) * progressStepThreshold
	announceThreshold := !tty && threshold > p.lastThreshold && threshold <= 100
	if announceThreshold {
		p.lastThreshold = threshold
	}
	currentMsg := p.msg
	p.mu.Unlock()

	if announceThreshold {
		p.r.Step(fmt.Sprintf("%s (%d%%)", currentMsg, threshold))
	}
}

// Done stops the bar and reports a success line.
func (p *progressBar) Done(msg string) {
	if !p.markFinished() {
		return
	}
	p.terminate()
	p.r.Success(msg)
}

// Fail stops the bar and reports an error line.
func (p *progressBar) Fail(msg string) {
	if !p.markFinished() {
		return
	}
	p.terminate()
	p.r.Error(msg)
}

// Stop terminates the bar without emitting any line. Used when the caller
// will propagate an error and rely on cmd/root.go to format it.
func (p *progressBar) Stop() {
	if !p.markFinished() {
		return
	}
	p.terminate()
}

func (p *progressBar) markFinished() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.finished {
		return false
	}
	p.finished = true
	return true
}

func (p *progressBar) terminate() {
	if p.tty {
		close(p.stop)
	}
	<-p.done
}
