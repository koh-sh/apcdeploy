package config

import "fmt"

// Config represents the apcdeploy.yml configuration file
type Config struct {
	Application          string `yaml:"application"`
	ConfigurationProfile string `yaml:"configuration_profile"`
	Environment          string `yaml:"environment"`
	DeploymentStrategy   string `yaml:"deployment_strategy"`
	DataFile             string `yaml:"data_file"`
	Region               string `yaml:"region,omitempty"`
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
	return nil
}

// setDefaults sets default values for optional fields
func (c *Config) setDefaults() {
	if c.DeploymentStrategy == "" {
		c.DeploymentStrategy = "AppConfig.AllAtOnce"
	}
}

// DeploymentConfig represents detailed deployment configuration
// This will be expanded in later Epics as needed
type DeploymentConfig struct {
	Description string            `yaml:"description,omitempty"`
	Tags        map[string]string `yaml:"tags,omitempty"`
}
