package config

import "testing"

func TestIdentifier(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		region string
		config Config
		want   string
	}{
		{
			name:   "all fields populated from config",
			region: "",
			config: Config{
				Application:          "my-app",
				ConfigurationProfile: "my-profile",
				Environment:          "production",
				Region:               "us-east-1",
			},
			want: "us-east-1/my-app/my-profile/production",
		},
		{
			name:   "region argument overrides empty config region",
			region: "eu-west-1",
			config: Config{
				Application:          "my-app",
				ConfigurationProfile: "my-profile",
				Environment:          "dev",
			},
			want: "eu-west-1/my-app/my-profile/dev",
		},
		{
			name:   "config region wins when both supplied",
			region: "eu-west-1",
			config: Config{
				Application:          "my-app",
				ConfigurationProfile: "my-profile",
				Environment:          "dev",
				Region:               "us-east-1",
			},
			want: "us-east-1/my-app/my-profile/dev",
		},
		{
			name:   "feature flag profile name preserved verbatim",
			region: "us-east-1",
			config: Config{
				Application:          "my-app",
				ConfigurationProfile: "feature-flags",
				Environment:          "staging",
			},
			want: "us-east-1/my-app/feature-flags/staging",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := Identifier(tt.region, &tt.config)
			if got != tt.want {
				t.Errorf("Identifier() = %q, want %q", got, tt.want)
			}
		})
	}
}
