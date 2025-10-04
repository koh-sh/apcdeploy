package diff

// ProgressReporter defines the interface for reporting progress during diff operations
type ProgressReporter interface {
	Progress(message string)
	Success(message string)
	Warning(message string)
}
