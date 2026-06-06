package config

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

// featureFlagsSchema is the built-in AWS.AppConfig.FeatureFlags JSON schema
// (draft-07), used to validate the structure of feature flag configuration data
// locally without contacting AWS.
//
// It is embedded verbatim from AWS's published type reference, including its
// quirks — e.g. `_variants` declares `maxLength` where `maxItems` would be the
// correct array keyword, so that 32-element bound is effectively inert. Kept
// as-is so local validation matches what AWS actually enforces.
//
//go:embed featureflags_schema.json
var featureFlagsSchema []byte

// ValidateFeatureFlagsStructure validates that data conforms to the built-in
// AWS.AppConfig.FeatureFlags schema. This is the structural ("layer A") check:
// it verifies the shape of flags/values/variants but not the per-attribute
// constraints declared inside the data (see ValidateFeatureFlagsConstraints).
func ValidateFeatureFlagsStructure(data []byte) error {
	return validateAgainstJSONSchema(data, featureFlagsSchema, nil, "")
}

// ValidateFeatureFlagsConstraints performs the "layer B" check: it derives a
// JSON Schema from each flag's per-attribute `constraints` and validates the
// corresponding entry under `values` against it. This mirrors how AWS AppConfig
// rejects attribute values that violate their declared constraints.
//
// Only the top-level flag values are checked; per-variant attributeValues are
// out of scope. Flags without attributes, or values without a matching flag
// definition, are skipped (structural problems are caught by layer A).
func ValidateFeatureFlagsConstraints(data []byte) error {
	var root map[string]any
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	if err := dec.Decode(&root); err != nil {
		return fmt.Errorf("invalid JSON syntax: %w", err)
	}

	flags, _ := root["flags"].(map[string]any)
	values, _ := root["values"].(map[string]any)
	if len(flags) == 0 || len(values) == 0 {
		return nil
	}

	var issues []string
	for _, flagKey := range sortedKeys(flags) {
		flagDef, ok := flags[flagKey].(map[string]any)
		if !ok {
			continue
		}
		attrs, ok := flagDef["attributes"].(map[string]any)
		if !ok {
			continue
		}
		flagVal, ok := values[flagKey].(map[string]any)
		if !ok {
			continue
		}

		schemaJSON, err := json.Marshal(buildFlagValueSchema(attrs))
		if err != nil {
			return fmt.Errorf("failed to build constraint schema for flag %q: %w", flagKey, err)
		}
		if err := collectFlagIssues(schemaJSON, flagKey, flagVal, &issues); err != nil {
			return err
		}
	}

	if len(issues) > 0 {
		return &SchemaValidationError{Issues: issues}
	}
	return nil
}

// collectFlagIssues validates one flag's value(s) against schemaJSON. Multi-variant
// flags carry their attribute values inside `_variants` (each variant has its own
// `attributeValues`) rather than at the top level; single-variant flags hold the
// value directly. Each variant's attributeValues is validated against the same
// constraint schema.
func collectFlagIssues(schemaJSON []byte, flagKey string, flagVal map[string]any, issues *[]string) error {
	variants, ok := flagVal["_variants"].([]any)
	if !ok || len(variants) == 0 {
		return collectValueIssues(schemaJSON, flagVal, "values/"+flagKey, issues)
	}
	for idx, v := range variants {
		variant, ok := v.(map[string]any)
		if !ok {
			continue
		}
		av, _ := variant["attributeValues"].(map[string]any)
		if av == nil {
			// Validate an empty object so missing required attributes are still reported.
			av = map[string]any{}
		}
		prefix := fmt.Sprintf("values/%s/_variants/%d/attributeValues", flagKey, idx)
		if err := collectValueIssues(schemaJSON, av, prefix, issues); err != nil {
			return err
		}
	}
	return nil
}

// collectValueIssues validates a single value object against the constraint
// schema and appends any violations to issues, prefixed with the value's
// location within the data.
func collectValueIssues(schemaJSON []byte, obj map[string]any, prefix string, issues *[]string) error {
	objJSON, err := json.Marshal(obj)
	if err != nil {
		return fmt.Errorf("failed to encode value at %s: %w", prefix, err)
	}
	if err := validateAgainstJSONSchema(objJSON, schemaJSON, jsonschema.Draft7, prefix); err != nil {
		var sve *SchemaValidationError
		if !errors.As(err, &sve) {
			return err
		}
		*issues = append(*issues, sve.Issues...)
	}
	return nil
}

// buildFlagValueSchema converts a flag's attribute definitions into a JSON
// Schema object that validates that flag's value entry.
func buildFlagValueSchema(attributes map[string]any) map[string]any {
	props := map[string]any{}
	var required []string
	for _, attrName := range sortedKeys(attributes) {
		attrDef, ok := attributes[attrName].(map[string]any)
		if !ok {
			continue
		}
		cons, ok := attrDef["constraints"].(map[string]any)
		if !ok {
			continue
		}
		props[attrName] = constraintToSchema(cons)
		if req, ok := cons["required"].(bool); ok && req {
			required = append(required, attrName)
		}
	}

	schema := map[string]any{
		"type":       "object",
		"properties": props,
	}
	if len(required) > 0 {
		schema["required"] = required
	}
	// additionalProperties is intentionally left open: value objects also carry
	// `enabled`, `_variants`, and timestamp metadata that are not attributes.
	return schema
}

// constraintAttributeKeywords are the AppConfig constraint keywords that map
// directly onto JSON Schema and are copied through verbatim. They mirror what
// AWS AppConfig actually enforces for a feature flag attribute, so restricting
// to this set keeps local validation faithful to AWS (and prevents arbitrary
// JSON Schema keywords in the data from altering how the derived schema behaves).
var constraintAttributeKeywords = map[string]bool{
	"type":    true,
	"enum":    true,
	"pattern": true,
	"minimum": true,
	"maximum": true,
}

// constraintToSchema maps a single AppConfig constraint object to an equivalent
// JSON Schema fragment. Recognized constraint keywords are copied through; only
// `required` (handled by the caller) and `elements` (array element constraints)
// need special handling. Unrecognized keys are dropped — AWS would not enforce
// them, so neither should the local check.
func constraintToSchema(cons map[string]any) map[string]any {
	schema := map[string]any{}
	for k, v := range cons {
		switch {
		case k == "elements":
			if elem, ok := v.(map[string]any); ok {
				schema["items"] = constraintToSchema(elem)
			}
		case constraintAttributeKeywords[k]:
			schema[k] = v
		}
	}
	return schema
}

// sortedKeys returns the keys of m in sorted order for deterministic iteration.
func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}
