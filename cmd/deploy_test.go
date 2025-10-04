package cmd

import (
	"testing"
)

func TestDeployCommand(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "no config file specified uses default",
			args:    []string{},
			wantErr: false,
		},
		{
			name:    "custom config file",
			args:    []string{"--config", "custom.yml"},
			wantErr: false,
		},
		{
			name:    "no-wait flag",
			args:    []string{"--no-wait"},
			wantErr: false,
		},
		{
			name:    "custom timeout",
			args:    []string{"--timeout", "600"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset global flags for each test
			deployConfigFile = "apcdeploy.yml"
			deployNoWait = false
			deployTimeout = 300

			cmd := newDeployCmd()
			cmd.SetArgs(tt.args)

			err := cmd.ParseFlags(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseFlags() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDeployCommandFlags(t *testing.T) {
	deployConfigFile = "apcdeploy.yml"
	deployNoWait = false
	deployTimeout = 300

	cmd := newDeployCmd()

	tests := []struct {
		name         string
		flagName     string
		defaultValue string
	}{
		{
			name:         "config flag has default",
			flagName:     "config",
			defaultValue: "apcdeploy.yml",
		},
		{
			name:         "timeout flag has default",
			flagName:     "timeout",
			defaultValue: "300",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag := cmd.Flags().Lookup(tt.flagName)
			if flag == nil {
				t.Errorf("Flag %s not found", tt.flagName)
				return
			}

			if flag.DefValue != tt.defaultValue {
				t.Errorf("Flag %s default = %v, want %v", tt.flagName, flag.DefValue, tt.defaultValue)
			}
		})
	}
}

func TestDeployCommandNoWaitFlag(t *testing.T) {
	deployConfigFile = "apcdeploy.yml"
	deployNoWait = false
	deployTimeout = 300

	cmd := newDeployCmd()

	flag := cmd.Flags().Lookup("no-wait")
	if flag == nil {
		t.Error("Flag no-wait not found")
		return
	}

	if flag.DefValue != "false" {
		t.Errorf("Flag no-wait default = %v, want false", flag.DefValue)
	}
}

func TestDeployTimeoutValidation(t *testing.T) {
	tests := []struct {
		name    string
		timeout int
		wantErr bool
		errMsg  string
	}{
		{
			name:    "negative timeout is invalid",
			timeout: -1,
			wantErr: true,
			errMsg:  "timeout must be a positive value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deployTimeout = tt.timeout
			err := runDeploy(nil, nil)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected timeout validation error, got nil")
				} else if err.Error() != tt.errMsg {
					t.Errorf("Expected error message '%s', got: %v", tt.errMsg, err)
				}
			}
		})
	}
}

func TestDeployCommandSilenceUsage(t *testing.T) {
	cmd := newDeployCmd()

	// SilenceUsage should be true to prevent usage display on runtime errors
	if !cmd.SilenceUsage {
		t.Error("deploy command should have SilenceUsage set to true")
	}
}
