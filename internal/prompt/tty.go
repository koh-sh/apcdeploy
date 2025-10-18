package prompt

import (
	"errors"
	"os"

	"golang.org/x/term"
)

// ErrNoTTY is returned when stdin is not a terminal
var ErrNoTTY = errors.New("interactive mode requires a TTY")

// CheckTTY verifies that stdin is connected to a terminal
// Returns ErrNoTTY if stdin is not a terminal
func CheckTTY() error {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return ErrNoTTY
	}
	return nil
}
