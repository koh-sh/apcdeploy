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
func GenerateConfigFile(app, profile, env, dataFile, region, deploymentStrategy, outputPath string, force bool) error {
	// Check if file already exists
	if _, err := os.Stat(outputPath); err == nil && !force {
		return fmt.Errorf("config file already exists at %s (use --force to overwrite)", outputPath)
	}

	// Use provided deployment strategy, or default to AllAtOnce if empty
	strategy := deploymentStrategy
	if strategy == "" {
		strategy = DefaultDeploymentStrategy
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
	return "data" + ExtensionForContentType(contentType)
}

// ExtensionForContentType returns the file extension (including the leading dot)
// for the given AppConfig content type. Unknown types fall back to ".json".
func ExtensionForContentType(contentType string) string {
	ct := strings.ToLower(strings.TrimSpace(contentType))
	if idx := strings.Index(ct, ";"); idx != -1 {
		ct = strings.TrimSpace(ct[:idx])
	}

	switch ct {
	case ContentTypeJSON:
		return ".json"
	case ContentTypeYAML, "application/yaml":
		return ".yaml"
	case ContentTypeText:
		return ".txt"
	default:
		return ".json"
	}
}

// WriteDataFile writes configuration data to a file with appropriate formatting
// For FeatureFlags profile type, it removes _updatedAt and _createdAt fields
func WriteDataFile(content []byte, contentType, outputPath, profileType string, force bool) error {
	// Check if file already exists
	if _, err := os.Stat(outputPath); err == nil && !force {
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
	if profileType == ProfileTypeFeatureFlags {
		obj = RemoveTimestampFieldsRecursive(obj)
	}

	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(obj); err != nil {
		return nil, fmt.Errorf("failed to encode JSON: %w", err)
	}

	return buf.Bytes(), nil
}
