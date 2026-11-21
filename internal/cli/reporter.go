package cli

import (
	"fmt"
	"io"
	"os"
	"strings"

	"charm.land/lipgloss/v2"
	ltable "charm.land/lipgloss/v2/table"
	"github.com/koh-sh/apcdeploy/internal/reporter"
	"github.com/koh-sh/apcdeploy/internal/sanitize"
)

// Reporter is the TTY-aware Reporter implementation backed by lipgloss styles.
// It writes human-facing kinds to stderr and machine-readable payloads to
// stdout, degrading borders/animations when the underlying file is not a
// terminal.
//
// Every caller-supplied string on a human-facing kind goes through
// internal/sanitize, because much of it comes from AWS and may be written
// by other principals. Error keeps line breaks (multi-line errors are by
// design); the other kinds are single-line. Data is never altered.
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
	fmt.Fprintf(r.errW, "%s %s\n", styles.step.Render(symStep), sanitize.Line(msg))
}

// Success marks a step as successfully completed.
func (r *Reporter) Success(msg string) {
	fmt.Fprintf(r.errW, "%s %s\n", styles.success.Render(symSuccess), sanitize.Line(msg))
}

// Info reports neutral information.
func (r *Reporter) Info(msg string) {
	fmt.Fprintf(r.errW, "%s %s\n", styles.info.Render(symInfo), sanitize.Line(msg))
}

// Warn reports a non-fatal anomaly.
func (r *Reporter) Warn(msg string) {
	fmt.Fprintf(r.errW, "%s %s\n", styles.warn.Render(symWarn), sanitize.Line(msg))
}

// Error reports a fatal error.
func (r *Reporter) Error(msg string) {
	fmt.Fprintf(r.errW, "%s %s\n", styles.errorS.Render(symError), sanitize.Text(msg))
}

// Header renders a section heading. In TTY mode it emits a styled title with
// a separator bar; in non-TTY mode it falls back to a plain title line.
func (r *Reporter) Header(title string) {
	title = sanitize.Line(title)
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
	title = sanitize.Line(title)
	lines = sanitizeCells(lines)
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
	headers = sanitizeCells(headers)
	clean := make([][]string, len(rows))
	for i, row := range rows {
		clean[i] = sanitizeCells(row)
	}
	rows = clean
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

// Diff writes a unified diff payload to stdout. Colorized when stdout is a TTY;
// only then, since the output is for a human, is the content neutralised.
// Piped output stays byte-for-byte.
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
	line = sanitize.Text(line)
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

// sanitizeCells returns a copy of cells with each one passed through
// sanitize.Line, leaving the caller's slice untouched.
func sanitizeCells(cells []string) []string {
	clean := make([]string, len(cells))
	for i, c := range cells {
		clean[i] = sanitize.Line(c)
	}
	return clean
}

// visibleWidth returns the terminal display width of s in cells.
// Full-width characters (e.g. CJK) each count as 2 cells; ANSI escape
// sequences count as 0. Uses lipgloss.Width which already handles both.
func visibleWidth(s string) int {
	return lipgloss.Width(s)
}
