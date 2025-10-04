package cmd

import (
	"fmt"

	"github.com/koh-sh/apcdeploy/internal/display"
)

// cliReporter implements the ProgressReporter interface for CLI output
type cliReporter struct{}

func (r *cliReporter) Progress(message string) {
	fmt.Println(display.Progress(message))
}

func (r *cliReporter) Success(message string) {
	fmt.Println(display.Success(message))
}

func (r *cliReporter) Warning(message string) {
	fmt.Println(display.Warning(message))
}
