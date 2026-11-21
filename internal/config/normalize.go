package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/goccy/go-yaml"
)

// NormalizeJSON normalizes JSON content by parsing and re-formatting with sorted keys.
// For FeatureFlags profile type, it removes _updatedAt and _createdAt fields recursively.
//
// UseNumber is used during decoding so that large integers (> 2^53) are preserved
// exactly instead of being converted to float64, which would cause precision loss.
// This keeps the comparison/diff path consistent with formatJSON's written output.
//
// Parameters:
//   - content: JSON string to normalize
//   - profileType: AWS AppConfig profile type (e.g., "AWS.AppConfig.FeatureFlags")
//
// Returns:
//   - string: Normalized JSON with consistent formatting
//   - error: Any error during parsing or formatting
func NormalizeJSON(content string, profileType string) (string, error) {
	data, err := decodeJSONUseNumber(strings.NewReader(content))
	if err != nil {
		return "", err
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

// decodeJSONUseNumber decodes a single JSON value from r with UseNumber enabled.
// Unlike a bare Decoder.Decode, it rejects any non-whitespace data after the
// first value (e.g. `{"a":1} garbage` or two concatenated documents), matching
// json.Unmarshal so normalization agrees with ValidateData.
func decodeJSONUseNumber(r io.Reader) (any, error) {
	var data any
	dec := json.NewDecoder(r)
	dec.UseNumber()
	if err := dec.Decode(&data); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return nil, errors.New("invalid JSON: unexpected data after top-level value")
	}
	return data, nil
}

// NormalizeYAML normalizes YAML content by parsing and re-formatting every
// document in it (joined with "---").
//
// The content may be the remote deployed version, so parse errors are
// reported without the source excerpt goccy/go-yaml adds by default, and a
// document whose aliases expand beyond MaxNormalizedYAMLSize is rejected
// before it is re-marshalled.
//
// Parameters:
//   - content: YAML string to normalize
//
// Returns:
//   - string: Normalized YAML with consistent formatting
//   - error: Any error during parsing or formatting
func NormalizeYAML(content string) (string, error) {
	docs, err := decodeYAMLDocuments([]byte(content))
	if err != nil {
		return "", fmt.Errorf("invalid YAML: %s", yaml.FormatError(err, false, false))
	}
	if len(docs) == 0 {
		// An empty input normalizes like a single null document.
		return marshalNormalizedYAML(nil)
	}
	normalized := make([]string, len(docs))
	for i, doc := range docs {
		if normalized[i], err = marshalNormalizedYAML(doc); err != nil {
			return "", err
		}
	}
	return strings.Join(normalized, "---\n"), nil
}

// decodeYAMLDocuments decodes every document in content. yaml.Unmarshal
// stops after the first one, which would leave later documents unchecked
// and uncompared.
func decodeYAMLDocuments(content []byte) ([]any, error) {
	dec := yaml.NewDecoder(bytes.NewReader(content))
	var docs []any
	for {
		var doc any
		err := dec.Decode(&doc)
		if errors.Is(err, io.EOF) {
			return docs, nil
		}
		if err != nil {
			return nil, err
		}
		docs = append(docs, doc)
	}
}

func marshalNormalizedYAML(data any) (string, error) {
	if expandedYAMLSizeExceeds(data, MaxNormalizedYAMLSize) {
		return "", fmt.Errorf("invalid YAML: document exceeds %d bytes once aliases are expanded", MaxNormalizedYAMLSize)
	}
	normalized, err := yaml.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("failed to format YAML: %w", err)
	}
	return string(normalized), nil
}

// expandedYAMLSizeExceeds approximates the marshalled size of a decoded YAML
// value and reports whether it exceeds limit. Aliases decode to shared
// values, so the walk visits each alias target once per reference, as
// Marshal would, and stops as soon as the limit is crossed.
func expandedYAMLSizeExceeds(v any, limit int) bool {
	size := 0
	var walk func(v any) bool
	walk = func(v any) bool {
		switch x := v.(type) {
		case map[string]any:
			for k, val := range x {
				size += len(k) + 2
				if size > limit || walk(val) {
					return true
				}
			}
		case []any:
			for _, val := range x {
				size += 2
				if size > limit || walk(val) {
					return true
				}
			}
		case string:
			size += len(x)
		default:
			size += 8
		}
		return size > limit
	}
	return walk(v)
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

// NormalizeByExtension dispatches to the appropriate normalizer based on a
// file extension (".json", ".yaml"/".yml", otherwise treated as text).
// The extension match is case-insensitive (".JSON" works the same as ".json").
func NormalizeByExtension(content, ext, profileType string) (string, error) {
	switch strings.ToLower(ext) {
	case ".json":
		return NormalizeJSON(content, profileType)
	case ".yaml", ".yml":
		return NormalizeYAML(content)
	default:
		return NormalizeText(content), nil
	}
}

// HasContentChanged reports whether two content blobs differ after
// normalization. It short-circuits when the raw bytes are already equal.
func HasContentChanged(before, after []byte, ext, profileType string) (bool, error) {
	if bytes.Equal(before, after) {
		return false, nil
	}
	normalizedBefore, err := NormalizeByExtension(string(before), ext, profileType)
	if err != nil {
		return false, fmt.Errorf("failed to normalize original content: %w", err)
	}
	normalizedAfter, err := NormalizeByExtension(string(after), ext, profileType)
	if err != nil {
		return false, fmt.Errorf("failed to normalize edited content: %w", err)
	}
	return normalizedBefore != normalizedAfter, nil
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
