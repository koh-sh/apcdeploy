package config

import (
	"strings"
	"testing"
)

func TestValidateData(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		data        []byte
		contentType string
		wantErr     string
	}{
		{name: "valid json", data: []byte(`{"a":1}`), contentType: ContentTypeJSON},
		{name: "invalid json", data: []byte(`{`), contentType: ContentTypeJSON, wantErr: "invalid JSON syntax"},
		{name: "valid yaml", data: []byte("a: 1"), contentType: ContentTypeYAML},
		{name: "invalid yaml", data: []byte("a: :\n  b"), contentType: ContentTypeYAML, wantErr: "invalid YAML syntax"},
		{name: "text ok", data: []byte("hello"), contentType: ContentTypeText},
		{name: "unsupported type", data: []byte("x"), contentType: "application/xml", wantErr: "unsupported content type"},
		{
			name:        "too large",
			data:        make([]byte, MaxConfigSize+1),
			contentType: ContentTypeText,
			wantErr:     "exceeds maximum allowed size",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateData(tt.data, tt.contentType)
			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("expected error containing %q, got %v", tt.wantErr, err)
			}
		})
	}
}
