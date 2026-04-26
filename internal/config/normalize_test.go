package config

import (
	"reflect"
	"testing"
)

func TestNormalizeJSON(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		profileType string
		want        string
		wantErr     bool
	}{
		{
			name:        "simple JSON normalization",
			content:     `{"b":2,"a":1}`,
			profileType: ProfileTypeFreeform,
			want:        "{\n  \"a\": 1,\n  \"b\": 2\n}",
			wantErr:     false,
		},
		{
			name:        "FeatureFlags with timestamps",
			content:     `{"_updatedAt":"2024-01-01","_createdAt":"2024-01-01","flag":"value"}`,
			profileType: ProfileTypeFeatureFlags,
			want:        "{\n  \"flag\": \"value\"\n}",
			wantErr:     false,
		},
		{
			name:        "nested timestamps in FeatureFlags",
			content:     `{"flags":{"flag1":{"_updatedAt":"2024-01-01","value":true}}}`,
			profileType: ProfileTypeFeatureFlags,
			want:        "{\n  \"flags\": {\n    \"flag1\": {\n      \"value\": true\n    }\n  }\n}",
			wantErr:     false,
		},
		{
			name:        "invalid JSON",
			content:     `{invalid}`,
			profileType: ProfileTypeFreeform,
			want:        "",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeJSON(tt.content, tt.profileType)
			if (err != nil) != tt.wantErr {
				t.Errorf("NormalizeJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("NormalizeJSON() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNormalizeYAML(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
		wantErr bool
	}{
		{
			name:    "simple YAML normalization",
			content: "b: 2\na: 1\n",
			want:    "a: 1\nb: 2\n",
			wantErr: false,
		},
		{
			name:    "nested YAML",
			content: "parent:\n  b: 2\n  a: 1\n",
			want:    "parent:\n  a: 1\n  b: 2\n",
			wantErr: false,
		},
		{
			name:    "invalid YAML",
			content: ":\n  invalid",
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeYAML(tt.content)
			if (err != nil) != tt.wantErr {
				t.Errorf("NormalizeYAML() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("NormalizeYAML() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNormalizeText(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "CRLF to LF",
			content: "line1\r\nline2\r\n",
			want:    "line1\nline2\n",
		},
		{
			name:    "multiple trailing newlines",
			content: "text\n\n\n",
			want:    "text\n",
		},
		{
			name:    "no trailing newline",
			content: "text",
			want:    "text\n",
		},
		{
			name:    "already normalized",
			content: "line1\nline2\n",
			want:    "line1\nline2\n",
		},
		{
			name:    "mixed line endings",
			content: "line1\nline2\r\nline3",
			want:    "line1\nline2\nline3\n",
		},
		{
			name:    "empty string",
			content: "",
			want:    "\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeText(tt.content)
			if got != tt.want {
				t.Errorf("NormalizeText() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNormalizeByExtension(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		content     string
		ext         string
		profileType string
		wantErr     bool
	}{
		{name: "json", content: `{"key":"value"}`, ext: ".json", profileType: ProfileTypeFreeform},
		{name: "yaml", content: "key: value\n", ext: ".yaml", profileType: ProfileTypeFreeform},
		{name: "yml", content: "key: value\n", ext: ".yml", profileType: ProfileTypeFreeform},
		{name: "txt falls back to text normalizer", content: "plain text\ncontent", ext: ".txt", profileType: ProfileTypeFreeform},
		{name: "unknown extension falls back to text", content: "hello", ext: ".unknown", profileType: ProfileTypeFreeform},
		{name: "invalid json", content: `{invalid}`, ext: ".json", profileType: ProfileTypeFreeform, wantErr: true},
		{name: "invalid yaml", content: ":\ninvalid\n:", ext: ".yaml", profileType: ProfileTypeFreeform, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := NormalizeByExtension(tt.content, tt.ext, tt.profileType)
			if (err != nil) != tt.wantErr {
				t.Errorf("NormalizeByExtension() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got == "" {
				t.Error("NormalizeByExtension() returned empty string for valid input")
			}
		})
	}
}

func TestHasContentChanged(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		before      []byte
		after       []byte
		ext         string
		profileType string
		wantChanged bool
		wantErr     bool
	}{
		{
			name:        "identical bytes short-circuit",
			before:      []byte(`{"a":1}`),
			after:       []byte(`{"a":1}`),
			ext:         ".json",
			profileType: ProfileTypeFreeform,
		},
		{
			name:        "whitespace-only difference is not a change",
			before:      []byte(`{"a":1,"b":2}`),
			after:       []byte("{\n  \"a\": 1,\n  \"b\": 2\n}\n"),
			ext:         ".json",
			profileType: ProfileTypeFreeform,
		},
		{
			name:        "value change is detected",
			before:      []byte(`{"a":1}`),
			after:       []byte(`{"a":2}`),
			ext:         ".json",
			profileType: ProfileTypeFreeform,
			wantChanged: true,
		},
		{
			name:        "FeatureFlags metadata is ignored",
			before:      []byte(`{"flags":{"f":{"_updatedAt":"x","name":"f"}}}`),
			after:       []byte(`{"flags":{"f":{"name":"f"}}}`),
			ext:         ".json",
			profileType: ProfileTypeFeatureFlags,
		},
		{
			name:        "invalid before propagates error",
			before:      []byte(`{invalid`),
			after:       []byte(`{}`),
			ext:         ".json",
			profileType: ProfileTypeFreeform,
			wantErr:     true,
		},
		{
			name:        "invalid after propagates error",
			before:      []byte(`{}`),
			after:       []byte(`{invalid`),
			ext:         ".json",
			profileType: ProfileTypeFreeform,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := HasContentChanged(tt.before, tt.after, tt.ext, tt.profileType)
			if (err != nil) != tt.wantErr {
				t.Errorf("HasContentChanged() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.wantChanged {
				t.Errorf("HasContentChanged() = %v, want %v", got, tt.wantChanged)
			}
		})
	}
}

func TestRemoveTimestampFieldsRecursive(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		obj  any
		want any
	}{
		{
			name: "simple map with timestamps",
			obj: map[string]any{
				"_updatedAt": "2024-01-01",
				"_createdAt": "2024-01-01",
				"value":      "data",
			},
			want: map[string]any{
				"value": "data",
			},
		},
		{
			name: "nested map with timestamps",
			obj: map[string]any{
				"outer": map[string]any{
					"_updatedAt": "2024-01-01",
					"inner":      "value",
				},
			},
			want: map[string]any{
				"outer": map[string]any{
					"inner": "value",
				},
			},
		},
		{
			name: "array of maps with timestamps",
			obj: []any{
				map[string]any{
					"_updatedAt": "2024-01-01",
					"value":      1,
				},
				map[string]any{
					"_createdAt": "2024-01-01",
					"value":      2,
				},
			},
			want: []any{
				map[string]any{
					"value": 1,
				},
				map[string]any{
					"value": 2,
				},
			},
		},
		{
			name: "primitive string",
			obj:  "string",
			want: "string",
		},
		{
			name: "primitive number",
			obj:  42,
			want: 42,
		},
		{
			name: "primitive bool",
			obj:  true,
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := RemoveTimestampFieldsRecursive(tt.obj)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RemoveTimestampFieldsRecursive() = %v, want %v", got, tt.want)
			}
		})
	}
}
