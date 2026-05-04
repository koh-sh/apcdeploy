package cmd

import (
	"bytes"
	"testing"
)

func TestSetLLMsContent(t *testing.T) {
	prev := llmsContent
	t.Cleanup(func() { llmsContent = prev })

	SetLLMsContent("hello-llms")
	if llmsContent != "hello-llms" {
		t.Errorf("llmsContent = %q, want %q", llmsContent, "hello-llms")
	}
}

func TestContextCommand_ExecutesWithoutError(t *testing.T) {
	prev := llmsContent
	t.Cleanup(func() { llmsContent = prev })

	SetLLMsContent("# llms\n")
	cmd := ContextCommand()

	// The command writes llmsContent via fmt.Print to os.Stdout, which
	// Cobra does not redirect via SetOut, so we cannot easily capture it
	// here. This test exercises the wiring (cobra args, RunE, content
	// flow) and trusts the embed/print path; the actual rendered string
	// is covered by the e2e harness.
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	if err := cmd.Execute(); err != nil {
		t.Errorf("Execute: %v", err)
	}
}

func TestSetVersionInfo(t *testing.T) {
	pv, pc, pd := version, commit, date
	t.Cleanup(func() {
		version, commit, date = pv, pc, pd
	})

	SetVersionInfo("1.2.3", "abc123", "2026-05-04")
	if version != "1.2.3" || commit != "abc123" || date != "2026-05-04" {
		t.Errorf("SetVersionInfo: got version=%q commit=%q date=%q",
			version, commit, date)
	}
}
