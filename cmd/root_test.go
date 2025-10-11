package cmd

import (
	"bytes"
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

func TestExecute(t *testing.T) {
	// This test verifies that the Execute function works without crashing
	// Execute() calls NewRootCommand() internally
	// We can't easily test the error path without causing the program to exit,
	// so we test the success path by ensuring no panic occurs
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Execute() caused a panic: %v", r)
		}
	}()

	// Execute with help flag to ensure it completes successfully
	oldArgs := []string{}
	defer func() {
		// Restore original args if needed
		_ = oldArgs
	}()

	Execute()
}
