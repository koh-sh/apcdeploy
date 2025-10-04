package cmd

import (
	"fmt"

	"github.com/koh-sh/apcdeploy/internal/display"
	"github.com/koh-sh/apcdeploy/internal/reporter"
)

// cliReporter implements the reporter.ProgressReporter interface for CLI output
type cliReporter struct{}

// Ensure cliReporter implements the interface
var _ reporter.ProgressReporter = (*cliReporter)(nil)

func (r *cliReporter) Progress(message string) {
	fmt.Println(display.Progress(message))
}

func (r *cliReporter) Success(message string) {
	fmt.Println(display.Success(message))
}

func (r *cliReporter) Warning(message string) {
	fmt.Println(display.Warning(message))
}
