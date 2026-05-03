package config

// Identifier returns the canonical "region/app/profile/env" string used
// throughout the CLI to label a single deployment target.
//
// The 4-tuple is the contract for target identity (see
// docs/design/multi-config.md F-06). The same function is used by the
// Targets reporter primitive and by the multi-config orchestrator so that
// the identifier shown in logs matches the identifier used for duplicate
// detection.
//
// region argument is used when cfg.Region is empty (e.g. resolved later
// from the AWS SDK default chain). When cfg.Region is set it always wins.
func Identifier(region string, cfg *Config) string {
	r := cfg.Region
	if r == "" {
		r = region
	}
	return r + "/" + cfg.Application + "/" + cfg.ConfigurationProfile + "/" + cfg.Environment
}
