package config

import (
	"encoding/json"
	"fmt"

	"github.com/goccy/go-yaml"
	"github.com/santhosh-tekuri/jsonschema/v6"
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

// ValidateConfigData runs the full local validation for configuration data:
// the size + syntax check from ValidateData, followed by schema validation
// appropriate to the profile type.
//
//   - FeatureFlags: structural check against the built-in schema (layer A) plus
//     per-attribute constraint checking against `values` (layer B). schema is
//     ignored.
//   - Freeform JSON: validated against schema (the profile's JSON_SCHEMA
//     validator) when one is supplied; skipped when schema is empty.
//   - Freeform YAML/text: syntax only — JSON Schema cannot apply, so schema is
//     ignored.
//
// AWS AppConfig draft 4.X is assumed for Freeform schemas that omit $schema.
func ValidateConfigData(data []byte, profileType, contentType string, schema []byte) error {
	if err := ValidateData(data, contentType); err != nil {
		return err
	}

	if profileType == ProfileTypeFeatureFlags {
		if err := ValidateFeatureFlagsStructure(data); err != nil {
			return err
		}
		return ValidateFeatureFlagsConstraints(data)
	}

	// Freeform
	if contentType == ContentTypeJSON && len(schema) > 0 {
		return validateAgainstJSONSchema(data, schema, jsonschema.Draft4, "")
	}
	return nil
}
