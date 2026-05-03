package cli

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	ltable "github.com/charmbracelet/lipgloss/table"
	"github.com/koh-sh/apcdeploy/internal/reporter"
)

// Reporter is the TTY-aware Reporter implementation backed by lipgloss styles.
// It writes human-facing kinds to stderr and machine-readable payloads to
// stdout, degrading borders/animations when the underlying file is not a
// terminal.
type Reporter struct {
	outW   io.Writer
	errW   io.Writer
	outTTY bool
	errTTY bool
}

var _ reporter.Reporter = (*Reporter)(nil)

// NewReporter constructs a Reporter bound to os.Stdout / os.Stderr.
func NewReporter() *Reporter {
	return &Reporter{
		outW:   os.Stdout,
		errW:   os.Stderr,
		outTTY: IsTerminal(os.Stdout),
		errTTY: IsTerminal(os.Stderr),
	}
}

// Step announces the start of a long-running step.
func (r *Reporter) Step(msg string) {
	fmt.Fprintf(r.errW, "%s %s\n", styles.step.Render(symStep), msg)
}

// Success marks a step as successfully completed.
func (r *Reporter) Success(msg string) {
	fmt.Fprintf(r.errW, "%s %s\n", styles.success.Render(symSuccess), msg)
}

// Info reports neutral information.
func (r *Reporter) Info(msg string) {
	fmt.Fprintf(r.errW, "%s %s\n", styles.info.Render(symInfo), msg)
}

// Warn reports a non-fatal anomaly.
func (r *Reporter) Warn(msg string) {
	fmt.Fprintf(r.errW, "%s %s\n", styles.warn.Render(symWarn), msg)
}

// Error reports a fatal error.
func (r *Reporter) Error(msg string) {
	fmt.Fprintf(r.errW, "%s %s\n", styles.errorS.Render(symError), msg)
}

// Header renders a section heading. In TTY mode it emits a styled title with
// a separator bar; in non-TTY mode it falls back to a plain title line.
func (r *Reporter) Header(title string) {
	if !r.errTTY {
		fmt.Fprintln(r.errW)
		fmt.Fprintln(r.errW, title)
		return
	}
	fmt.Fprintln(r.errW)
	fmt.Fprintln(r.errW, styles.header.Render(title))
	fmt.Fprintln(r.errW, styles.headerBar.Render(strings.Repeat("─", visibleWidth(title)+2)))
}

// Box renders a multi-line panel with a title.
func (r *Reporter) Box(title string, lines []string) {
	if !r.errTTY {
		fmt.Fprintln(r.errW)
		if title != "" {
			fmt.Fprintln(r.errW, title)
		}
		for _, ln := range lines {
			fmt.Fprintln(r.errW, ln)
		}
		return
	}
	body := strings.Join(lines, "\n")
	if title != "" {
		body = styles.boxTitle.Render(title) + "\n\n" + body
	}
	fmt.Fprintln(r.errW)
	fmt.Fprintln(r.errW, styles.box.Render(body))
}

// Table renders a structured table with column headers.
func (r *Reporter) Table(headers []string, rows [][]string) {
	if !r.errTTY {
		// Plain text fallback: tab-separated.
		if len(headers) > 0 {
			fmt.Fprintln(r.errW, strings.Join(headers, "\t"))
		}
		for _, row := range rows {
			fmt.Fprintln(r.errW, strings.Join(row, "\t"))
		}
		return
	}
	t := ltable.New().
		Border(lipgloss.RoundedBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(styles.tableBorder)).
		StyleFunc(func(row, _ int) lipgloss.Style {
			if row == ltable.HeaderRow {
				return styles.tableHeader.Padding(0, 1)
			}
			return lipgloss.NewStyle().Padding(0, 1)
		}).
		Headers(headers...).
		Rows(rows...)
	fmt.Fprintln(r.errW, t.Render())
}

// Spin starts an animated indicator wrapping a long-running call. The caller
// MUST eventually invoke either Done or Fail on the returned Spinner.
func (r *Reporter) Spin(msg string) reporter.Spinner {
	return newSpinner(r, msg)
}

// Data writes a machine-readable payload to stdout.
func (r *Reporter) Data(p []byte) {
	_, _ = r.outW.Write(p)
}

// Diff writes a unified diff payload to stdout. Colorized when stdout is a TTY.
func (r *Reporter) Diff(p []byte) {
	if !r.outTTY {
		_, _ = r.outW.Write(p)
		return
	}
	for _, line := range strings.SplitAfter(string(p), "\n") {
		if line == "" {
			continue
		}
		_, _ = io.WriteString(r.outW, colorizeDiffLine(line))
	}
}

func colorizeDiffLine(line string) string {
	// Strip the trailing newline for styling, then re-attach.
	nl := ""
	if strings.HasSuffix(line, "\n") {
		nl = "\n"
		line = strings.TrimSuffix(line, "\n")
	}
	switch {
	case strings.HasPrefix(line, "+++") || strings.HasPrefix(line, "---"):
		return styles.diffMeta.Render(line) + nl
	case strings.HasPrefix(line, "@@"):
		return styles.diffHunk.Render(line) + nl
	case strings.HasPrefix(line, "+"):
		return styles.diffAdd.Render(line) + nl
	case strings.HasPrefix(line, "-"):
		return styles.diffDel.Render(line) + nl
	default:
		return styles.diffPlain.Render(line) + nl
	}
}

// visibleWidth returns the rune count of s, used for header underline width.
// Lipgloss color codes never reach this helper because callers pass the raw
// string before styling.
func visibleWidth(s string) int {
	n := 0
	for range s {
		n++
	}
	return n
}
