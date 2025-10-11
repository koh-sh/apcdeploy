package status

// Options contains the configuration for status operation
type Options struct {
	// ConfigFile is the path to the apcdeploy configuration file
	ConfigFile string
	// DeploymentID is the deployment number to check (optional, defaults to latest)
	DeploymentID string
	// Silent indicates whether to suppress verbose output
	Silent bool
}
