package config

import (
	"errors"
	"strings"
	"testing"
)

func TestValidateFeatureFlagsStructure(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		data          string
		wantSchemaErr bool
		wantErr       bool
		wantContains  string
	}{
		{
			name: "valid minimal",
			data: `{"version": "1"}`,
		},
		{
			name: "valid with flags and values",
			data: `{
				"version": "1",
				"flags": {
					"myflag": {
						"name": "My Flag",
						"attributes": {
							"color": {"constraints": {"type": "string", "enum": ["red", "blue"]}}
						}
					}
				},
				"values": {
					"myflag": {"enabled": true, "color": "red"}
				}
			}`,
		},
		{
			name:          "missing version",
			data:          `{"flags": {}}`,
			wantSchemaErr: true,
			wantErr:       true,
			wantContains:  "version",
		},
		{
			name:          "invalid version value",
			data:          `{"version": "2"}`,
			wantSchemaErr: true,
			wantErr:       true,
			wantContains:  "version",
		},
		{
			name:          "values not an object",
			data:          `{"version": "1", "values": []}`,
			wantSchemaErr: true,
			wantErr:       true,
			wantContains:  "values",
		},
		{
			name:          "unknown top-level property",
			data:          `{"version": "1", "bogus": true}`,
			wantSchemaErr: true,
			wantErr:       true,
			wantContains:  "bogus",
		},
		{
			name:    "invalid json",
			data:    `{not json}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := ValidateFeatureFlagsStructure([]byte(tt.data))
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
