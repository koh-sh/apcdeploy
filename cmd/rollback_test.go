package cmd

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestRollbackCommand(t *testing.T) {
	t.Parallel()

	cmd := RollbackCommand()

	tests := []struct {
		name  string
		check func(*testing.T, *cobra.Command)
	}{
		{
			name: "command is non-nil",
			check: func(t *testing.T, cmd *cobra.Command) {
				require.NotNil(t, cmd, "expected command to be non-nil")
			},
		},
		{
			name: "Use is set to rollback",
			check: func(t *testing.T, cmd *cobra.Command) {
				require.Equal(t, "rollback", cmd.Use, "expected Use to be 'rollback'")
			},
		},
		{
			name: "Long description mentions StopDeployment",
			check: func(t *testing.T, cmd *cobra.Command) {
				require.Contains(t, cmd.Long, "StopDeployment", "expected Long description to mention StopDeployment")
			},
		},
		{
			name: "Short description is not empty",
			check: func(t *testing.T, cmd *cobra.Command) {
				require.NotEmpty(t, cmd.Short, "expected Short description to be non-empty")
			},
		},
		{
			name: "RunE is set",
			check: func(t *testing.T, cmd *cobra.Command) {
				require.NotNil(t, cmd.RunE, "expected RunE to be set")
			},
		},
		{
			name: "SilenceUsage is true",
			check: func(t *testing.T, cmd *cobra.Command) {
				require.True(t, cmd.SilenceUsage, "expected SilenceUsage to be true")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.check(t, cmd)
		})
	}
}
