package diff

// Options contains the configuration for diff operation
type Options struct {
	// ConfigFile is the path to the apcdeploy configuration file
	ConfigFile string
	// ExitNonzero indicates whether to exit with code 1 if differences exist
	ExitNonzero bool
}
