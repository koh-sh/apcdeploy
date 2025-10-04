package cmd

import (
	"testing"
)

func TestDiffCommand(t *testing.T) {
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
			name:    "custom region",
			args:    []string{"--region", "us-west-2"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset global flags for each test
			diffConfigFile = "apcdeploy.yml"
			diffRegion = ""

			cmd := newDiffCmd()
			cmd.SetArgs(tt.args)

			err := cmd.ParseFlags(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseFlags() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDiffCommandStructure(t *testing.T) {
	cmd := newDiffCmd()

	if cmd.Use != "diff" {
		t.Errorf("Use = %v, want diff", cmd.Use)
	}

	if cmd.Short == "" {
		t.Error("Short description should not be empty")
	}

	if cmd.Long == "" {
		t.Error("Long description should not be empty")
	}

	if cmd.RunE == nil {
		t.Error("RunE should be set")
	}
}

func TestDiffCommandFlags(t *testing.T) {
	diffConfigFile = "apcdeploy.yml"
	diffRegion = ""

	cmd := newDiffCmd()

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
			name:         "region flag has default",
			flagName:     "region",
			defaultValue: "",
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

func TestRunDiffInvalidConfig(t *testing.T) {
	// Reset flags
	diffConfigFile = "nonexistent.yml"
	diffRegion = ""

	err := runDiff(nil, nil)
	if err == nil {
		t.Error("Expected error for nonexistent config, got nil")
	}
}
