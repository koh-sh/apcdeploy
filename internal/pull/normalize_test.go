package pull

import (
	"strings"
	"testing"

	"github.com/koh-sh/apcdeploy/internal/config"
)

func TestNormalizeContent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		content     string
		ext         string
		profileType string
		wantErr     bool
	}{
		{
			name:        "JSON with Freeform profile",
			content:     `{"key": "value"}`,
			ext:         ".json",
			profileType: config.ProfileTypeFreeform,
			wantErr:     false,
		},
		{
			name:        "JSON with FeatureFlags profile",
			content:     `{"key": "value", "_updatedAt": "2024-01-01"}`,
			ext:         ".json",
			profileType: config.ProfileTypeFeatureFlags,
			wantErr:     false,
		},
		{
			name:        "YAML content",
			content:     "key: value\n",
			ext:         ".yaml",
			profileType: config.ProfileTypeFreeform,
			wantErr:     false,
		},
		{
			name:        "YML content",
			content:     "key: value\n",
			ext:         ".yml",
			profileType: config.ProfileTypeFreeform,
			wantErr:     false,
		},
		{
			name:        "Text content",
			content:     "some text\r\n",
			ext:         ".txt",
			profileType: config.ProfileTypeFreeform,
			wantErr:     false,
		},
		{
			name:        "Invalid JSON",
			content:     `{invalid}`,
			ext:         ".json",
			profileType: config.ProfileTypeFreeform,
			wantErr:     true,
		},
		{
			name:        "Invalid YAML",
			content:     "key: [invalid",
			ext:         ".yaml",
			profileType: config.ProfileTypeFreeform,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := normalizeContent(tt.content, tt.ext, tt.profileType)
			if (err != nil) != tt.wantErr {
				t.Errorf("normalizeContent() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && result == "" {
				t.Error("normalizeContent() returned empty string for valid input")
			}
		})
	}
}

func TestHasChanges(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		localContent  string
		remoteContent string
		fileName      string
		profileType   string
		wantChanges   bool
		wantErr       bool
	}{
		{
			name:          "Identical JSON content",
			localContent:  `{"key": "value"}`,
			remoteContent: `{"key": "value"}`,
			fileName:      "data.json",
			profileType:   config.ProfileTypeFreeform,
			wantChanges:   false,
			wantErr:       false,
		},
		{
			name:          "Different JSON content",
			localContent:  `{"key": "value1"}`,
			remoteContent: `{"key": "value2"}`,
			fileName:      "data.json",
			profileType:   config.ProfileTypeFreeform,
			wantChanges:   true,
			wantErr:       false,
		},
		{
			name:          "FeatureFlags with only timestamp differences",
			localContent:  `{"key": "value"}`,
			remoteContent: `{"key": "value", "_updatedAt": "2024-01-01"}`,
			fileName:      "data.json",
			profileType:   config.ProfileTypeFeatureFlags,
			wantChanges:   false,
			wantErr:       false,
		},
		{
			name:          "FeatureFlags with real differences",
			localContent:  `{"key": "value1"}`,
			remoteContent: `{"key": "value2", "_updatedAt": "2024-01-01"}`,
			fileName:      "data.json",
			profileType:   config.ProfileTypeFeatureFlags,
			wantChanges:   true,
			wantErr:       false,
		},
		{
			name:          "Identical YAML content",
			localContent:  "key: value\n",
			remoteContent: "key: value\n",
			fileName:      "data.yaml",
			profileType:   config.ProfileTypeFreeform,
			wantChanges:   false,
			wantErr:       false,
		},
		{
			name:          "Different YAML content",
			localContent:  "key: value1\n",
			remoteContent: "key: value2\n",
			fileName:      "data.yaml",
			profileType:   config.ProfileTypeFreeform,
			wantChanges:   true,
			wantErr:       false,
		},
		{
			name:          "Identical text content",
			localContent:  "some text",
			remoteContent: "some text",
			fileName:      "data.txt",
			profileType:   config.ProfileTypeFreeform,
			wantChanges:   false,
			wantErr:       false,
		},
		{
			name:          "Different text content",
			localContent:  "some text",
			remoteContent: "different text",
			fileName:      "data.txt",
			profileType:   config.ProfileTypeFreeform,
			wantChanges:   true,
			wantErr:       false,
		},
		{
			name:          "Invalid local JSON",
			localContent:  `{invalid}`,
			remoteContent: `{"key": "value"}`,
			fileName:      "data.json",
			profileType:   config.ProfileTypeFreeform,
			wantChanges:   false,
			wantErr:       true,
		},
		{
			name:          "Invalid remote JSON",
			localContent:  `{"key": "value"}`,
			remoteContent: `{invalid}`,
			fileName:      "data.json",
			profileType:   config.ProfileTypeFreeform,
			wantChanges:   false,
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			executor := &Executor{}
			hasChanges, err := executor.hasChanges(tt.localContent, tt.remoteContent, tt.fileName, tt.profileType)

			if (err != nil) != tt.wantErr {
				t.Errorf("hasChanges() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && hasChanges != tt.wantChanges {
				t.Errorf("hasChanges() = %v, want %v", hasChanges, tt.wantChanges)
			}
		})
	}
}

func TestNormalizeContentJSON(t *testing.T) {
	t.Parallel()

	// Test that JSON is properly normalized (formatted)
	input := `{"b":"2","a":"1"}`
	result, err := normalizeContent(input, ".json", config.ProfileTypeFreeform)
	if err != nil {
		t.Fatalf("normalizeContent() error = %v", err)
	}

	// Should be formatted with sorted keys and indentation
	if !strings.Contains(result, "  ") {
		t.Error("expected JSON to be indented")
	}
}

func TestNormalizeContentFeatureFlags(t *testing.T) {
	t.Parallel()

	// Test that FeatureFlags metadata is removed
	input := `{"key":"value","_updatedAt":"2024-01-01","_createdAt":"2024-01-01"}`
	result, err := normalizeContent(input, ".json", config.ProfileTypeFeatureFlags)
	if err != nil {
		t.Fatalf("normalizeContent() error = %v", err)
	}

	if strings.Contains(result, "_updatedAt") {
		t.Error("expected _updatedAt to be removed from FeatureFlags JSON")
	}
	if strings.Contains(result, "_createdAt") {
		t.Error("expected _createdAt to be removed from FeatureFlags JSON")
	}
	if !strings.Contains(result, "key") {
		t.Error("expected 'key' field to be preserved")
	}
}
