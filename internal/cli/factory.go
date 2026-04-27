package cli

import "github.com/koh-sh/apcdeploy/internal/reporter"

// GetReporter returns the appropriate Reporter based on the --silent flag.
// This is the single source of truth for silent-mode selection — executors
// must not branch on opts.Silent themselves.
func GetReporter(silent bool) reporter.Reporter {
	if silent {
		return NewSilentReporter()
	}
	return NewReporter()
}
