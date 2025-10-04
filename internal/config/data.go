package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/goccy/go-yaml"
)

const (
	// MaxDataFileSize is the maximum size for configuration data (2MB)
	MaxDataFileSize = 2 * 1024 * 1024
)

// LoadDataFile loads a configuration data file
func LoadDataFile(path string) ([]byte, error) {
	// Check file size first
	if err := CheckDataFileSize(path); err != nil {
		return nil, err
	}

	// Read file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read data file: %w", err)
	}

	return data, nil
}

// DetermineContentType determines the content type based on file extension
func DetermineContentType(path string) string {
	ext := filepath.Ext(path)
	switch ext {
	case ".json":
		return "application/json"
	case ".yaml", ".yml":
		return "application/x-yaml"
	default:
		return "text/plain"
	}
}

// ValidateDataFile validates the syntax of a configuration data file
func ValidateDataFile(data []byte, contentType string) error {
	switch contentType {
	case "application/json":
		var js any
		if err := json.Unmarshal(data, &js); err != nil {
			return fmt.Errorf("invalid JSON: %w", err)
		}
	case "application/x-yaml":
		var y any
		if err := yaml.Unmarshal(data, &y); err != nil {
			return fmt.Errorf("invalid YAML: %w", err)
		}
	case "text/plain":
		// Text files are always valid
		return nil
	default:
		return fmt.Errorf("unsupported content type: %s", contentType)
	}
	return nil
}

// CheckDataFileSize checks if a file is within the size limit
func CheckDataFileSize(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	if info.Size() > MaxDataFileSize {
		return fmt.Errorf("file size (%d bytes) exceeds maximum allowed size (%d bytes)", info.Size(), MaxDataFileSize)
	}

	return nil
}
