package deploy

// Options contains the configuration options for deployment
type Options struct {
	ConfigFile string
	NoWait     bool
	Timeout    int
}
