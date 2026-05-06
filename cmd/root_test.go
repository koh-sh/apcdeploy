package cmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"testing"

	smithy "github.com/aws/smithy-go"
	awsInternal "github.com/koh-sh/apcdeploy/internal/aws"
	reportertest "github.com/koh-sh/apcdeploy/internal/reporter/testing"
	"github.com/koh-sh/apcdeploy/internal/rollback"
)

func TestRootCommand(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "root command without args",
			args:    []string{},
			wantErr: false,
		},
		{
			name:    "help flag",
			args:    []string{"--help"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rootCmd := NewRootCommand()
			rootCmd.SetArgs(tt.args)

			buf := new(bytes.Buffer)
			rootCmd.SetOut(buf)
			rootCmd.SetErr(buf)

			err := rootCmd.Execute()
			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestVersionFlag(t *testing.T) {
	rootCmd := NewRootCommand()
	rootCmd.SetArgs([]string{"--version"})

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	err := rootCmd.Execute()
	if err != nil {
		t.Errorf("Execute() error = %v", err)
	}

	output := buf.String()
	if output == "" {
		t.Error("Expected version output, got empty string")
	}
}

func TestGlobalFlags(t *testing.T) {
	rootCmd := NewRootCommand()

	// Test --config flag
	configFlag := rootCmd.PersistentFlags().Lookup("config")
	if configFlag == nil {
		t.Error("config flag not found")
	}
}

// TestRunExitCodes drives the runRoot() wrapper through its end-to-end
// success / unknown-subcommand branches without invoking os.Exit. The
// other exit-code branches (context.Canceled → 130, ErrNoDeployment → 2,
// Resolution hint emission) live in classifyAndReport and are tested by
// TestClassifyAndReport, which avoids the cobra/signal plumbing.
func TestRunExitCodes(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantCode int
	}{
		{name: "help flag returns 0", args: []string{"--help"}, wantCode: 0},
		{name: "unknown subcommand returns 1", args: []string{"definitely-not-a-real-cmd"}, wantCode: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldArgs := os.Args
			defer func() { os.Args = oldArgs }()
			os.Args = append([]string{"apcdeploy"}, tt.args...)
			if got := runRoot(); got != tt.wantCode {
				t.Errorf("runRoot() = %d, want %d", got, tt.wantCode)
			}
		})
	}
}

// TestClassifyAndReport exercises every exit-code branch of the top-level
// error classifier directly, without the cobra/signal plumbing in
// runRoot. This is the only place that asserts the 130 (cancelled) and
// 2 (no-prior-deployment) paths and the Reporter-side effects (warn for
// cancellation / Resolution hint, error for fatal).
func TestClassifyAndReport(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		err          error
		wantCode     int
		wantMessages []string // substring matches in MockReporter.Messages
		wantNoError  bool     // assert no "error:" line was emitted
	}{
		{
			name:        "nil error returns 0 and emits nothing",
			err:         nil,
			wantCode:    0,
			wantNoError: true,
		},
		{
			name:         "context.Canceled returns 130 with cancellation warn (no error line)",
			err:          context.Canceled,
			wantCode:     exitInterrupted,
			wantMessages: []string{"warn: cancelled by user"},
			wantNoError:  true,
		},
		{
			name:         "wrapped context.Canceled also returns 130",
			err:          fmt.Errorf("polling failed: %w", context.Canceled),
			wantCode:     exitInterrupted,
			wantMessages: []string{"warn: cancelled by user"},
			wantNoError:  true,
		},
		{
			name:         "ErrNoDeployment returns 2 with error line",
			err:          awsInternal.ErrNoDeployment,
			wantCode:     exitNoDeployment,
			wantMessages: []string{"error: " + awsInternal.ErrNoDeployment.Error()},
		},
		{
			name:         "wrapped ErrNoDeployment also returns 2",
			err:          fmt.Errorf("pull failed: %w", awsInternal.ErrNoDeployment),
			wantCode:     exitNoDeployment,
			wantMessages: []string{"error: pull failed"},
		},
		{
			name:         "ErrNoOngoingDeployment returns 2 with error line",
			err:          rollback.ErrNoOngoingDeployment,
			wantCode:     exitNoDeployment,
			wantMessages: []string{"error: " + rollback.ErrNoOngoingDeployment.Error()},
		},
		{
			name:         "wrapped ErrNoOngoingDeployment also returns 2",
			err:          fmt.Errorf("rollback failed: %w", rollback.ErrNoOngoingDeployment),
			wantCode:     exitNoDeployment,
			wantMessages: []string{"error: rollback failed"},
		},
		{
			name:         "generic error returns 1 with error line and no resolution",
			err:          errors.New("something blew up"),
			wantCode:     1,
			wantMessages: []string{"error: something blew up"},
		},
		{
			name:     "AWS ConflictException returns 1 and emits Resolution hint",
			err:      &smithy.GenericAPIError{Code: "ConflictException", Message: "in progress"},
			wantCode: 1,
			wantMessages: []string{
				"error: ",
				"warn: Resolution: wait for the current deployment to complete or run 'apcdeploy rollback'",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			rep := &reportertest.MockReporter{}
			got := classifyAndReport(tt.err, rep)
			if got != tt.wantCode {
				t.Errorf("classifyAndReport() = %d, want %d (messages: %v)", got, tt.wantCode, rep.Messages)
			}
			for _, want := range tt.wantMessages {
				if !rep.HasMessage(want) {
					t.Errorf("missing reporter message %q; got %v", want, rep.Messages)
				}
			}
			if tt.wantNoError {
				for _, m := range rep.Messages {
					if len(m) >= 7 && m[:7] == "error: " {
						t.Errorf("expected no error line; got %q", m)
					}
				}
			}
		})
	}
}
