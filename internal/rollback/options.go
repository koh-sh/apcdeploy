package rollback

// Options contains the options for the rollback operation
type Options struct {
	ConfigFile       string
	SkipConfirmation bool
	// ExpectedDeployment, when non-zero, is the only deployment number
	// rollback may stop (--deployment). Zero stops whatever is in flight.
	ExpectedDeployment int32
}
