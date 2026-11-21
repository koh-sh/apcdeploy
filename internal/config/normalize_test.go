package config

import (
	"fmt"
	"reflect"
	"strings"
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
			// Integers > 2^53 cannot be represented exactly as float64.
			// NormalizeJSON must preserve them without precision loss so that
			// comparison/diff paths agree with formatJSON's written output.
			name:        "large integer preserves precision",
			content:     `{"id":9007199254740993}`,
			profileType: ProfileTypeFreeform,
			want:        "{\n  \"id\": 9007199254740993\n}",
			wantErr:     false,
		},
		{
			name:        "multiple large integers preserve precision",
			content:     `{"a":9007199254740993,"b":9007199254740994}`,
			profileType: ProfileTypeFreeform,
			want:        "{\n  \"a\": 9007199254740993,\n  \"b\": 9007199254740994\n}",
			wantErr:     false,
		},
		{
			name:        "invalid JSON",
			content:     `{invalid}`,
			profileType: ProfileTypeFreeform,
			want:        "",
			wantErr:     true,
		},
		{
			// Trailing data after the first value must be rejected so that
			// diff/pull agree with run/validate (json.Unmarshal rejects it).
			name:        "trailing garbage after value",
			content:     `{"a":1} garbage`,
			profileType: ProfileTypeFreeform,
			want:        "",
			wantErr:     true,
		},
		{
			name:        "two concatenated documents",
			content:     `{"a":1}{"b":2}`,
			profileType: ProfileTypeFreeform,
			want:        "",
			wantErr:     true,
		},
		{
			name:        "trailing whitespace and newline allowed",
			content:     "{\"a\":1}\n \t\n",
			profileType: ProfileTypeFreeform,
			want:        "{\n  \"a\": 1\n}",
			wantErr:     false,
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
		{
			name:    "empty document",
			content: "",
			want:    "null\n",
			wantErr: false,
		},
		{
			// Every document is kept: a second document used to be dropped,
			// so differences in it were reported as "no changes".
			name:    "multiple documents are all normalized",
			content: "b: 2\na: 1\n---\nd: 4\nc: 3\n",
			want:    "a: 1\nb: 2\n---\nc: 3\nd: 4\n",
			wantErr: false,
		},
		{
			// goccy/go-yaml decodes every mapping into map[string]any, so
			// non-string keys come back as quoted strings.
			name:    "non-string keys are stringified",
			content: "2: b\n1: a\n",
			want:    "\"1\": a\n\"2\": b\n",
			wantErr: false,
		},
		{
			name:    "alias expansion beyond the size limit is rejected",
			content: yamlAliasBomb(9),
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

// yamlAliasBomb builds a "billion laughs" document: each level is a list of
// nine aliases to the previous level, so a few hundred bytes expand to 9^levels
// scalars once the aliases are resolved.
func yamlAliasBomb(levels int) string {
	var b strings.Builder
	b.WriteString("a0: &a0 [x, x, x, x, x, x, x, x, x]\n")
	for i := 1; i <= levels; i++ {
		fmt.Fprintf(&b, "a%d: &a%d [", i, i)
		for j := range 9 {
			if j > 0 {
				b.WriteString(", ")
			}
			fmt.Fprintf(&b, "*a%d", i-1)
		}
		b.WriteString("]\n")
	}
	return b.String()
}

func TestExpandedYAMLSizeExceeds(t *testing.T) {
	t.Parallel()

	ones := make([]any, 100)
	for i := range ones {
		ones[i] = uint64(1)
	}
	shared := []any{"xxxxxxxxxx"}
	aliased := make([]any, 100)
	for i := range aliased {
		aliased[i] = shared // what an alias decodes to: the same value again
	}

	tests := []struct {
		name  string
		value any
		limit int
		want  bool
	}{
		// 100 one-digit items marshal to about 300 bytes; a flat per-scalar
		// estimate used to reject large alias-free documents.
		{name: "short scalars are estimated by their encoded length", value: ones, limit: 500, want: false},
		{name: "booleans and nulls", value: []any{true, false, nil}, limit: 30, want: false},
		{name: "aliases are counted once per reference", value: aliased, limit: 500, want: true},
		{name: "map keys and values", value: map[string]any{"key": "value"}, limit: 5, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := expandedYAMLSizeExceeds(tt.value, tt.limit); got != tt.want {
				t.Errorf("expandedYAMLSizeExceeds() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNormalizeYAML_ErrorOmitsSource(t *testing.T) {
	t.Parallel()

	// The content may be the remote deployed version, so a parse error must
	// not echo its lines into the terminal or a CI log.
	_, err := NormalizeYAML("a: 1\nb: [1, 2\nsecret: hunter2\n")
	if err == nil {
		t.Fatal("expected a parse error, got nil")
	}
	if strings.Contains(err.Error(), "hunter2") {
		t.Errorf("parse error includes source content: %q", err.Error())
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
			// 9007199254740993 rounds to 9007199254740992 as float64, so a
			// float64-based normalizer would treat this genuine change as a
			// no-op. UseNumber must keep the two integers distinct.
			name:        "large integer change is detected",
			before:      []byte(`{"id":9007199254740992}`),
			after:       []byte(`{"id":9007199254740993}`),
			ext:         ".json",
			profileType: ProfileTypeFreeform,
			wantChanged: true,
		},
		{
			name:        "identical large integer is not a change",
			before:      []byte(`{"id":9007199254740993}`),
			after:       []byte(`{"id": 9007199254740993}`),
			ext:         ".json",
			profileType: ProfileTypeFreeform,
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
