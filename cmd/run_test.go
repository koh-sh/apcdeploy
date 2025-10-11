package cmd

import (
	"testing"
)

func TestRunCommand(t *testing.T) {
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
			name:    "wait-deploy flag",
			args:    []string{"--wait-deploy"},
			wantErr: false,
		},
		{
			name:    "wait-bake flag",
			args:    []string{"--wait-bake"},
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
			runConfigFile = "apcdeploy.yml"
			runWaitDeploy = false
			runWaitBake = false
			runTimeout = DefaultDeploymentTimeout

			cmd := newRunCmd()
			cmd.SetArgs(tt.args)

			err := cmd.ParseFlags(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseFlags() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRunCommandFlags(t *testing.T) {
	runConfigFile = "apcdeploy.yml"
	runWaitDeploy = false
	runWaitBake = false
	runTimeout = DefaultDeploymentTimeout

	cmd := newRunCmd()

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
			defaultValue: "600",
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

func TestRunCommandWaitFlags(t *testing.T) {
	runConfigFile = "apcdeploy.yml"
	runWaitDeploy = false
	runWaitBake = false
	runTimeout = DefaultDeploymentTimeout

	cmd := newRunCmd()

	tests := []struct {
		name     string
		flagName string
	}{
		{
			name:     "wait-deploy flag exists",
			flagName: "wait-deploy",
		},
		{
			name:     "wait-bake flag exists",
			flagName: "wait-bake",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag := cmd.Flags().Lookup(tt.flagName)
			if flag == nil {
				t.Errorf("Flag %s not found", tt.flagName)
				return
			}

			if flag.DefValue != "false" {
				t.Errorf("Flag %s default = %v, want false", tt.flagName, flag.DefValue)
			}
		})
	}
}

func TestRunTimeoutValidation(t *testing.T) {
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
			runTimeout = tt.timeout
			err := runRun(nil, nil)

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

func TestRunCommandSilenceUsage(t *testing.T) {
	cmd := newRunCmd()

	// SilenceUsage should be true to prevent usage display on runtime errors
	if !cmd.SilenceUsage {
		t.Error("run command should have SilenceUsage set to true")
	}
}
