package config

import (
	"fmt"
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
	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
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
	config.DataFile = resolveDataFilePath(absConfigPath, config.DataFile)

	return &config, nil
}

// resolveDataFilePath resolves a data file path relative to the config file
func resolveDataFilePath(configPath, dataFile string) string {
	// If data file is already absolute, return as-is
	if filepath.IsAbs(dataFile) {
		return dataFile
	}

	// Otherwise, resolve relative to config file directory
	configDir := filepath.Dir(configPath)
	return filepath.Join(configDir, dataFile)
}
