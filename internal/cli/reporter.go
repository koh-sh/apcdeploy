package cli

import (
	"fmt"

	"github.com/koh-sh/apcdeploy/internal/display"
	"github.com/koh-sh/apcdeploy/internal/reporter"
)

// Reporter implements the reporter.ProgressReporter interface for CLI output
type Reporter struct{}

// Ensure Reporter implements the interface
var _ reporter.ProgressReporter = (*Reporter)(nil)

// NewReporter creates a new CLI reporter
func NewReporter() *Reporter {
	return &Reporter{}
}

func (r *Reporter) Progress(message string) {
	fmt.Println(display.Progress(message))
}

func (r *Reporter) Success(message string) {
	fmt.Println(display.Success(message))
}

func (r *Reporter) Warning(message string) {
	fmt.Println(display.Warning(message))
}
