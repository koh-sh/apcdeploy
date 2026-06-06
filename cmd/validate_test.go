package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateCommandStructure(t *testing.T) {
	cmd := newValidateCmd()

	if cmd.Use != "validate" {
		t.Errorf("Use = %v, want validate", cmd.Use)
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
	if !cmd.SilenceUsage {
		t.Error("validate command should have SilenceUsage set to true")
	}
	for _, flagName := range []string{"parallel", "continue-on-error"} {
		if cmd.Flags().Lookup(flagName) == nil {
			t.Errorf("flag %q not registered", flagName)
		}
	}
}

func TestRunValidate(t *testing.T) {
	tests := []struct {
		name       string
		setupFiles func(t *testing.T, dir string) string
		wantErr    bool
	}{
		{
			name: "missing config file",
			setupFiles: func(t *testing.T, dir string) string {
				return filepath.Join(dir, "nonexistent.yml")
			},
			wantErr: true,
		},
		{
			name: "invalid config file",
			setupFiles: func(t *testing.T, dir string) string {
				configPath := filepath.Join(dir, "invalid.yml")
				if err := os.WriteFile(configPath, []byte("invalid: yaml: content:\n  - bad"), 0o644); err != nil {
					t.Fatalf("Failed to create test file: %v", err)
				}
				return configPath
			},
			wantErr: true,
		},
		{
			name: "valid config but AWS error",
			setupFiles: func(t *testing.T, dir string) string {
				configPath := filepath.Join(dir, "valid.yml")
				content := "application: test-app\nenvironment: test-env\nconfiguration_profile: test-profile\nregion: us-east-1\ndata_file: data.json\n"
				if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
					t.Fatalf("Failed to create test file: %v", err)
				}
				if err := os.WriteFile(filepath.Join(dir, "data.json"), []byte("{}"), 0o644); err != nil {
					t.Fatalf("Failed to create data file: %v", err)
				}
				return configPath
			},
			wantErr: true, // Will fail due to AWS credentials/connection
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			configPath := tt.setupFiles(t, tmpDir)

			configFiles = []string{configPath}
			silent = true
			t.Cleanup(func() {
				configFiles = []string{defaultConfigFile}
				silent = false
			})

			cmd := newValidateCmd()
			err := runValidate(cmd, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("runValidate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRunValidate_MultiConfigLoadError(t *testing.T) {
	tmpDir := t.TempDir()

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

	cmd := newValidateCmd()
	err := runValidate(cmd, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "failed to load configurations") {
		t.Errorf("err = %q, want substring 'failed to load configurations'", err.Error())
	}
}
