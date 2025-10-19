package cmd

import (
	"strings"
	"testing"
)

func TestRollbackCommand(t *testing.T) {
	t.Parallel()

	cmd := RollbackCommand()

	if cmd == nil {
		t.Fatal("expected command to be non-nil")
	}

	if cmd.Use != "rollback" {
		t.Errorf("expected Use to be 'rollback', got %s", cmd.Use)
	}

	if !strings.Contains(cmd.Long, "StopDeployment") {
		t.Error("expected Long description to mention StopDeployment")
	}
}
