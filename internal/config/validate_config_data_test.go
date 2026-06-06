package config

import (
	"errors"
	"testing"
)

func TestValidateConfigData(t *testing.T) {
	t.Parallel()

	const freeformSchema = `{
		"$schema": "http://json-schema.org/draft-04/schema#",
		"type": "object",
		"properties": {"port": {"type": "integer", "minimum": 1}},
		"required": ["port"]
	}`

	tests := []struct {
		name        string
		data        string
		profileType string
		contentType string
		schema      string
		wantErr     bool
	}{
		{
			name:        "freeform json valid against schema",
			data:        `{"port": 8080}`,
			profileType: ProfileTypeFreeform,
			contentType: ContentTypeJSON,
			schema:      freeformSchema,
		},
		{
			name:        "freeform json violates schema",
			data:        `{"port": 0}`,
			profileType: ProfileTypeFreeform,
			contentType: ContentTypeJSON,
			schema:      freeformSchema,
			wantErr:     true,
		},
		{
			name:        "freeform json missing required violates schema",
			data:        `{}`,
			profileType: ProfileTypeFreeform,
			contentType: ContentTypeJSON,
			schema:      freeformSchema,
			wantErr:     true,
		},
		{
			name:        "freeform json without schema only syntax",
			data:        `{"anything": true}`,
			profileType: ProfileTypeFreeform,
			contentType: ContentTypeJSON,
			schema:      "",
		},
		{
			name:        "freeform json invalid syntax",
			data:        `{bad}`,
			profileType: ProfileTypeFreeform,
			contentType: ContentTypeJSON,
			schema:      "",
			wantErr:     true,
		},
		{
			name:        "freeform yaml schema ignored",
			data:        "key: value\n",
			profileType: ProfileTypeFreeform,
			contentType: ContentTypeYAML,
			schema:      freeformSchema,
		},
		{
			name:        "freeform yaml invalid syntax",
			data:        "key: : :\n  - bad\n: nope",
			profileType: ProfileTypeFreeform,
			contentType: ContentTypeYAML,
			wantErr:     true,
		},
		{
			name:        "freeform text always passes",
			data:        "anything goes",
			profileType: ProfileTypeFreeform,
			contentType: ContentTypeText,
		},
		{
			name:        "featureflags valid ignores schema arg",
			data:        `{"version": "1", "flags": {"f": {"attributes": {"c": {"constraints": {"type": "string", "enum": ["a"]}}}}}, "values": {"f": {"enabled": true, "c": "a"}}}`,
			profileType: ProfileTypeFeatureFlags,
			contentType: ContentTypeJSON,
		},
		{
			name:        "featureflags constraint violation",
			data:        `{"version": "1", "flags": {"f": {"attributes": {"c": {"constraints": {"type": "string", "enum": ["a"]}}}}}, "values": {"f": {"enabled": true, "c": "z"}}}`,
			profileType: ProfileTypeFeatureFlags,
			contentType: ContentTypeJSON,
			wantErr:     true,
		},
		{
			name:        "featureflags structural violation",
			data:        `{"flags": {}}`,
			profileType: ProfileTypeFeatureFlags,
			contentType: ContentTypeJSON,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var schema []byte
			if tt.schema != "" {
				schema = []byte(tt.schema)
			}
			err := ValidateConfigData([]byte(tt.data), tt.profileType, tt.contentType, schema)
			if tt.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}
		})
	}

	t.Run("schema violation is SchemaValidationError", func(t *testing.T) {
		t.Parallel()
		err := ValidateConfigData([]byte(`{"port": 0}`), ProfileTypeFreeform, ContentTypeJSON, []byte(freeformSchema))
		var sve *SchemaValidationError
		if !errors.As(err, &sve) {
			t.Fatalf("expected *SchemaValidationError, got %T: %v", err, err)
		}
	})
}
