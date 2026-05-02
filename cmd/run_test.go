package cmd

import (
	"strings"
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
			// Reset global flags for each test. We touch every flag-bound
			// global so `go test -shuffle=on` can't expose ordering bugs.
			configFile = "apcdeploy.yml"
			runWaitDeploy = false
			runWaitBake = false
			runTimeout = DefaultDeploymentTimeout
			runForce = false
			runDescription = ""

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
	configFile = "apcdeploy.yml"
	runWaitDeploy = false
	runWaitBake = false
	runTimeout = DefaultDeploymentTimeout
	runForce = false
	runDescription = ""

	cmd := newRunCmd()

	tests := []struct {
		name         string
		flagName     string
		defaultValue string
	}{
		{
			name:         "timeout flag has default",
			flagName:     "timeout",
			defaultValue: "1800",
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
	configFile = "apcdeploy.yml"
	runWaitDeploy = false
	runWaitBake = false
	runTimeout = DefaultDeploymentTimeout
	runForce = false
	runDescription = ""

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
			errMsg:  "timeout must be a non-negative value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// resolveDescription inside runRun reads cmd.Flags(), so a nil cmd
			// would panic. Build a real run command, then override the timeout
			// global AFTER newRunCmd() — IntVar registration resets the flag
			// to its default during construction.
			cmd := newRunCmd()
			runTimeout = tt.timeout
			runDescription = ""

			err := runRun(cmd, nil)

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

// TestResolveDescription verifies the default-vs-explicit behavior:
//   - flag not passed → defaultDescription marker
//   - --description "x" → "x"
//   - --description "" (explicit empty) → "" (opt-out from default)
//
// We use a freshly constructed run command so the test owns the flag state
// and isn't affected by leftover globals from neighboring tests.
func TestResolveDescription(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "not passed uses default", args: []string{}, want: defaultDescription},
		{name: "explicit value", args: []string{"--description", "hotfix"}, want: "hotfix"},
		{name: "explicit empty opts out", args: []string{"--description", ""}, want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runDescription = ""
			cmd := newRunCmd()
			if err := cmd.ParseFlags(tt.args); err != nil {
				t.Fatalf("ParseFlags: %v", err)
			}
			got := resolveDescription(cmd, runDescription)
			if got != tt.want {
				t.Errorf("resolveDescription = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestValidateDescription covers the 1024-rune client-side guard. We exercise
// the boundary explicitly (1024 OK, 1025 rejected) for both ASCII and a
// multibyte rune so a regression to byte-counting (len(s) > 1024) would be
// caught — "あ" is 3 UTF-8 bytes, so 1024 of them is 3072 bytes.
func TestValidateDescription(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{name: "empty", input: "", wantErr: false},
		{name: "short", input: "hotfix", wantErr: false},
		{name: "exactly 1024 ascii", input: strings.Repeat("a", 1024), wantErr: false},
		{name: "1025 ascii rejected", input: strings.Repeat("a", 1025), wantErr: true},
		{name: "exactly 1024 multibyte", input: strings.Repeat("あ", 1024), wantErr: false},
		{name: "1025 multibyte rejected", input: strings.Repeat("あ", 1025), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateDescription(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateDescription(runes=%d) error = %v, wantErr %v", len([]rune(tt.input)), err, tt.wantErr)
			}
		})
	}
}
