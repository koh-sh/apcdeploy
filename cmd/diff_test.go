package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDiffCommand(t *testing.T) {
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
			name:    "exit-nonzero flag",
			args:    []string{"--exit-nonzero"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
	// Config flag is tested in root_test.go as a persistent flag
	cmd := newDiffCmd()

	// Test exit-nonzero flag
	flag := cmd.Flags().Lookup("exit-nonzero")
	if flag == nil {
		t.Error("Flag exit-nonzero not found")
	}
}

func TestRunDiffInvalidConfig(t *testing.T) {
	// Reset flags
	configFiles = []string{"nonexistent.yml"}

	err := runDiff(nil, nil)
	if err == nil {
		t.Error("Expected error for nonexistent config, got nil")
	}
}

func TestDiffCommandSilenceUsage(t *testing.T) {
	cmd := newDiffCmd()

	// SilenceUsage should be true to prevent usage display on runtime errors
	if !cmd.SilenceUsage {
		t.Error("diff command should have SilenceUsage set to true")
	}
}

func TestDiffCommandExitNonzeroFlag(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		expectedFlag bool
	}{
		{
			name:         "exit-nonzero flag not specified",
			args:         []string{},
			expectedFlag: false,
		},
		{
			name:         "exit-nonzero flag specified",
			args:         []string{"--exit-nonzero"},
			expectedFlag: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset flags
			configFiles = []string{"apcdeploy.yml"}
			diffExitNonzero = false

			cmd := newDiffCmd()
			cmd.SetArgs(tt.args)

			err := cmd.ParseFlags(tt.args)
			if err != nil {
				t.Errorf("ParseFlags() error = %v", err)
			}

			if diffExitNonzero != tt.expectedFlag {
				t.Errorf("diffExitNonzero = %v, want %v", diffExitNonzero, tt.expectedFlag)
			}
		})
	}
}

// TestRunDiff_MultiConfigLoadError exercises the multi-config branch in
// runDiff: when one of the supplied -c paths fails to load, the
// orchestrator never starts and the error wraps "failed to load
// configurations".
func TestRunDiff_MultiConfigLoadError(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "diff-multi-load-*")
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

	cmd := newDiffCmd()
	err = runDiff(cmd, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "failed to load configurations") {
		t.Errorf("err = %q, want substring 'failed to load configurations'", err.Error())
	}
}

// TestRunDiff_MultiConfigOrchestratorAWSError exercises the multi-config
// branch in runDiff through to the orchestrator (covers payload buffer
// setup, flushDiffPayloads, renderBatchSummary). AWS calls must fail
// for this to work without credentials.
func TestRunDiff_MultiConfigOrchestratorAWSError(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "diff-multi-orch-*")
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

	cmd := newDiffCmd()
	err = runDiff(cmd, nil)
	if err == nil {
		t.Fatal("expected error from multi-config orchestrator path, got nil")
	}
}
