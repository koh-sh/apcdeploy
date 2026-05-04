package batch

import (
	"testing"

	"github.com/koh-sh/apcdeploy/internal/config"
)

func TestTarget_FieldsRoundTrip(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Region:               "us-east-1",
		Application:          "my-app",
		ConfigurationProfile: "my-profile",
		Environment:          "production",
	}
	tgt := &Target{
		Path:       "./envs/prod.yml",
		Config:     cfg,
		Identifier: config.Identifier("", cfg),
	}

	if got, want := tgt.Identifier, "us-east-1/my-app/my-profile/production"; got != want {
		t.Errorf("Identifier = %q, want %q", got, want)
	}
	if tgt.Config != cfg {
		t.Errorf("Config pointer not preserved")
	}
	if tgt.Path != "./envs/prod.yml" {
		t.Errorf("Path = %q, want preserved verbatim", tgt.Path)
	}
}
