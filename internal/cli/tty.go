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
