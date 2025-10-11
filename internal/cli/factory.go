package cli

import "github.com/koh-sh/apcdeploy/internal/reporter"

// GetReporter returns the appropriate reporter based on silent mode.
// When silent is true, returns a SilentReporter that suppresses all
// progress, success, and warning messages. Error messages are still
// displayed through stderr in the root command.
// When false, returns a regular Reporter with full colored output.
func GetReporter(silent bool) reporter.ProgressReporter {
	if silent {
		return NewSilentReporter()
	}
	return NewReporter()
}
