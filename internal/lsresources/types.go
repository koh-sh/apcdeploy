package lsresources

// ResourcesTree represents the hierarchical structure of AppConfig resources
type ResourcesTree struct {
	Region               string               `json:"region"`
	Applications         []Application        `json:"applications"`
	DeploymentStrategies []DeploymentStrategy `json:"deployment_strategies"`
}

// Application represents an AppConfig application with its child resources
type Application struct {
	Name         string                 `json:"name"`
	ID           string                 `json:"id"`
	Profiles     []ConfigurationProfile `json:"configuration_profiles"`
	Environments []Environment          `json:"environments"`
}

// ConfigurationProfile represents an AppConfig configuration profile
type ConfigurationProfile struct {
	Name string `json:"name"`
	ID   string `json:"id"`
}

// Environment represents an AppConfig environment
type Environment struct {
	Name string `json:"name"`
	ID   string `json:"id"`
}

// DeploymentStrategy represents an AppConfig deployment strategy
type DeploymentStrategy struct {
	Name                        string  `json:"name"`
	ID                          string  `json:"id"`
	Description                 string  `json:"description,omitempty"`
	DeploymentDurationInMinutes int32   `json:"deployment_duration_in_minutes,omitempty"`
	FinalBakeTimeInMinutes      int32   `json:"final_bake_time_in_minutes,omitempty"`
	GrowthFactor                float32 `json:"growth_factor,omitempty"`
	GrowthType                  string  `json:"growth_type,omitempty"`
	ReplicateTo                 string  `json:"replicate_to,omitempty"`
}
