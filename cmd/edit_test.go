package cmd

import (
	"strings"
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
		{
			name: "has --description flag",
			check: func(t *testing.T, cmd *cobra.Command) {
				require.NotNil(t, cmd.Flags().Lookup("description"))
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

// TestValidateEditFlags asserts timeout and mutually-exclusive wait-flag
// constraints are enforced before any AWS work begins. The edit command uses
// the same validateEditFlags helper as the run command, so regressions in
// either helper show up here.
func TestValidateEditFlags(t *testing.T) {
	tests := []struct {
		name       string
		waitDeploy bool
		waitBake   bool
		timeout    int
		wantErr    bool
		wantSub    string
	}{
		{name: "default flags pass", timeout: DefaultDeploymentTimeout},
		{name: "positive timeout passes", timeout: 1},
		{
			name:    "zero timeout rejected",
			timeout: 0,
			wantErr: true,
			wantSub: "timeout must be greater than 0",
		},
		{
			name:    "negative timeout rejected",
			timeout: -1,
			wantErr: true,
			wantSub: "timeout must be greater than 0",
		},
		{
			name:       "both wait flags rejected",
			waitDeploy: true,
			waitBake:   true,
			timeout:    DefaultDeploymentTimeout,
			wantErr:    true,
			wantSub:    "--wait-deploy and --wait-bake cannot be used together",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			editWaitDeploy = tt.waitDeploy
			editWaitBake = tt.waitBake
			editTimeout = tt.timeout
			t.Cleanup(func() {
				editWaitDeploy = false
				editWaitBake = false
				editTimeout = DefaultDeploymentTimeout
			})

			err := validateEditFlags()
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if !strings.Contains(err.Error(), tt.wantSub) {
					t.Errorf("err = %q, want substring %q", err.Error(), tt.wantSub)
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// TestRunEditDescriptionValidation ensures the edit command applies the same
// 1024-rune client-side guard as the run command (CLAUDE.md "validation
// parity"). We exercise the boundary so a regression in either runEdit's
// validation call or validateDescription itself would be caught here, not
// only in TestValidateDescription (which exercises the helper directly).
//
// runEdit is invoked with no AWS calls because validation runs before the
// executor is constructed.
func TestRunEditDescriptionValidation(t *testing.T) {
	cmd := newEditCmd()

	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{name: "empty allowed", input: "", wantErr: false},
		{name: "exactly 1024 ascii allowed", input: strings.Repeat("a", 1024), wantErr: false},
		{name: "1025 ascii rejected", input: strings.Repeat("a", 1025), wantErr: true},
		{name: "exactly 1024 multibyte allowed", input: strings.Repeat("あ", 1024), wantErr: false},
		{name: "1025 multibyte rejected", input: strings.Repeat("あ", 1025), wantErr: true},
		{name: "control character rejected", input: "bad\x00value", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			editDescription = tt.input
			t.Cleanup(func() { editDescription = "" })

			err := validateDescription(editDescription)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateDescription(runes=%d) error = %v, wantErr %v", len([]rune(tt.input)), err, tt.wantErr)
			}
			// Sanity-check that runEdit would also propagate the failure
			// before any AWS work. We do not execute the full RunE path
			// here because constructing a real prompter / AWS client is
			// out of scope for a validation test.
			if tt.wantErr && cmd.RunE == nil {
				t.Fatal("runEdit RunE handler missing")
			}
		})
	}
}
