package config

import (
	"errors"
	"strings"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

func TestValidateAgainstJSONSchema(t *testing.T) {
	t.Parallel()

	const objectSchema = `{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"type": "object",
		"properties": {
			"name": {"type": "string"},
			"age": {"type": "integer", "minimum": 0}
		},
		"required": ["name"],
		"additionalProperties": false
	}`

	tests := []struct {
		name          string
		data          string
		schema        string
		forceDraft    *jsonschema.Draft
		wantSchemaErr bool   // expect *SchemaValidationError
		wantErr       bool   // expect any error
		wantContains  string // substring expected in the error message
	}{
		{
			name:    "valid data",
			data:    `{"name": "alice", "age": 30}`,
			schema:  objectSchema,
			wantErr: false,
		},
		{
			name:          "type violation",
			data:          `{"name": "alice", "age": "thirty"}`,
			schema:        objectSchema,
			wantSchemaErr: true,
			wantErr:       true,
			wantContains:  "/age",
		},
		{
			name:          "missing required",
			data:          `{"age": 30}`,
			schema:        objectSchema,
			wantSchemaErr: true,
			wantErr:       true,
			wantContains:  "name",
		},
		{
			name:          "additional property rejected",
			data:          `{"name": "alice", "extra": true}`,
			schema:        objectSchema,
			wantSchemaErr: true,
			wantErr:       true,
		},
		{
			name:       "draft-4 schema without $schema using default draft",
			data:       `{"name": "bob"}`,
			schema:     `{"type": "object", "properties": {"name": {"type": "string"}}, "required": ["name"]}`,
			forceDraft: jsonschema.Draft4,
			wantErr:    false,
		},
		{
			// boolean exclusiveMinimum is draft-4 semantics; with forceDraft the
			// document's draft-07 $schema is ignored and draft-4 rules apply, so
			// value 5 fails minimum:5 + exclusiveMinimum:true.
			name:          "force draft-4 overrides document $schema",
			data:          `{"n": 5}`,
			schema:        `{"$schema": "http://json-schema.org/draft-07/schema#", "type": "object", "properties": {"n": {"minimum": 5, "exclusiveMinimum": true}}}`,
			forceDraft:    jsonschema.Draft4,
			wantSchemaErr: true,
			wantErr:       true,
		},
		{
			name:    "invalid schema json",
			data:    `{"name": "alice"}`,
			schema:  `{not json}`,
			wantErr: true,
		},
		{
			name:    "invalid instance json",
			data:    `{not json}`,
			schema:  objectSchema,
			wantErr: true,
		},
		{
			name:    "large integer preserved",
			data:    `{"name": "alice", "age": 9007199254740993}`,
			schema:  objectSchema,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := validateAgainstJSONSchema([]byte(tt.data), []byte(tt.schema), tt.forceDraft, "")

			if tt.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
			if err == nil {
				return
			}

			var sve *SchemaValidationError
			isSchemaErr := errors.As(err, &sve)
			if tt.wantSchemaErr && !isSchemaErr {
				t.Fatalf("expected *SchemaValidationError, got %T: %v", err, err)
			}
			if !tt.wantSchemaErr && isSchemaErr {
				t.Fatalf("did not expect *SchemaValidationError, got: %v", err)
			}
			if tt.wantContains != "" && !strings.Contains(err.Error(), tt.wantContains) {
				t.Fatalf("error %q does not contain %q", err.Error(), tt.wantContains)
			}
		})
	}
}
