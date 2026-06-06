package config

import (
	"bytes"
	"errors"
	"fmt"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

// schemaResourceURL is the internal identifier used to register a schema with
// the jsonschema compiler. It is never dereferenced over the network.
const schemaResourceURL = "apcdeploy://schema.json"

// SchemaValidationError reports one or more JSON Schema constraint violations,
// each annotated with the location within the instance that failed.
type SchemaValidationError struct {
	Issues []string
}

func (e *SchemaValidationError) Error() string {
	if len(e.Issues) == 0 {
		return "schema validation failed"
	}
	return "schema validation failed:\n  - " + strings.Join(e.Issues, "\n  - ")
}

// blockedURLLoader rejects any external schema reference ($ref to file:// or
// http(s)://). The only schema we evaluate is the one registered in-memory via
// AddResource, so no external loading is ever legitimate; blocking it prevents
// a crafted schema from reading arbitrary local files.
type blockedURLLoader struct{}

func (blockedURLLoader) Load(url string) (any, error) {
	return nil, fmt.Errorf("external schema reference not allowed: %s", url)
}

// validateAgainstJSONSchema validates raw JSON instance data against the given
// JSON Schema document. forceDraft, when non-nil, pins the dialect to that draft
// regardless of any $schema declared in the document — AWS AppConfig only
// supports JSON Schema draft 4.X for Freeform validators, so we evaluate locally
// with the same dialect AWS would use. Pass nil to honor the schema's own
// $schema (used for the built-in FeatureFlags schema, which is draft-07).
//
// locationPrefix is prepended to each violation's instance location, so callers
// that validate a sub-document (e.g. one flag's value) can report the full path
// within the original data. Pass "" when validating the whole document.
//
// A *SchemaValidationError is returned when the data violates the schema. Errors
// in the schema or instance syntax are returned as plain wrapped errors.
func validateAgainstJSONSchema(data, schemaJSON []byte, forceDraft *jsonschema.Draft, locationPrefix string) error {
	schemaDoc, err := jsonschema.UnmarshalJSON(bytes.NewReader(schemaJSON))
	if err != nil {
		return fmt.Errorf("invalid JSON schema: %w", err)
	}
	instanceDoc, err := jsonschema.UnmarshalJSON(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("invalid JSON syntax: %w", err)
	}

	compiler := jsonschema.NewCompiler()
	compiler.UseLoader(blockedURLLoader{})
	if forceDraft != nil {
		// Drop any $schema so the forced draft actually takes effect: the v6
		// compiler otherwise honors an in-document $schema over DefaultDraft.
		if m, ok := schemaDoc.(map[string]any); ok {
			delete(m, "$schema")
		}
		compiler.DefaultDraft(forceDraft)
	}
	if err := compiler.AddResource(schemaResourceURL, schemaDoc); err != nil {
		return fmt.Errorf("invalid JSON schema: %w", err)
	}
	sch, err := compiler.Compile(schemaResourceURL)
	if err != nil {
		return fmt.Errorf("invalid JSON schema: %w", err)
	}

	if err := sch.Validate(instanceDoc); err != nil {
		var ve *jsonschema.ValidationError
		if errors.As(err, &ve) {
			return &SchemaValidationError{Issues: collectSchemaIssues(ve, locationPrefix)}
		}
		return err
	}
	return nil
}

// collectSchemaIssues flattens a jsonschema validation error tree into a list of
// human-readable strings, one per leaf violation. Each entry is formatted as
// "<location>: <message>", where location is locationPrefix joined with the
// instance location; when the combined location is empty (a violation at the
// document root with no prefix) only the message is returned.
//
// DetailedOutput is used rather than BasicOutput because the v6 BasicOutput
// collapses every leaf message to a generic "validation failed", whereas the
// detailed tree carries the specific reason (e.g. "missing property 'version'").
// Only leaf units (those without nested errors) are reported; intermediate
// aggregation nodes carry no message of their own.
func collectSchemaIssues(ve *jsonschema.ValidationError, locationPrefix string) []string {
	var issues []string
	var walk func(unit *jsonschema.OutputUnit)
	walk = func(unit *jsonschema.OutputUnit) {
		if len(unit.Errors) == 0 {
			if unit.Error == nil {
				return
			}
			loc := locationPrefix + unit.InstanceLocation
			msg := unit.Error.String()
			if loc == "" {
				issues = append(issues, msg)
			} else {
				issues = append(issues, loc+": "+msg)
			}
			return
		}
		for i := range unit.Errors {
			walk(&unit.Errors[i])
		}
	}
	walk(ve.DetailedOutput())
	return issues
}
