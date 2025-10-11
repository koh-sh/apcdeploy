package cli

import (
	"testing"
)

func TestGetReporter(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		silent      bool
		wantType    string
		description string
	}{
		{
			name:        "returns regular reporter when silent is false",
			silent:      false,
			wantType:    "*cli.Reporter",
			description: "Should return a regular Reporter when silent mode is disabled",
		},
		{
			name:        "returns silent reporter when silent is true",
			silent:      true,
			wantType:    "*cli.SilentReporter",
			description: "Should return a SilentReporter when silent mode is enabled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := GetReporter(tt.silent)

			// Check that it implements the interface
			_ = got

			// Check the concrete type
			switch tt.silent {
			case true:
				if _, ok := got.(*SilentReporter); !ok {
					t.Errorf("GetReporter(%v) = %T, want %s", tt.silent, got, tt.wantType)
				}
			case false:
				if _, ok := got.(*Reporter); !ok {
					t.Errorf("GetReporter(%v) = %T, want %s", tt.silent, got, tt.wantType)
				}
			}
		})
	}
}
