package cli

import "github.com/koh-sh/apcdeploy/internal/reporter"

// SilentReporter implements the reporter.ProgressReporter interface but suppresses all output
type SilentReporter struct{}

// Ensure SilentReporter implements the interface
var _ reporter.ProgressReporter = (*SilentReporter)(nil)

// NewSilentReporter creates a new silent reporter
func NewSilentReporter() *SilentReporter {
	return &SilentReporter{}
}

func (r *SilentReporter) Progress(message string) {
	// Suppress progress messages in silent mode
}

func (r *SilentReporter) Success(message string) {
	// Suppress success messages in silent mode
}

func (r *SilentReporter) Warning(message string) {
	// Suppress warning messages in silent mode
}
