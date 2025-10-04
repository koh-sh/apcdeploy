package status

// ProgressReporter defines the interface for reporting progress during status operations
type ProgressReporter interface {
	Progress(message string)
	Success(message string)
	Warning(message string)
}
