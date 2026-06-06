package config

import "testing"

func TestDetermineContentType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		profileType string
		dataPath    string
		want        string
	}{
		{"feature flags always json", ProfileTypeFeatureFlags, "flags.json", ContentTypeJSON},
		{"feature flags ignores extension", ProfileTypeFeatureFlags, "flags.yaml", ContentTypeJSON},
		{"freeform json", ProfileTypeFreeform, "config.json", ContentTypeJSON},
		{"freeform yaml", ProfileTypeFreeform, "config.yaml", ContentTypeYAML},
		{"freeform yml", ProfileTypeFreeform, "config.yml", ContentTypeYAML},
		{"freeform text", ProfileTypeFreeform, "config.txt", ContentTypeText},
		{"freeform unknown extension defaults to text", ProfileTypeFreeform, "config.conf", ContentTypeText},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := DetermineContentType(tt.profileType, tt.dataPath); got != tt.want {
				t.Errorf("DetermineContentType(%q, %q) = %q, want %q", tt.profileType, tt.dataPath, got, tt.want)
			}
		})
	}
}
