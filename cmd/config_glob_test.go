package cmd

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestResolveConfigTargets(t *testing.T) {
	// Not parallel: each case mutates the configFiles package global.
	tests := []struct {
		name            string
		args            []string
		config          []string
		want            []string
		wantErrContains string
	}{
		{
			name:   "single literal path",
			args:   nil,
			config: []string{"apcdeploy.yml"},
			want:   []string{"apcdeploy.yml"},
		},
		{
			name:   "repeated -c values pass through",
			args:   nil,
			config: []string{"a.yml", "b.yml"},
			want:   []string{"a.yml", "b.yml"},
		},
		{
			name:            "positional args rejected (unquoted glob leftovers)",
			args:            []string{"env/stg.yml", "env/prod.yml"},
			config:          []string{"env/dev.yml"},
			wantErrContains: "quote glob patterns",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Cleanup(func() { configFiles = []string{defaultConfigFile} })
			configFiles = tt.config

			got, err := resolveConfigTargets(tt.args)
			if tt.wantErrContains != "" {
				if err == nil {
					t.Fatalf("expected error, got nil (result=%v)", got)
				}
				if !strings.Contains(err.Error(), tt.wantErrContains) {
					t.Errorf("error %q does not contain %q", err.Error(), tt.wantErrContains)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExpandConfigGlobs(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	for _, name := range []string{"dev.yml", "stg.yml", "prod.yml"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("region: us-east-1\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	pattern := filepath.Join(dir, "*.yml")
	dev := filepath.Join(dir, "dev.yml")
	prod := filepath.Join(dir, "prod.yml")
	stg := filepath.Join(dir, "stg.yml")

	tests := []struct {
		name            string
		paths           []string
		want            []string
		wantErrContains string
	}{
		{
			name:  "glob expands and sorts",
			paths: []string{pattern},
			want:  []string{dev, prod, stg}, // filepath.Glob returns sorted
		},
		{
			name:  "literal path without metachar passes through unchanged",
			paths: []string{filepath.Join(dir, "missing.yml")},
			want:  []string{filepath.Join(dir, "missing.yml")},
		},
		{
			name:  "duplicates removed preserving first occurrence",
			paths: []string{dev, pattern},
			want:  []string{dev, prod, stg},
		},
		{
			name:  "multiple globs preserve argument order",
			paths: []string{filepath.Join(dir, "s*.yml"), filepath.Join(dir, "d*.yml")},
			want:  []string{stg, dev},
		},
		{
			name:            "pattern matching nothing is an error",
			paths:           []string{filepath.Join(dir, "nope-*.yml")},
			wantErrContains: "no files matched pattern",
		},
		{
			name:            "malformed pattern is an error",
			paths:           []string{filepath.Join(dir, "[")},
			wantErrContains: "invalid glob pattern",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := expandConfigGlobs(tt.paths)
			if tt.wantErrContains != "" {
				if err == nil {
					t.Fatalf("expected error, got nil (result=%v)", got)
				}
				if !strings.Contains(err.Error(), tt.wantErrContains) {
					t.Errorf("error %q does not contain %q", err.Error(), tt.wantErrContains)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}
