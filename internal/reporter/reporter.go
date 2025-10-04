// Package reporter provides common interfaces for progress reporting across commands.
package reporter

// ProgressReporter defines the interface for reporting progress during operations
type ProgressReporter interface {
	Progress(message string)
	Success(message string)
	Warning(message string)
}
