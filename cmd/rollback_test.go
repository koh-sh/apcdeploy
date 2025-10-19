package cmd

import (
	"strings"
	"testing"
)

func TestRollbackCommand(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		check func(*testing.T)
	}{
		{
			name: "command is non-nil",
			check: func(t *testing.T) {
				cmd := RollbackCommand()
				if cmd == nil {
					t.Fatal("expected command to be non-nil")
				}
			},
		},
		{
			name: "Use is set to rollback",
			check: func(t *testing.T) {
				cmd := RollbackCommand()
				if cmd.Use != "rollback" {
					t.Errorf("expected Use to be 'rollback', got %s", cmd.Use)
				}
			},
		},
		{
			name: "Long description mentions StopDeployment",
			check: func(t *testing.T) {
				cmd := RollbackCommand()
				if !strings.Contains(cmd.Long, "StopDeployment") {
					t.Error("expected Long description to mention StopDeployment")
				}
			},
		},
		{
			name: "Short description is not empty",
			check: func(t *testing.T) {
				cmd := RollbackCommand()
				if cmd.Short == "" {
					t.Error("expected Short description to be non-empty")
				}
			},
		},
		{
			name: "RunE is set",
			check: func(t *testing.T) {
				cmd := RollbackCommand()
				if cmd.RunE == nil {
					t.Error("expected RunE to be set")
				}
			},
		},
		{
			name: "SilenceUsage is true",
			check: func(t *testing.T) {
				cmd := RollbackCommand()
				if !cmd.SilenceUsage {
					t.Error("expected SilenceUsage to be true")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.check(t)
		})
	}
}
