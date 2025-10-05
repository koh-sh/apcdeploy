package diff

import (
	"fmt"
	"path/filepath"
	"strings"

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
		return config.NormalizeJSON(content, profileType)
	case ".yaml", ".yml":
		return config.NormalizeYAML(content)
	default:
		// For text files, just ensure consistent line endings
		return config.NormalizeText(content), nil
	}
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
