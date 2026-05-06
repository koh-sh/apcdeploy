package run

// Options contains the configuration options for deployment
type Options struct {
	WaitDeploy  bool
	WaitBake    bool
	Timeout     int
	Force       bool
	Description string
}
