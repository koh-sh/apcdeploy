package diff

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/goccy/go-yaml"
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

// calculate computes the diff between remote and local configuration
// For FeatureFlags profile type, it removes _updatedAt and _createdAt before comparing
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
// For FeatureFlags profile type, it removes _updatedAt and _createdAt fields
func normalizeJSON(content string, profileType string) (string, error) {
	var data any
	if err := json.Unmarshal([]byte(content), &data); err != nil {
		return "", fmt.Errorf("invalid JSON: %w", err)
	}

	// For FeatureFlags, remove _updatedAt and _createdAt fields
	if profileType == "AWS.AppConfig.FeatureFlags" {
		if objMap, ok := data.(map[string]any); ok {
			delete(objMap, "_updatedAt")
			delete(objMap, "_createdAt")
		}
	}

	// Re-marshal with indentation for consistent formatting
	normalized, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to format JSON: %w", err)
	}

	return string(normalized), nil
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

// formatDiffs converts line-based diffs to simple diff format
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
