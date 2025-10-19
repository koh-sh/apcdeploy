package init

import (
	awsInternal "github.com/koh-sh/apcdeploy/internal/aws"
)

// Options contains all options for initialization
type Options struct {
	Application string
	Profile     string
	Environment string
	Region      string
	ConfigFile  string
	OutputData  string
	Force       bool
	Silent      bool
}

// Result contains the result of initialization
type Result struct {
	AppID              string
	AppName            string
	ProfileID          string
	ProfileName        string
	ProfileType        string
	EnvID              string
	EnvName            string
	DeployedConfig     *awsInternal.DeployedConfigInfo
	ConfigFile         string
	DataFile           string
	DeploymentStrategy string
}
