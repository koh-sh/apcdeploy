package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/goccy/go-yaml"
)

func TestGenerateConfigFile(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name           string
		app            string
		profile        string
		env            string
		dataFile       string
		outputPath     string
		wantErr        bool
		validateConfig func(*testing.T, *Config)
	}{
		{
			name:       "generate basic config",
			app:        "test-app",
			profile:    "test-profile",
			env:        "test-env",
			dataFile:   "data.json",
			outputPath: filepath.Join(tempDir, "apcdeploy.yml"),
			wantErr:    false,
			validateConfig: func(t *testing.T, cfg *Config) {
				if cfg.Application != "test-app" {
					t.Errorf("expected application %q, got %q", "test-app", cfg.Application)
				}
				if cfg.ConfigurationProfile != "test-profile" {
					t.Errorf("expected profile %q, got %q", "test-profile", cfg.ConfigurationProfile)
				}
				if cfg.Environment != "test-env" {
					t.Errorf("expected environment %q, got %q", "test-env", cfg.Environment)
				}
				if cfg.DataFile != "data.json" {
					t.Errorf("expected data file %q, got %q", "data.json", cfg.DataFile)
				}
			},
		},
		{
			name:       "generate with custom data file",
			app:        "my-app",
			profile:    "my-profile",
			env:        "production",
			dataFile:   "config.yaml",
			outputPath: filepath.Join(tempDir, "custom.yml"),
			wantErr:    false,
			validateConfig: func(t *testing.T, cfg *Config) {
				if cfg.DataFile != "config.yaml" {
					t.Errorf("expected data file %q, got %q", "config.yaml", cfg.DataFile)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := GenerateConfigFile(tt.app, tt.profile, tt.env, tt.dataFile, tt.outputPath)

			if tt.wantErr && err == nil {
				t.Error("expected error but got none")
			} else if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if !tt.wantErr {
				// Verify file was created
				if _, err := os.Stat(tt.outputPath); os.IsNotExist(err) {
					t.Errorf("config file was not created at %s", tt.outputPath)
				}

				// Load and validate config
				data, err := os.ReadFile(tt.outputPath)
				if err != nil {
					t.Fatalf("failed to read generated config: %v", err)
				}

				var cfg Config
				if err := yaml.Unmarshal(data, &cfg); err != nil {
					t.Fatalf("failed to parse generated config: %v", err)
				}

				if tt.validateConfig != nil {
					tt.validateConfig(t, &cfg)
				}
			}
		})
	}
}

func TestGenerateConfigFileOverwrite(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "apcdeploy.yml")

	// Create initial file
	initialContent := "existing: content\n"
	if err := os.WriteFile(configPath, []byte(initialContent), 0o644); err != nil {
		t.Fatalf("failed to create initial file: %v", err)
	}

	// Try to generate - should fail without force flag
	err := GenerateConfigFile("app", "profile", "env", "data.json", configPath)
	if err == nil {
		t.Error("expected error when overwriting existing file, but got none")
	}
}

func TestDetermineDataFileName(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		want        string
	}{
		{
			name:        "json content type",
			contentType: "application/json",
			want:        "data.json",
		},
		{
			name:        "json with charset",
			contentType: "application/json; charset=utf-8",
			want:        "data.json",
		},
		{
			name:        "yaml content type",
			contentType: "application/x-yaml",
			want:        "data.yaml",
		},
		{
			name:        "yaml alternative content type",
			contentType: "application/yaml",
			want:        "data.yaml",
		},
		{
			name:        "text content type",
			contentType: "text/plain",
			want:        "data.txt",
		},
		{
			name:        "unknown content type defaults to json",
			contentType: "application/octet-stream",
			want:        "data.json",
		},
		{
			name:        "empty content type defaults to json",
			contentType: "",
			want:        "data.json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetermineDataFileName(tt.contentType)
			if got != tt.want {
				t.Errorf("DetermineDataFileName(%q) = %q, want %q", tt.contentType, got, tt.want)
			}
		})
	}
}

func TestWriteDataFile(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name        string
		content     []byte
		contentType string
		outputPath  string
		wantErr     bool
		validate    func(*testing.T, string)
	}{
		{
			name:        "write json data",
			content:     []byte(`{"key":"value"}`),
			contentType: "application/json",
			outputPath:  filepath.Join(tempDir, "data.json"),
			wantErr:     false,
			validate: func(t *testing.T, path string) {
				data, err := os.ReadFile(path)
				if err != nil {
					t.Fatalf("failed to read file: %v", err)
				}
				expected := "{\n  \"key\": \"value\"\n}\n"
				if string(data) != expected {
					t.Errorf("expected formatted JSON:\n%s\ngot:\n%s", expected, string(data))
				}
			},
		},
		{
			name:        "write json data with charset",
			content:     []byte(`{"foo":"bar"}`),
			contentType: "application/json; charset=utf-8",
			outputPath:  filepath.Join(tempDir, "data2.json"),
			wantErr:     false,
			validate: func(t *testing.T, path string) {
				data, err := os.ReadFile(path)
				if err != nil {
					t.Fatalf("failed to read file: %v", err)
				}
				expected := "{\n  \"foo\": \"bar\"\n}\n"
				if string(data) != expected {
					t.Errorf("expected formatted JSON:\n%s\ngot:\n%s", expected, string(data))
				}
			},
		},
		{
			name:        "write invalid json",
			content:     []byte(`{invalid`),
			contentType: "application/json",
			outputPath:  filepath.Join(tempDir, "invalid.json"),
			wantErr:     true,
		},
		{
			name:        "write yaml data",
			content:     []byte("key: value\n"),
			contentType: "application/x-yaml",
			outputPath:  filepath.Join(tempDir, "data.yaml"),
			wantErr:     false,
			validate: func(t *testing.T, path string) {
				data, err := os.ReadFile(path)
				if err != nil {
					t.Fatalf("failed to read file: %v", err)
				}
				if string(data) != "key: value\n" {
					t.Errorf("expected YAML content, got: %s", string(data))
				}
			},
		},
		{
			name:        "write text data",
			content:     []byte("plain text content"),
			contentType: "text/plain",
			outputPath:  filepath.Join(tempDir, "data.txt"),
			wantErr:     false,
			validate: func(t *testing.T, path string) {
				data, err := os.ReadFile(path)
				if err != nil {
					t.Fatalf("failed to read file: %v", err)
				}
				if string(data) != "plain text content" {
					t.Errorf("expected text content, got: %s", string(data))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := WriteDataFile(tt.content, tt.contentType, tt.outputPath)

			if tt.wantErr && err == nil {
				t.Error("expected error but got none")
			} else if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if !tt.wantErr && tt.validate != nil {
				tt.validate(t, tt.outputPath)
			}
		})
	}
}

func Test_formatJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   []byte
		want    string
		wantErr bool
	}{
		{
			name:    "format compact json",
			input:   []byte(`{"key":"value","nested":{"foo":"bar"}}`),
			want:    "{\n  \"key\": \"value\",\n  \"nested\": {\n    \"foo\": \"bar\"\n  }\n}\n",
			wantErr: false,
		},
		{
			name:    "format already formatted json",
			input:   []byte("{\n  \"key\": \"value\"\n}"),
			want:    "{\n  \"key\": \"value\"\n}\n",
			wantErr: false,
		},
		{
			name:    "invalid json",
			input:   []byte(`{invalid`),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := formatJSON(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if string(got) != tt.want {
					t.Errorf("expected:\n%s\ngot:\n%s", tt.want, string(got))
				}
			}
		})
	}
}
