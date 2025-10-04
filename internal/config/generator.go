package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/goccy/go-yaml"
)

// GenerateConfigFile generates an apcdeploy.yml file with the given parameters
func GenerateConfigFile(app, profile, env, dataFile, region, deploymentStrategy, outputPath string) error {
	// Check if file already exists
	if _, err := os.Stat(outputPath); err == nil {
		return fmt.Errorf("config file already exists at %s (use --force to overwrite)", outputPath)
	}

	// Use provided deployment strategy, or default to AllAtOnce if empty
	strategy := deploymentStrategy
	if strategy == "" {
		strategy = "AppConfig.AllAtOnce"
	}

	// Create config structure
	cfg := Config{
		Application:          app,
		ConfigurationProfile: profile,
		Environment:          env,
		DataFile:             dataFile,
		DeploymentStrategy:   strategy,
		Region:               region,
	}

	// Marshal to YAML
	data, err := yaml.Marshal(&cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write to file
	if err := os.WriteFile(outputPath, data, 0o644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// DetermineDataFileName determines the appropriate data file name based on content type
func DetermineDataFileName(contentType string) string {
	// Normalize content type (remove charset and other parameters)
	ct := strings.ToLower(strings.TrimSpace(contentType))
	if idx := strings.Index(ct, ";"); idx != -1 {
		ct = strings.TrimSpace(ct[:idx])
	}

	switch ct {
	case ContentTypeJSON:
		return "data.json"
	case ContentTypeYAML, "application/yaml":
		return "data.yaml"
	case ContentTypeText:
		return "data.txt"
	default:
		// Default to JSON for unknown types
		return "data.json"
	}
}

// WriteDataFile writes configuration data to a file with appropriate formatting
// For FeatureFlags profile type, it removes _updatedAt and _createdAt fields
func WriteDataFile(content []byte, contentType, outputPath, profileType string) error {
	// Check if file already exists
	if _, err := os.Stat(outputPath); err == nil {
		return fmt.Errorf("data file already exists at %s (use --force to overwrite)", outputPath)
	}

	// Normalize content type
	ct := strings.ToLower(strings.TrimSpace(contentType))
	if idx := strings.Index(ct, ";"); idx != -1 {
		ct = strings.TrimSpace(ct[:idx])
	}

	var dataToWrite []byte
	var err error

	// Format based on content type
	switch ct {
	case ContentTypeJSON:
		// Format JSON with indentation
		dataToWrite, err = formatJSON(content, profileType)
		if err != nil {
			return fmt.Errorf("failed to format JSON: %w", err)
		}
	default:
		// Write as-is for YAML and text
		dataToWrite = content
	}

	// Write to file
	if err := os.WriteFile(outputPath, dataToWrite, 0o644); err != nil {
		return fmt.Errorf("failed to write data file: %w", err)
	}

	return nil
}

// formatJSON formats JSON data with proper indentation
// For FeatureFlags profile type, it removes _updatedAt and _createdAt fields recursively
func formatJSON(data []byte, profileType string) ([]byte, error) {
	var obj any
	if err := json.Unmarshal(data, &obj); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	// For FeatureFlags, remove _updatedAt and _createdAt fields recursively
	if profileType == "AWS.AppConfig.FeatureFlags" {
		obj = removeTimestampFields(obj)
	}

	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(obj); err != nil {
		return nil, fmt.Errorf("failed to encode JSON: %w", err)
	}

	return buf.Bytes(), nil
}

// removeTimestampFields recursively removes _updatedAt and _createdAt from all maps in the object
func removeTimestampFields(obj any) any {
	switch v := obj.(type) {
	case map[string]any:
		// Remove timestamp fields from this map
		delete(v, "_updatedAt")
		delete(v, "_createdAt")
		// Recursively process all values in the map
		for key, value := range v {
			v[key] = removeTimestampFields(value)
		}
		return v
	case []any:
		// Recursively process all elements in the array
		for i, value := range v {
			v[i] = removeTimestampFields(value)
		}
		return v
	default:
		// Return primitive values as-is
		return v
	}
}
