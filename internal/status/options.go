package status

// Options contains the configuration for status operation
type Options struct {
	// ConfigFile is the path to the apcdeploy configuration file
	ConfigFile string
	// Region is the AWS region (optional, overrides config file)
	Region string
	// DeploymentID is the deployment number to check (optional, defaults to latest)
	DeploymentID string
}
