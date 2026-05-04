package edit

import (
	"testing"

	awsInternal "github.com/koh-sh/apcdeploy/internal/aws"
)

func TestResolvedTargetsIdentifier(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		region string
		t      *resolvedTargets
		want   string
	}{
		{
			name:   "uses profile name from resolver",
			region: "us-east-1",
			t: &resolvedTargets{
				AppName: "app",
				EnvName: "prod",
				Profile: &awsInternal.ProfileInfo{Name: "feature-flags"},
			},
			want: "us-east-1/app/feature-flags/prod",
		},
		{
			name:   "different region",
			region: "eu-west-1",
			t: &resolvedTargets{
				AppName: "my-app",
				EnvName: "dev",
				Profile: &awsInternal.ProfileInfo{Name: "config"},
			},
			want: "eu-west-1/my-app/config/dev",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.t.Identifier(tt.region); got != tt.want {
				t.Errorf("Identifier(%q) = %q, want %q", tt.region, got, tt.want)
			}
		})
	}
}
