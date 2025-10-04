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
}

// Result contains the result of initialization
type Result struct {
	AppID       string
	AppName     string
	ProfileID   string
	ProfileName string
	ProfileType string
	EnvID       string
	EnvName     string
	VersionInfo *awsInternal.ConfigVersionInfo
	ConfigFile  string
	DataFile    string
}

// ProgressReporter is an interface for reporting progress
type ProgressReporter interface {
	Progress(message string)
	Success(message string)
	Warning(message string)
}
