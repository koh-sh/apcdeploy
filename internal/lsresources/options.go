package lsresources

// Options contains the configuration options for listing resources
type Options struct {
	// Region specifies the AWS region (empty string uses SDK default)
	Region string
	// JSON enables JSON output format
	JSON bool
	// ShowStrategies includes deployment strategies in output
	ShowStrategies bool
	// Silent indicates whether to suppress verbose output
	Silent bool
}
