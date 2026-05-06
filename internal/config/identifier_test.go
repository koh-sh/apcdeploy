package config

import "testing"

func TestIdentifier(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		config Config
		want   string
	}{
		{
			name: "all fields populated",
			config: Config{
				Application:          "my-app",
				ConfigurationProfile: "my-profile",
				Environment:          "production",
				Region:               "us-east-1",
			},
			want: "us-east-1/my-app/my-profile/production",
		},
		{
			name: "feature flag profile name preserved verbatim",
			config: Config{
				Application:          "my-app",
				ConfigurationProfile: "feature-flags",
				Environment:          "staging",
				Region:               "us-east-1",
			},
			want: "us-east-1/my-app/feature-flags/staging",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := Identifier(&tt.config)
			if got != tt.want {
				t.Errorf("Identifier() = %q, want %q", got, tt.want)
			}
		})
	}
}
