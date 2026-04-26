package edit

// Options contains the configuration options for the edit command
type Options struct {
	Region             string
	Application        string
	Profile            string
	Environment        string
	DeploymentStrategy string
	WaitDeploy         bool
	WaitBake           bool
	Timeout            int
	Silent             bool
}
