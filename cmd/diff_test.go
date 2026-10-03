package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/koh-sh/apcdeploy/internal/batch"
	reportertest "github.com/koh-sh/apcdeploy/internal/reporter/testing"
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

// TestRenderDiffResults covers the post-run output that runDiff emits once
// the orchestrator has closed the Targets block (issue #151): in-progress
// deployment warnings are written in argument order, as whole blocks, and
// still surface under --silent while the aggregate summary does not.
func TestRenderDiffResults(t *testing.T) {
	targets := []*batch.Target{
		{Identifier: "us-east-1/app/p/dev"},
		{Identifier: "us-east-1/app/p/stg"},
		{Identifier: "us-east-1/app/p/prod"},
	}
	const (
		devWarning  = "⚠ us-east-1/app/p/dev: Deployment #1 is currently DEPLOYING\nThe diff is calculated against the currently deploying version.\n"
		prodWarning = "⚠ us-east-1/app/p/prod: Deployment #9 is currently BAKING\nThe diff is calculated against the currently deploying version.\n"
		summaryLine = "3 ok, 0 no-op, 0 failed"
	)

	tests := []struct {
		name        string
		silent      bool
		wantSummary bool
	}{
		{name: "non-silent renders warnings before summary", silent: false, wantSummary: true},
		{name: "silent still renders warnings", silent: true, wantSummary: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			collector := batch.NewPayloadCollector(targets)
			// Complete out of argument order, as a parallel run would.
			collector.Set("us-east-1/app/p/prod", []byte("+prod\n"), true, "Deployment #9 is currently BAKING")
			collector.Set("us-east-1/app/p/stg", nil, false, "")
			collector.Set("us-east-1/app/p/dev", []byte("+dev\n"), true, "Deployment #1 is currently DEPLOYING")

			rep := &reportertest.MockReporter{}
			out := captureStderr(t, func() {
				renderDiffResults(rep, targets, collector, batch.Summary{OK: 3}, tt.silent)
			})

			devIdx := strings.Index(out, devWarning)
			prodIdx := strings.Index(out, prodWarning)
			if devIdx < 0 || prodIdx < 0 {
				t.Fatalf("expected both warning blocks on stderr; got %q", out)
			}
			if devIdx > prodIdx {
				t.Errorf("warnings must follow argument order (dev before prod); got %q", out)
			}
			if strings.Contains(out, "us-east-1/app/p/stg:") {
				t.Errorf("target without a warning must not be listed; got %q", out)
			}

			summaryIdx := strings.Index(out, summaryLine)
			switch {
			case tt.wantSummary && summaryIdx < 0:
				t.Errorf("expected summary line; got %q", out)
			case tt.wantSummary && summaryIdx < prodIdx:
				t.Errorf("warnings must precede the summary line; got %q", out)
			case !tt.wantSummary && summaryIdx >= 0:
				t.Errorf("summary must be suppressed under silent; got %q", out)
			}

			if !strings.Contains(string(rep.Stdout), "=== us-east-1/app/p/dev ===\n+dev\n") {
				t.Errorf("expected diff payloads flushed via Reporter.Diff; got %q", rep.Stdout)
			}
		})
	}
}
