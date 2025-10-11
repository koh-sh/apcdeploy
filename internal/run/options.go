package run

// Options contains the configuration options for deployment
type Options struct {
	ConfigFile string
	Wait       bool
	Timeout    int
	Force      bool
	Silent     bool
}
