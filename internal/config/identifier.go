package config

// Identifier returns the canonical "region/app/profile/env" string used
// throughout the CLI to label a single deployment target.
//
// The 4-tuple is the contract for target identity. The same function is
// used by the Targets reporter primitive and by the multi-config
// orchestrator so that the identifier shown in logs matches the
// identifier used for duplicate detection.
//
// Region is required in apcdeploy.yml as of v1.0 (see Config.validate),
// so this function trusts cfg.Region and does not consult any fallback.
func Identifier(cfg *Config) string {
	return cfg.Region + "/" + cfg.Application + "/" + cfg.ConfigurationProfile + "/" + cfg.Environment
}
