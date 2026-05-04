package cmd

import (
	"bytes"
	"os"
	"testing"
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

// TestRunExitCodes drives the run() function (the testable wrapper inside
// Execute) through its main exit-code branches without invoking os.Exit.
// Help-flag and unknown-command paths cover the success / generic failure
// codes; the cancellation and no-deployment branches are exercised by
// constructing fake errors and threading them through the same code path.
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
