package diff

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/koh-sh/apcdeploy/internal/config"
	"github.com/sergi/go-diff/diffmatchpatch"
)

// Result represents the result of a diff calculation
type Result struct {
	// RemoteContent is the deployed configuration content
	RemoteContent string
	// LocalContent is the local configuration content
	LocalContent string
	// UnifiedDiff is the unified diff output
	UnifiedDiff string
	// HasChanges indicates whether there are any differences
	HasChanges bool
	// FileName is the name of the local file being compared
	FileName string
}

// calculate computes the diff between remote and local configuration.
// For FeatureFlags profile type, it removes _updatedAt and _createdAt fields
// before comparing to avoid false positives from auto-generated timestamps.
//
// The function normalizes both contents based on file type (JSON/YAML/text)
// to ensure consistent formatting before comparison.
//
// Parameters:
//   - remoteContent: The deployed configuration content
//   - localContent: The local configuration content
//   - fileName: Name of the local file (used to determine file type)
//   - profileType: AWS AppConfig profile type (e.g., "AWS.AppConfig.FeatureFlags")
//
// Returns:
//   - *Result: Diff result containing normalized contents and unified diff
//   - error: Any error during normalization or diff calculation
func calculate(remoteContent, localContent, fileName, profileType string) (*Result, error) {
	// Normalize content based on file extension
	ext := strings.ToLower(filepath.Ext(fileName))

	normalizedRemote, err := normalizeContent(remoteContent, ext, profileType)
	if err != nil {
		return nil, fmt.Errorf("failed to normalize remote content: %w", err)
	}

	normalizedLocal, err := normalizeContent(localContent, ext, profileType)
	if err != nil {
		return nil, fmt.Errorf("failed to normalize local content: %w", err)
	}

	// Generate line-based unified diff
	dmp := diffmatchpatch.New()

	// Convert texts to line-based diffs
	lineText1, lineText2, lineArray := dmp.DiffLinesToChars(normalizedRemote, normalizedLocal)
	diffs := dmp.DiffMain(lineText1, lineText2, false)
	diffs = dmp.DiffCharsToLines(diffs, lineArray)

	// Check if there are any changes
	hasChanges := false
	for _, d := range diffs {
		if d.Type != diffmatchpatch.DiffEqual {
			hasChanges = true
			break
		}
	}

	// Generate unified diff format
	unifiedDiff := formatDiffs(diffs)

	return &Result{
		RemoteContent: normalizedRemote,
		LocalContent:  normalizedLocal,
		UnifiedDiff:   unifiedDiff,
		HasChanges:    hasChanges,
		FileName:      fileName,
	}, nil
}

// normalizeContent normalizes content based on file type
// For FeatureFlags profile type, it removes _updatedAt and _createdAt from JSON
func normalizeContent(content, ext, profileType string) (string, error) {
	switch ext {
	case ".json":
		return normalizeJSON(content, profileType)
	case ".yaml", ".yml":
		return normalizeYAML(content)
	default:
		// For text files, just ensure consistent line endings
		return normalizeText(content), nil
	}
}

// normalizeJSON normalizes JSON content by parsing and re-formatting with sorted keys
// For FeatureFlags profile type, it removes _updatedAt and _createdAt fields recursively
func normalizeJSON(content string, profileType string) (string, error) {
	var data any
	if err := json.Unmarshal([]byte(content), &data); err != nil {
		return "", fmt.Errorf("invalid JSON: %w", err)
	}

	// For FeatureFlags, remove _updatedAt and _createdAt fields recursively
	if profileType == config.ProfileTypeFeatureFlags {
		data = removeTimestampFieldsRecursive(data)
	}

	// Re-marshal with indentation for consistent formatting
	normalized, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to format JSON: %w", err)
	}

	return string(normalized), nil
}

// removeTimestampFieldsRecursive recursively removes _updatedAt and _createdAt from all maps in the object
func removeTimestampFieldsRecursive(obj any) any {
	switch v := obj.(type) {
	case map[string]any:
		// Remove timestamp fields from this map
		delete(v, "_updatedAt")
		delete(v, "_createdAt")
		// Recursively process all values in the map
		for key, value := range v {
			v[key] = removeTimestampFieldsRecursive(value)
		}
		return v
	case []any:
		// Recursively process all elements in the array
		for i, value := range v {
			v[i] = removeTimestampFieldsRecursive(value)
		}
		return v
	default:
		// Return primitive values as-is
		return v
	}
}

// normalizeYAML normalizes YAML content by parsing and re-formatting
func normalizeYAML(content string) (string, error) {
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

// normalizeText normalizes text content by ensuring consistent line endings
func normalizeText(content string) string {
	// Convert CRLF to LF
	content = strings.ReplaceAll(content, "\r\n", "\n")
	// Ensure single trailing newline
	content = strings.TrimRight(content, "\n") + "\n"
	return content
}

// formatDiffs converts line-based diffs to a simple diff format.
// It processes each diff chunk and formats lines with prefixes:
//   - "+" for added lines
//   - "-" for deleted lines
//   - " " for context lines (unchanged)
//
// Empty lines are skipped to produce a cleaner output.
//
// Parameters:
//   - diffs: Slice of diff chunks from go-diff library
//
// Returns:
//   - string: Formatted diff output
func formatDiffs(diffs []diffmatchpatch.Diff) string {
	var result strings.Builder

	for _, diff := range diffs {
		lines := strings.SplitSeq(diff.Text, "\n")
		for line := range lines {
			// Skip empty lines
			if line == "" {
				continue
			}

			switch diff.Type {
			case diffmatchpatch.DiffInsert:
				result.WriteString("+")
				result.WriteString(line)
				result.WriteString("\n")
			case diffmatchpatch.DiffDelete:
				result.WriteString("-")
				result.WriteString(line)
				result.WriteString("\n")
			case diffmatchpatch.DiffEqual:
				result.WriteString(" ")
				result.WriteString(line)
				result.WriteString("\n")
			}
		}
	}

	return result.String()
}
