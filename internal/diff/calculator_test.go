package diff

import (
	"strings"
	"testing"
)

func TestCalculate(t *testing.T) {
	tests := []struct {
		name          string
		remoteContent string
		localContent  string
		fileName      string
		wantChanges   bool
		wantErr       bool
	}{
		{
			name:          "no changes - JSON",
			remoteContent: `{"key": "value"}`,
			localContent:  `{"key": "value"}`,
			fileName:      "config.json",
			wantChanges:   false,
			wantErr:       false,
		},
		{
			name:          "changes detected - JSON",
			remoteContent: `{"key": "old"}`,
			localContent:  `{"key": "new"}`,
			fileName:      "config.json",
			wantChanges:   true,
			wantErr:       false,
		},
		{
			name:          "no changes - YAML",
			remoteContent: "key: value\n",
			localContent:  "key: value\n",
			fileName:      "config.yaml",
			wantChanges:   false,
			wantErr:       false,
		},
		{
			name:          "changes detected - YAML",
			remoteContent: "key: old\n",
			localContent:  "key: new\n",
			fileName:      "config.yml",
			wantChanges:   true,
			wantErr:       false,
		},
		{
			name:          "no changes - text file",
			remoteContent: "line1\nline2\n",
			localContent:  "line1\nline2\n",
			fileName:      "config.txt",
			wantChanges:   false,
			wantErr:       false,
		},
		{
			name:          "changes detected - text file",
			remoteContent: "line1\nline2\n",
			localContent:  "line1\nline3\n",
			fileName:      "config.txt",
			wantChanges:   true,
			wantErr:       false,
		},
		{
			name:          "invalid JSON - remote",
			remoteContent: `{invalid}`,
			localContent:  `{"key": "value"}`,
			fileName:      "config.json",
			wantChanges:   false,
			wantErr:       true,
		},
		{
			name:          "invalid JSON - local",
			remoteContent: `{"key": "value"}`,
			localContent:  `{invalid}`,
			fileName:      "config.json",
			wantChanges:   false,
			wantErr:       true,
		},
		{
			name:          "different formatting - same content - JSON",
			remoteContent: `{"key":"value","other":"data"}`,
			localContent:  `{"key": "value", "other": "data"}`,
			fileName:      "config.json",
			wantChanges:   false,
			wantErr:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := calculate(tt.remoteContent, tt.localContent, tt.fileName)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result.HasChanges != tt.wantChanges {
				t.Errorf("HasChanges = %v, want %v", result.HasChanges, tt.wantChanges)
			}

			if result.FileName != tt.fileName {
				t.Errorf("FileName = %v, want %v", result.FileName, tt.fileName)
			}

			if result.RemoteContent == "" {
				t.Error("RemoteContent should not be empty")
			}

			if result.LocalContent == "" {
				t.Error("LocalContent should not be empty")
			}

			if tt.wantChanges && result.UnifiedDiff == "" {
				t.Error("UnifiedDiff should not be empty when changes exist")
			}
		})
	}
}

func TestNormalizeJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "valid JSON",
			input:   `{"key": "value"}`,
			wantErr: false,
		},
		{
			name:    "invalid JSON",
			input:   `{invalid}`,
			wantErr: true,
		},
		{
			name:    "empty JSON object",
			input:   `{}`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := normalizeJSON(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result == "" {
				t.Error("result should not be empty")
			}

			// Verify it's valid JSON by normalizing again
			_, err = normalizeJSON(result)
			if err != nil {
				t.Errorf("normalized result is not valid JSON: %v", err)
			}
		})
	}
}

func TestNormalizeYAML(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "valid YAML",
			input:   "key: value\n",
			wantErr: false,
		},
		{
			name:    "invalid YAML",
			input:   "key: value: invalid\n",
			wantErr: true,
		},
		{
			name:    "empty YAML",
			input:   "",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := normalizeYAML(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Verify it's valid YAML by normalizing again
			_, err = normalizeYAML(result)
			if err != nil {
				t.Errorf("normalized result is not valid YAML: %v", err)
			}
		})
	}
}

func TestNormalizeText(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "CRLF to LF",
			input:    "line1\r\nline2\r\n",
			expected: "line1\nline2\n",
		},
		{
			name:     "already LF",
			input:    "line1\nline2\n",
			expected: "line1\nline2\n",
		},
		{
			name:     "multiple trailing newlines",
			input:    "line1\nline2\n\n\n",
			expected: "line1\nline2\n",
		},
		{
			name:     "no trailing newline",
			input:    "line1\nline2",
			expected: "line1\nline2\n",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeText(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeText() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestNormalizeContent(t *testing.T) {
	tests := []struct {
		name    string
		content string
		ext     string
		wantErr bool
	}{
		{
			name:    "JSON file",
			content: `{"key": "value"}`,
			ext:     ".json",
			wantErr: false,
		},
		{
			name:    "YAML file - .yaml",
			content: "key: value\n",
			ext:     ".yaml",
			wantErr: false,
		},
		{
			name:    "YAML file - .yml",
			content: "key: value\n",
			ext:     ".yml",
			wantErr: false,
		},
		{
			name:    "text file",
			content: "some text\n",
			ext:     ".txt",
			wantErr: false,
		},
		{
			name:    "unknown extension",
			content: "some content\n",
			ext:     ".xyz",
			wantErr: false,
		},
		{
			name:    "invalid JSON",
			content: `{invalid}`,
			ext:     ".json",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := normalizeContent(tt.content, tt.ext)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result == "" {
				t.Error("result should not be empty")
			}
		})
	}
}

func TestCalculateResult(t *testing.T) {
	// Test that Calculate returns proper Result structure
	remoteContent := `{"key": "old"}`
	localContent := `{"key": "new"}`
	fileName := "config.json"

	result, err := calculate(remoteContent, localContent, fileName)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify all fields are populated
	if result.RemoteContent == "" {
		t.Error("RemoteContent should be populated")
	}

	if result.LocalContent == "" {
		t.Error("LocalContent should be populated")
	}

	if result.UnifiedDiff == "" {
		t.Error("UnifiedDiff should be populated when there are changes")
	}

	if !result.HasChanges {
		t.Error("HasChanges should be true when contents differ")
	}

	if result.FileName != fileName {
		t.Errorf("FileName = %v, want %v", result.FileName, fileName)
	}

	// Verify the diff contains expected markers
	if !strings.Contains(result.UnifiedDiff, "old") || !strings.Contains(result.UnifiedDiff, "new") {
		t.Error("UnifiedDiff should contain both old and new values")
	}
}
