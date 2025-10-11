package run

// Options contains the configuration options for deployment
type Options struct {
	ConfigFile string
	WaitDeploy bool
	WaitBake   bool
	Timeout    int
	Force      bool
	Silent     bool
}
