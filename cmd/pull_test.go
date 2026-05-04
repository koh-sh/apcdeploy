package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPullCommand(t *testing.T) {
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := newPullCmd()
			cmd.SetArgs(tt.args)

			err := cmd.ParseFlags(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseFlags() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPullCommandStructure(t *testing.T) {
	cmd := newPullCmd()

	if cmd.Use != "pull" {
		t.Errorf("Use = %v, want pull", cmd.Use)
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

func TestRunPull(t *testing.T) {
	tests := []struct {
		name       string
		setupFiles func(t *testing.T, dir string) string
		args       []string
		wantErr    bool
	}{
		{
			name: "missing config file",
			setupFiles: func(t *testing.T, dir string) string {
				return filepath.Join(dir, "nonexistent.yml")
			},
			args:    []string{},
			wantErr: true,
		},
		{
			name: "invalid config file",
			setupFiles: func(t *testing.T, dir string) string {
				configPath := filepath.Join(dir, "invalid.yml")
				err := os.WriteFile(configPath, []byte("invalid: yaml: content:\n  - bad"), 0o644)
				if err != nil {
					t.Fatalf("Failed to create test file: %v", err)
				}
				return configPath
			},
			args:    []string{},
			wantErr: true,
		},
		{
			name: "valid config but AWS error",
			setupFiles: func(t *testing.T, dir string) string {
				configPath := filepath.Join(dir, "valid.yml")
				content := `application: test-app
environment: test-env
configuration_profile: test-profile
deployment_strategy: test-strategy
data_file: data.json
`
				err := os.WriteFile(configPath, []byte(content), 0o644)
				if err != nil {
					t.Fatalf("Failed to create test file: %v", err)
				}
				return configPath
			},
			args:    []string{},
			wantErr: true, // Will fail due to AWS credentials/connection
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory
			tmpDir, err := os.MkdirTemp("", "pull-test-*")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			// Setup test files
			configPath := tt.setupFiles(t, tmpDir)

			// Reset global flags
			configFiles = []string{configPath}

			// Create command
			cmd := newPullCmd()
			cmd.SetArgs(tt.args)

			// Execute command
			err = runPull(cmd, tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("runPull() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestRunPull_MultiConfigOrchestratorAWSError exercises the multi-config
// branch in runPull through to the orchestrator, expecting AWS to fail
// (no credentials in the test environment). This covers the branch
// from LoadAll success → orchestrator setup → renderBatchSummary, which
// the load-error test exits before reaching.
func TestRunPull_MultiConfigOrchestratorAWSError(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "pull-multi-orch-*")
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
	silent = true // suppress orchestrator output during test
	t.Cleanup(func() {
		configFiles = []string{defaultConfigFile}
		silent = false
	})

	cmd := newPullCmd()
	err = runPull(cmd, nil)
	// AWS call (or credentials lookup) must fail because we have no
	// real backing. Either an aggregate "N target(s) failed" from the
	// orchestrator or any non-nil error is acceptable — the goal is to
	// exercise the multi-config branch.
	if err == nil {
		t.Fatal("expected error from multi-config orchestrator path, got nil")
	}
}

// TestRunPull_MultiConfigLoadError exercises the multi-config branch in
// runPull: when one of the supplied -c paths fails to load, the
// orchestrator never starts and the error is wrapped with
// "failed to load configurations".
func TestRunPull_MultiConfigLoadError(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "pull-multi-load-*")
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

	cmd := newPullCmd()
	err = runPull(cmd, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "failed to load configurations") {
		t.Errorf("err = %q, want substring 'failed to load configurations'", err.Error())
	}
	if !strings.Contains(err.Error(), missing) {
		t.Errorf("err = %q, want substring %q", err.Error(), missing)
	}
}

func TestPullCommandSilenceUsage(t *testing.T) {
	cmd := newPullCmd()

	// SilenceUsage should be true to prevent usage display on runtime errors
	if !cmd.SilenceUsage {
		t.Error("pull command should have SilenceUsage set to true")
	}
}
