package config

import (
	"fmt"
	"regexp"
)

// Config represents the apcdeploy.yml configuration file
type Config struct {
	Application          string `yaml:"application"`
	ConfigurationProfile string `yaml:"configuration_profile"`
	Environment          string `yaml:"environment"`
	DeploymentStrategy   string `yaml:"deployment_strategy"`
	DataFile             string `yaml:"data_file"`
	Region               string `yaml:"region"`
}

// validate checks if the configuration is valid
func (c *Config) validate() error {
	if c.Application == "" {
		return fmt.Errorf("application is required")
	}
	if c.ConfigurationProfile == "" {
		return fmt.Errorf("configuration_profile is required")
	}
	if c.Environment == "" {
		return fmt.Errorf("environment is required")
	}
	if c.DataFile == "" {
		return fmt.Errorf("data_file is required")
	}
	// Region became required in v1.0. Previously omitting it meant "use the
	// AWS SDK default region", but that made yml non-portable across machines
	// (env / profile differences) and let multi-config users silently collide
	// on identifier when several files all defaulted. Point users at
	// `init --force` rather than have them hand-edit yml.
	if c.Region == "" {
		return fmt.Errorf("region is required (add `region: <aws-region>` to apcdeploy.yml, or regenerate with `apcdeploy init --force`)")
	}
	return ValidateRegion(c.Region)
}

// regionRE matches a single lowercase DNS label such as "us-east-1" or
// "eusc-de-east-1". It is deliberately not a list of known regions, so new
// regions keep working.
var regionRE = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// ValidateRegion rejects a region that is not a single lowercase host label.
// The AWS SDK builds the endpoint host name from the region, so a value such
// as "s3.amazonaws.com" would otherwise change which host requests go to.
func ValidateRegion(region string) error {
	if !regionRE.MatchString(region) {
		return fmt.Errorf("invalid region %q: expected an AWS region name such as us-east-1", region)
	}
	return nil
}

// setDefaults sets default values for optional fields
func (c *Config) setDefaults() {
	if c.DeploymentStrategy == "" {
		c.DeploymentStrategy = DefaultDeploymentStrategy
	}
}
