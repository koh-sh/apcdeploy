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
func calculate(remoteContent, localContent, fileName string) (*Result, error) {
	// Normalize content based on file extension
	ext := strings.ToLower(filepath.Ext(fileName))

	normalizedRemote, err := normalizeContent(remoteContent, ext)
	if err != nil {
		return nil, fmt.Errorf("failed to normalize remote content: %w", err)
	}

	normalizedLocal, err := normalizeContent(localContent, ext)
	if err != nil {
		return nil, fmt.Errorf("failed to normalize local content: %w", err)
	}

	// Generate unified diff
	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(normalizedRemote, normalizedLocal, false)
	unifiedDiff := dmp.DiffPrettyText(diffs)

	// Check if there are any changes
	hasChanges := false
	for _, d := range diffs {
		if d.Type != diffmatchpatch.DiffEqual {
			hasChanges = true
			break
		}
	}

	return &Result{
		RemoteContent: normalizedRemote,
		LocalContent:  normalizedLocal,
		UnifiedDiff:   unifiedDiff,
		HasChanges:    hasChanges,
		FileName:      fileName,
	}, nil
}

// normalizeContent normalizes content based on file type
func normalizeContent(content, ext string) (string, error) {
	switch ext {
	case ".json":
		return normalizeJSON(content)
	case ".yaml", ".yml":
		return normalizeYAML(content)
	default:
		// For text files, just ensure consistent line endings
		return normalizeText(content), nil
	}
}

// normalizeJSON normalizes JSON content by parsing and re-formatting with sorted keys
func normalizeJSON(content string) (string, error) {
	var data any
	if err := json.Unmarshal([]byte(content), &data); err != nil {
		return "", fmt.Errorf("invalid JSON: %w", err)
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
