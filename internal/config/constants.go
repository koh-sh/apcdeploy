package config

const (
	// MaxConfigSize is the maximum size for configuration data (2MB)
	MaxConfigSize = 2 * 1024 * 1024

	// Profile types
	// ProfileTypeFeatureFlags represents AWS AppConfig FeatureFlags profile type
	ProfileTypeFeatureFlags = "AWS.AppConfig.FeatureFlags"

	// ProfileTypeFreeform represents AWS AppConfig Freeform profile type
	ProfileTypeFreeform = "AWS.Freeform"

	// Deployment strategy prefixes
	// StrategyPrefixPredefined is the prefix for predefined AWS AppConfig deployment strategies
	StrategyPrefixPredefined = "AppConfig."

	// Content types
	// ContentTypeJSON represents JSON content type
	ContentTypeJSON = "application/json"

	// ContentTypeYAML represents YAML content type
	ContentTypeYAML = "application/x-yaml"

	// ContentTypeText represents plain text content type
	ContentTypeText = "text/plain"
)
