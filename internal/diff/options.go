package diff

// Options contains the configuration for diff operation
type Options struct {
	// ConfigFile is the path to the apcdeploy configuration file
	ConfigFile string
	// Region is the AWS region (optional, overrides config file)
	Region string
}
