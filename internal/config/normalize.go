package config

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/goccy/go-yaml"
)

// NormalizeJSON normalizes JSON content by parsing and re-formatting with sorted keys.
// For FeatureFlags profile type, it removes _updatedAt and _createdAt fields recursively.
//
// Parameters:
//   - content: JSON string to normalize
//   - profileType: AWS AppConfig profile type (e.g., "AWS.AppConfig.FeatureFlags")
//
// Returns:
//   - string: Normalized JSON with consistent formatting
//   - error: Any error during parsing or formatting
func NormalizeJSON(content string, profileType string) (string, error) {
	var data any
	if err := json.Unmarshal([]byte(content), &data); err != nil {
		return "", fmt.Errorf("invalid JSON: %w", err)
	}

	// For FeatureFlags, remove _updatedAt and _createdAt fields recursively
	if profileType == ProfileTypeFeatureFlags {
		data = RemoveTimestampFieldsRecursive(data)
	}

	// Re-marshal with indentation for consistent formatting
	normalized, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to format JSON: %w", err)
	}

	return string(normalized), nil
}

// NormalizeYAML normalizes YAML content by parsing and re-formatting.
//
// Parameters:
//   - content: YAML string to normalize
//
// Returns:
//   - string: Normalized YAML with consistent formatting
//   - error: Any error during parsing or formatting
func NormalizeYAML(content string) (string, error) {
	var data any
	if err := yaml.Unmarshal([]byte(content), &data); err != nil {
		return "", fmt.Errorf("invalid YAML: %w", err)
	}

	// Re-marshal with consistent formatting
	normalized, err := yaml.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("failed to format YAML: %w", err)
	}

	return string(normalized), nil
}

// NormalizeText normalizes text content by ensuring consistent line endings.
//
// Parameters:
//   - content: Text string to normalize
//
// Returns:
//   - string: Normalized text with LF line endings and single trailing newline
func NormalizeText(content string) string {
	// Convert CRLF to LF
	content = strings.ReplaceAll(content, "\r\n", "\n")
	// Ensure single trailing newline
	content = strings.TrimRight(content, "\n") + "\n"
	return content
}

// RemoveTimestampFieldsRecursive recursively removes _updatedAt and _createdAt from all maps in the object.
// This is used when comparing FeatureFlags configurations to ignore auto-generated timestamps.
//
// Parameters:
//   - obj: Any object (map, slice, or primitive value)
//
// Returns:
//   - any: Object with timestamp fields removed
func RemoveTimestampFieldsRecursive(obj any) any {
	switch v := obj.(type) {
	case map[string]any:
		// Remove timestamp fields from this map
		delete(v, "_updatedAt")
		delete(v, "_createdAt")
		// Recursively process all values in the map
		for key, value := range v {
			v[key] = RemoveTimestampFieldsRecursive(value)
		}
		return v
	case []any:
		// Recursively process all elements in the array
		for i, value := range v {
			v[i] = RemoveTimestampFieldsRecursive(value)
		}
		return v
	default:
		// Return primitive values as-is
		return v
	}
}
