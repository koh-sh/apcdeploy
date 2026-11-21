package config

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/goccy/go-yaml"
)

// LoadConfig loads and validates a configuration file
func LoadConfig(path string) (*Config, error) {
	// Read file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse YAML
	config, err := decodeConfig(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Set defaults
	config.setDefaults()

	// Validate
	if err := config.validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Resolve data file path if relative
	absConfigPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve config path: %w", err)
	}
	config.DataFile = ResolveDataFilePath(absConfigPath, config.DataFile)

	return &config, nil
}

// decodeConfig decodes an apcdeploy.yml body. Unknown keys and additional YAML
// documents are rejected: yaml.Unmarshal silently decodes only the first
// document and ignores unknown keys, so a typo (or keys smuggled into a later
// document) would otherwise fall back to defaults without any notice.
// GenerateConfigFile reuses it to verify that what it writes reads back
// unchanged.
func decodeConfig(data []byte) (Config, error) {
	var config Config
	dec := yaml.NewDecoder(bytes.NewReader(data), yaml.DisallowUnknownField())
	if err := dec.Decode(&config); err != nil && !errors.Is(err, io.EOF) {
		return Config{}, err
	}
	var extra any
	if err := dec.Decode(&extra); !errors.Is(err, io.EOF) {
		if err != nil {
			return Config{}, err
		}
		return Config{}, errors.New("multiple YAML documents are not supported")
	}
	return config, nil
}

// ResolveDataFilePath resolves a data file path relative to the config file.
// Absolute data file paths are returned as-is. Both the loader and init use
// this so the path recorded in apcdeploy.yml and the path written agree.
func ResolveDataFilePath(configPath, dataFile string) string {
	// If data file is already absolute, return as-is
	if filepath.IsAbs(dataFile) {
		return dataFile
	}

	// Otherwise, resolve relative to config file directory
	configDir := filepath.Dir(configPath)
	return filepath.Join(configDir, dataFile)
}
