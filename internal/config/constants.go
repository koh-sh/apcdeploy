package config

import "time"

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

	// DefaultDeploymentStrategy is the default deployment strategy when none is specified
	DefaultDeploymentStrategy = "AppConfig.AllAtOnce"

	// DefaultPollingInterval is the default interval for polling deployment status
	DefaultPollingInterval = 5 * time.Second

	// Content types
	// ContentTypeJSON represents JSON content type
	ContentTypeJSON = "application/json"

	// ContentTypeYAML represents YAML content type
	ContentTypeYAML = "application/x-yaml"

	// ContentTypeText represents plain text content type
	ContentTypeText = "text/plain"
)
