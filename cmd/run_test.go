package cmd

import (
	"os"
	"path/filepath"
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
			configFiles = []string{"apcdeploy.yml"}
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
	configFiles = []string{"apcdeploy.yml"}
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
	configFiles = []string{"apcdeploy.yml"}
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
		{name: "tab allowed", input: "release\tnotes", wantErr: false},
		{name: "newline allowed", input: "release\nnotes", wantErr: false},
		{name: "carriage return allowed", input: "release\rnotes", wantErr: false},
		{name: "null byte rejected", input: "bad\x00value", wantErr: true},
		{name: "ANSI escape rejected", input: "\x1b[31mred\x1b[0m", wantErr: true},
		{name: "DEL rejected", input: "bad\x7fvalue", wantErr: true},
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

// TestValidateRunFlags asserts the cmd-layer flag invariants are
// rejected before LoadAll/orchestrator setup. Validation lives in the
// cmd layer (not RunOnTarget) so the user gets the same error
// regardless of -c count and the orchestrator never starts when the
// flags are bogus.
func TestValidateRunFlags(t *testing.T) {
	tests := []struct {
		name       string
		waitDeploy bool
		waitBake   bool
		timeout    int
		wantErr    bool
		wantSub    string
	}{
		{name: "default flags pass", timeout: DefaultDeploymentTimeout},
		{name: "positive timeout passes", timeout: 1},
		{
			name:    "zero timeout rejected",
			timeout: 0,
			wantErr: true,
			wantSub: "timeout must be greater than 0",
		},
		{
			name:    "negative timeout rejected",
			timeout: -1,
			wantErr: true,
			wantSub: "timeout must be greater than 0",
		},
		{
			name:       "both wait flags rejected",
			waitDeploy: true,
			waitBake:   true,
			timeout:    DefaultDeploymentTimeout,
			wantErr:    true,
			wantSub:    "--wait-deploy and --wait-bake cannot be used together",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runWaitDeploy = tt.waitDeploy
			runWaitBake = tt.waitBake
			runTimeout = tt.timeout
			t.Cleanup(func() {
				runWaitDeploy = false
				runWaitBake = false
				runTimeout = DefaultDeploymentTimeout
			})

			err := validateRunFlags()
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if !strings.Contains(err.Error(), tt.wantSub) {
					t.Errorf("err = %q, want substring %q", err.Error(), tt.wantSub)
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// TestRunRun_MultiConfigLoadError exercises the multi-config branch in
// runRun: when one of the supplied -c paths fails to load, the
// orchestrator never starts and the error wraps "failed to load
// configurations".
func TestRunRun_MultiConfigLoadError(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "run-multi-load-*")
	if err != nil {
		t.Fatalf("temp: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	good := filepath.Join(tmpDir, "good.yml")
	if err := os.WriteFile(good, []byte("application: a\nconfiguration_profile: p\nenvironment: e\nregion: us-east-1\ndata_file: data.json\n"), 0o644); err != nil {
		t.Fatalf("good: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "data.json"), []byte("{}"), 0o644); err != nil {
		t.Fatalf("data: %v", err)
	}
	missing := filepath.Join(tmpDir, "missing.yml")

	configFiles = []string{good, missing}
	t.Cleanup(func() { configFiles = []string{defaultConfigFile} })

	cmd := newRunCmd()
	err = runRun(cmd, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "failed to load configurations") {
		t.Errorf("err = %q, want substring 'failed to load configurations'", err.Error())
	}
}

// TestRunRun_MultiConfigOrchestratorAWSError exercises the multi-config
// branch in runRun through to the orchestrator setup + summary render.
// AWS calls must fail for this to work without credentials.
func TestRunRun_MultiConfigOrchestratorAWSError(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "run-multi-orch-*")
	if err != nil {
		t.Fatalf("temp: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	makeConfig := func(name, env string) string {
		path := filepath.Join(tmpDir, name)
		body := "application: a\nconfiguration_profile: p\nregion: us-east-1\ndata_file: data.json\nenvironment: " + env + "\n"
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatalf("config: %v", err)
		}
		return path
	}
	a := makeConfig("a.yml", "dev")
	b := makeConfig("b.yml", "prod")
	if err := os.WriteFile(filepath.Join(tmpDir, "data.json"), []byte("{}"), 0o644); err != nil {
		t.Fatalf("data: %v", err)
	}

	configFiles = []string{a, b}
	silent = true
	t.Cleanup(func() {
		configFiles = []string{defaultConfigFile}
		silent = false
	})

	cmd := newRunCmd()
	err = runRun(cmd, nil)
	if err == nil {
		t.Fatal("expected error from multi-config orchestrator path, got nil")
	}
}
