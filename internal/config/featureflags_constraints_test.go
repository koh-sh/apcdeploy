package config

import (
	"errors"
	"strings"
	"testing"
)

func TestValidateFeatureFlagsConstraints(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		data          string
		wantSchemaErr bool
		wantErr       bool
		wantContains  string
	}{
		{
			name: "value satisfies string enum",
			data: `{
				"version": "1",
				"flags": {"f": {"attributes": {"color": {"constraints": {"type": "string", "enum": ["red", "blue"]}}}}},
				"values": {"f": {"enabled": true, "color": "red"}}
			}`,
		},
		{
			name: "value violates string enum",
			data: `{
				"version": "1",
				"flags": {"f": {"attributes": {"color": {"constraints": {"type": "string", "enum": ["red", "blue"]}}}}},
				"values": {"f": {"enabled": true, "color": "green"}}
			}`,
			wantSchemaErr: true,
			wantErr:       true,
			wantContains:  "values/f/color",
		},
		{
			name: "value satisfies string pattern",
			data: `{
				"version": "1",
				"flags": {"f": {"attributes": {"code": {"constraints": {"type": "string", "pattern": "^[a-z]+$"}}}}},
				"values": {"f": {"enabled": true, "code": "abc"}}
			}`,
		},
		{
			name: "value violates string pattern",
			data: `{
				"version": "1",
				"flags": {"f": {"attributes": {"code": {"constraints": {"type": "string", "pattern": "^[a-z]+$"}}}}},
				"values": {"f": {"enabled": true, "code": "ABC123"}}
			}`,
			wantSchemaErr: true,
			wantErr:       true,
			wantContains:  "values/f/code",
		},
		{
			name: "number below minimum",
			data: `{
				"version": "1",
				"flags": {"f": {"attributes": {"rate": {"constraints": {"type": "number", "minimum": 1, "maximum": 100}}}}},
				"values": {"f": {"enabled": true, "rate": 0}}
			}`,
			wantSchemaErr: true,
			wantErr:       true,
			wantContains:  "values/f/rate",
		},
		{
			name: "number within range",
			data: `{
				"version": "1",
				"flags": {"f": {"attributes": {"rate": {"constraints": {"type": "number", "minimum": 1, "maximum": 100}}}}},
				"values": {"f": {"enabled": true, "rate": 50}}
			}`,
		},
		{
			name: "required attribute missing",
			data: `{
				"version": "1",
				"flags": {"f": {"attributes": {"color": {"constraints": {"type": "string", "required": true}}}}},
				"values": {"f": {"enabled": true}}
			}`,
			wantSchemaErr: true,
			wantErr:       true,
			wantContains:  "values/f",
		},
		{
			name: "wrong type",
			data: `{
				"version": "1",
				"flags": {"f": {"attributes": {"flag2": {"constraints": {"type": "boolean"}}}}},
				"values": {"f": {"enabled": true, "flag2": "notbool"}}
			}`,
			wantSchemaErr: true,
			wantErr:       true,
			wantContains:  "values/f/flag2",
		},
		{
			name: "array of numbers violates element constraint",
			data: `{
				"version": "1",
				"flags": {"f": {"attributes": {"nums": {"constraints": {"type": "array", "elements": {"type": "number", "minimum": 0}}}}}},
				"values": {"f": {"enabled": true, "nums": [1, -5]}}
			}`,
			wantSchemaErr: true,
			wantErr:       true,
			wantContains:  "values/f/nums",
		},
		{
			name: "multi-variant attributeValues valid",
			data: `{
				"version": "1",
				"flags": {"f": {"attributes": {"rate": {"constraints": {"type": "number", "minimum": 0, "maximum": 100}}}}},
				"values": {"f": {"_variants": [
					{"name": "default", "enabled": true, "attributeValues": {"rate": 50}},
					{"name": "beta", "enabled": true, "attributeValues": {"rate": 80}}
				]}}
			}`,
		},
		{
			name: "multi-variant attributeValues violation",
			data: `{
				"version": "1",
				"flags": {"f": {"attributes": {"rate": {"constraints": {"type": "number", "minimum": 0, "maximum": 100}}}}},
				"values": {"f": {"_variants": [
					{"name": "default", "enabled": true, "attributeValues": {"rate": 50}},
					{"name": "beta", "enabled": true, "attributeValues": {"rate": 999}}
				]}}
			}`,
			wantSchemaErr: true,
			wantErr:       true,
			wantContains:  "values/f/_variants/1/attributeValues/rate",
		},
		{
			name: "multi-variant enum violation in first variant",
			data: `{
				"version": "1",
				"flags": {"f": {"attributes": {"color": {"constraints": {"type": "string", "enum": ["red", "blue"]}}}}},
				"values": {"f": {"_variants": [
					{"name": "default", "enabled": true, "attributeValues": {"color": "green"}}
				]}}
			}`,
			wantSchemaErr: true,
			wantErr:       true,
			wantContains:  "values/f/_variants/0/attributeValues/color",
		},
		{
			name: "multi-variant missing required attribute",
			data: `{
				"version": "1",
				"flags": {"f": {"attributes": {"color": {"constraints": {"type": "string", "required": true}}}}},
				"values": {"f": {"_variants": [
					{"name": "default", "enabled": true, "attributeValues": {"color": "red"}},
					{"name": "beta", "enabled": true}
				]}}
			}`,
			wantSchemaErr: true,
			wantErr:       true,
			wantContains:  "values/f/_variants/1/attributeValues",
		},
		{
			name: "array of numbers satisfies element constraint",
			data: `{
				"version": "1",
				"flags": {"f": {"attributes": {"nums": {"constraints": {"type": "array", "elements": {"type": "number", "minimum": 0}}}}}},
				"values": {"f": {"enabled": true, "nums": [0, 50, 100]}}
			}`,
		},
		{
			name:    "invalid json",
			data:    `{not json}`,
			wantErr: true,
		},
		{
			name: "no flags section",
			data: `{"version": "1", "values": {"f": {"enabled": true}}}`,
		},
		{
			name: "value without matching flag is skipped",
			data: `{
				"version": "1",
				"flags": {"f": {"attributes": {"color": {"constraints": {"type": "string", "enum": ["red"]}}}}},
				"values": {"other": {"enabled": true}}
			}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := ValidateFeatureFlagsConstraints([]byte(tt.data))
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
			if tt.wantSchemaErr && !errors.As(err, &sve) {
				t.Fatalf("expected *SchemaValidationError, got %T: %v", err, err)
			}
			if tt.wantContains != "" && !strings.Contains(err.Error(), tt.wantContains) {
				t.Fatalf("error %q does not contain %q", err.Error(), tt.wantContains)
			}
		})
	}
}
