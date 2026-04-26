package config

import (
	"fmt"
	"os"
)

// LoadDataFile loads a configuration data file
func LoadDataFile(path string) ([]byte, error) {
	// Check file size first
	if err := checkDataFileSize(path); err != nil {
		return nil, err
	}

	// Read file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read data file: %w", err)
	}

	return data, nil
}

// checkDataFileSize checks if a file is within the size limit
func checkDataFileSize(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	if info.Size() > MaxConfigSize {
		return fmt.Errorf("file size (%d bytes) exceeds maximum allowed size (%d bytes)", info.Size(), MaxConfigSize)
	}

	return nil
}
