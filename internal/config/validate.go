package config

import (
	"encoding/json"
	"fmt"

	"github.com/goccy/go-yaml"
)

// ValidateData validates configuration data against the AppConfig size limit
// and the syntax rules for the given content type.
//
// Supported content types:
//   - ContentTypeJSON: rejects invalid JSON
//   - ContentTypeYAML: rejects invalid YAML
//   - ContentTypeText: no syntax check
//
// Any other content type returns an error.
func ValidateData(data []byte, contentType string) error {
	if len(data) > MaxConfigSize {
		return fmt.Errorf("configuration data size %d bytes exceeds maximum allowed size of %d bytes (2MB)", len(data), MaxConfigSize)
	}

	switch contentType {
	case ContentTypeJSON:
		var js any
		if err := json.Unmarshal(data, &js); err != nil {
			return fmt.Errorf("invalid JSON syntax: %w", err)
		}
	case ContentTypeYAML:
		var ym any
		if err := yaml.Unmarshal(data, &ym); err != nil {
			return fmt.Errorf("invalid YAML syntax: %w", err)
		}
	case ContentTypeText:
		// no syntax check
	default:
		return fmt.Errorf("unsupported content type: %s", contentType)
	}

	return nil
}
