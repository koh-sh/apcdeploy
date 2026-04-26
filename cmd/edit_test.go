package cmd

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestEditCommand(t *testing.T) {
	t.Parallel()

	cmd := EditCommand()

	tests := []struct {
		name  string
		check func(*testing.T, *cobra.Command)
	}{
		{
			name: "command is non-nil",
			check: func(t *testing.T, cmd *cobra.Command) {
				require.NotNil(t, cmd)
			},
		},
		{
			name: "Use is set to edit",
			check: func(t *testing.T, cmd *cobra.Command) {
				require.Equal(t, "edit", cmd.Use)
			},
		},
		{
			name: "Long description mentions EDITOR",
			check: func(t *testing.T, cmd *cobra.Command) {
				require.Contains(t, cmd.Long, "$EDITOR")
			},
		},
		{
			name: "Short description is not empty",
			check: func(t *testing.T, cmd *cobra.Command) {
				require.NotEmpty(t, cmd.Short)
			},
		},
		{
			name: "RunE is set",
			check: func(t *testing.T, cmd *cobra.Command) {
				require.NotNil(t, cmd.RunE)
			},
		},
		{
			name: "SilenceUsage is true",
			check: func(t *testing.T, cmd *cobra.Command) {
				require.True(t, cmd.SilenceUsage)
			},
		},
		{
			name: "has --region flag",
			check: func(t *testing.T, cmd *cobra.Command) {
				require.NotNil(t, cmd.Flags().Lookup("region"))
			},
		},
		{
			name: "has --app flag",
			check: func(t *testing.T, cmd *cobra.Command) {
				require.NotNil(t, cmd.Flags().Lookup("app"))
			},
		},
		{
			name: "has --profile flag",
			check: func(t *testing.T, cmd *cobra.Command) {
				require.NotNil(t, cmd.Flags().Lookup("profile"))
			},
		},
		{
			name: "has --env flag",
			check: func(t *testing.T, cmd *cobra.Command) {
				require.NotNil(t, cmd.Flags().Lookup("env"))
			},
		},
		{
			name: "has --deployment-strategy flag",
			check: func(t *testing.T, cmd *cobra.Command) {
				require.NotNil(t, cmd.Flags().Lookup("deployment-strategy"))
			},
		},
		{
			name: "has --wait-deploy flag",
			check: func(t *testing.T, cmd *cobra.Command) {
				require.NotNil(t, cmd.Flags().Lookup("wait-deploy"))
			},
		},
		{
			name: "has --wait-bake flag",
			check: func(t *testing.T, cmd *cobra.Command) {
				require.NotNil(t, cmd.Flags().Lookup("wait-bake"))
			},
		},
		{
			name: "has --timeout flag",
			check: func(t *testing.T, cmd *cobra.Command) {
				require.NotNil(t, cmd.Flags().Lookup("timeout"))
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
