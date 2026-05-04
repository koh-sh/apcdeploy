package cli

import (
	"os"

	"golang.org/x/term"
)

// IsTerminal reports whether the given file is connected to a terminal.
// Used by the Reporter to decide whether to enable animations and color.
func IsTerminal(f *os.File) bool {
	if f == nil {
		return false
	}
	return term.IsTerminal(int(f.Fd()))
}

// TerminalCols returns the column width of f when it is a TTY, or 0 when
// the size cannot be determined (non-TTY, closed file, syscall error).
// Callers MUST treat 0 as "unknown — do not truncate".
func TerminalCols(f *os.File) int {
	if f == nil {
		return 0
	}
	w, _, err := term.GetSize(int(f.Fd()))
	if err != nil {
		return 0
	}
	return w
}
