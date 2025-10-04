package deploy

// ProgressReporter defines the interface for reporting progress during deployment
type ProgressReporter interface {
	Progress(message string)
	Success(message string)
	Warning(message string)
}
