package cmd

import (
	"testing"
)

func TestLsResourcesCommand(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "no flags specified",
			args:    []string{},
			wantErr: false,
		},
		{
			name:    "with region flag",
			args:    []string{"--region", "us-east-1"},
			wantErr: false,
		},
		{
			name:    "with json flag",
			args:    []string{"--json"},
			wantErr: false,
		},
		{
			name:    "with show-strategies flag",
			args:    []string{"--show-strategies"},
			wantErr: false,
		},
		{
			name:    "with all flags",
			args:    []string{"--region", "us-west-2", "--json", "--show-strategies"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset global flags for each test
			lsResourcesRegion = ""
			lsResourcesJSON = false
			lsResourcesShowStrategies = false

			cmd := newLsResourcesCmd()
			cmd.SetArgs(tt.args)

			err := cmd.ParseFlags(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseFlags() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLsResourcesCommandStructure(t *testing.T) {
	cmd := newLsResourcesCmd()

	if cmd.Use != "ls-resources" {
		t.Errorf("Use = %v, want ls-resources", cmd.Use)
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

func TestLsResourcesCommandFlags(t *testing.T) {
	lsResourcesRegion = ""
	lsResourcesJSON = false
	lsResourcesShowStrategies = false

	cmd := newLsResourcesCmd()

	tests := []struct {
		name         string
		flagName     string
		defaultValue string
		flagType     string
	}{
		{
			name:         "region flag has default",
			flagName:     "region",
			defaultValue: "",
			flagType:     "string",
		},
		{
			name:         "json flag has default",
			flagName:     "json",
			defaultValue: "false",
			flagType:     "bool",
		},
		{
			name:         "show-strategies flag has default",
			flagName:     "show-strategies",
			defaultValue: "false",
			flagType:     "bool",
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

			if flag.Value.Type() != tt.flagType {
				t.Errorf("Flag %s type = %v, want %v", tt.flagName, flag.Value.Type(), tt.flagType)
			}
		})
	}
}

func TestLsResourcesCommandSilenceUsage(t *testing.T) {
	cmd := newLsResourcesCmd()

	// SilenceUsage should be true to prevent usage display on runtime errors
	if !cmd.SilenceUsage {
		t.Error("ls-resources command should have SilenceUsage set to true")
	}
}
