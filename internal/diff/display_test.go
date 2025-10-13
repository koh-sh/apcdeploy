package diff

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/koh-sh/apcdeploy/internal/aws"
	"github.com/koh-sh/apcdeploy/internal/config"
)

func Test_displayColorizedDiff(t *testing.T) {
	tests := []struct {
		name string
		diff string
	}{
		{
			name: "empty diff",
			diff: "",
		},
		{
			name: "additions only",
			diff: "+added line 1\n+added line 2",
		},
		{
			name: "deletions only",
			diff: "-removed line 1\n-removed line 2",
		},
		{
			name: "diff headers",
			diff: "@@ -1,3 +1,3 @@\ncontext line",
		},
		{
			name: "mixed changes",
			diff: "@@ -1,3 +1,3 @@\n context line\n-removed line\n+added line",
		},
		{
			name: "empty lines in diff",
			diff: "+added\n\n-removed",
		},
		{
			name: "context lines",
			diff: " context line 1\n context line 2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Just verify it doesn't panic
			displayColorizedDiff(tt.diff)
		})
	}
}

func TestDisplaySilent(t *testing.T) {
	tests := []struct {
		name       string
		result     *Result
		deployment *aws.DeploymentInfo
		wantOutput bool
		wantText   string
	}{
		{
			name: "no changes - should produce no output",
			result: &Result{
				HasChanges:  false,
				UnifiedDiff: "",
				FileName:    "data.json",
			},
			deployment: nil,
			wantOutput: false,
		},
		{
			name: "has changes - should show diff only",
			result: &Result{
				HasChanges:  true,
				UnifiedDiff: "+added line\n-removed line",
				FileName:    "data.json",
			},
			deployment: nil,
			wantOutput: true,
			wantText:   "added line",
		},
		{
			name: "has changes with multiple lines",
			result: &Result{
				HasChanges:  true,
				UnifiedDiff: "@@ -1,3 +1,3 @@\n context\n+new line\n-old line",
				FileName:    "config.yaml",
			},
			deployment: nil,
			wantOutput: true,
			wantText:   "new line",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := captureOutput(func() {
				displaySilent(tt.result, tt.deployment)
			})

			if tt.wantOutput {
				if output == "" {
					t.Error("displaySilent() expected output but got empty string")
				}
				if tt.wantText != "" && !strings.Contains(output, tt.wantText) {
					t.Errorf("displaySilent() output missing %q\nGot:\n%s", tt.wantText, output)
				}
			} else if output != "" {
				t.Errorf("displaySilent() expected no output but got:\n%s", output)
			}
		})
	}
}

func TestDisplay(t *testing.T) {
	tests := []struct {
		name               string
		result             *Result
		cfg                *config.Config
		resources          *aws.ResolvedResources
		deployment         *aws.DeploymentInfo
		wantWarning        bool
		wantWarningContain string
	}{
		{
			name: "deployment in DEPLOYING state - should show warning after summary",
			result: &Result{
				HasChanges:  true,
				UnifiedDiff: "+new line\n-old line",
				FileName:    "data.json",
			},
			cfg: &config.Config{
				Application:          "test-app",
				ConfigurationProfile: "test-prof",
				Environment:          "test-env",
				DataFile:             "data.json",
			},
			resources: &aws.ResolvedResources{
				Profile: &aws.ProfileInfo{
					Name: "test-prof",
					Type: "AWS.Freeform",
				},
			},
			deployment: &aws.DeploymentInfo{
				DeploymentNumber:     42,
				ConfigurationVersion: "1",
				State:                "DEPLOYING",
			},
			wantWarning:        true,
			wantWarningContain: "Deployment #42 is currently DEPLOYING",
		},
		{
			name: "deployment in BAKING state - should show warning after summary",
			result: &Result{
				HasChanges:  true,
				UnifiedDiff: "+new line\n-old line",
				FileName:    "data.json",
			},
			cfg: &config.Config{
				Application:          "test-app",
				ConfigurationProfile: "test-prof",
				Environment:          "test-env",
				DataFile:             "data.json",
			},
			resources: &aws.ResolvedResources{
				Profile: &aws.ProfileInfo{
					Name: "test-prof",
					Type: "AWS.Freeform",
				},
			},
			deployment: &aws.DeploymentInfo{
				DeploymentNumber:     42,
				ConfigurationVersion: "1",
				State:                "BAKING",
			},
			wantWarning:        true,
			wantWarningContain: "Deployment #42 is currently BAKING",
		},
		{
			name: "deployment in COMPLETE state - should not show warning",
			result: &Result{
				HasChanges:  true,
				UnifiedDiff: "+new line\n-old line",
				FileName:    "data.json",
			},
			cfg: &config.Config{
				Application:          "test-app",
				ConfigurationProfile: "test-prof",
				Environment:          "test-env",
				DataFile:             "data.json",
			},
			resources: &aws.ResolvedResources{
				Profile: &aws.ProfileInfo{
					Name: "test-prof",
					Type: "AWS.Freeform",
				},
			},
			deployment: &aws.DeploymentInfo{
				DeploymentNumber:     42,
				ConfigurationVersion: "1",
				State:                "COMPLETE",
			},
			wantWarning: false,
		},
		{
			name: "no changes and deployment in DEPLOYING state - should show warning",
			result: &Result{
				HasChanges:  false,
				UnifiedDiff: "",
				FileName:    "data.json",
			},
			cfg: &config.Config{
				Application:          "test-app",
				ConfigurationProfile: "test-prof",
				Environment:          "test-env",
				DataFile:             "data.json",
			},
			resources: &aws.ResolvedResources{
				Profile: &aws.ProfileInfo{
					Name: "test-prof",
					Type: "AWS.Freeform",
				},
			},
			deployment: &aws.DeploymentInfo{
				DeploymentNumber:     42,
				ConfigurationVersion: "1",
				State:                "DEPLOYING",
			},
			wantWarning:        true,
			wantWarningContain: "Deployment #42 is currently DEPLOYING",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, stderr := captureOutputAndError(func() {
				display(tt.result, tt.cfg, tt.resources, tt.deployment)
			})

			if tt.wantWarning {
				if !strings.Contains(stderr, tt.wantWarningContain) {
					t.Errorf("display() stderr missing warning %q\nGot:\n%s", tt.wantWarningContain, stderr)
				}
			} else if strings.Contains(stderr, "is currently") {
				t.Errorf("display() should not show warning but got:\n%s", stderr)
			}
		})
	}
}

func TestDisplaySilentWithDeployment(t *testing.T) {
	tests := []struct {
		name               string
		result             *Result
		deployment         *aws.DeploymentInfo
		wantWarning        bool
		wantWarningContain string
	}{
		{
			name: "has changes with DEPLOYING deployment - should show diff and warning",
			result: &Result{
				HasChanges:  true,
				UnifiedDiff: "+added line\n-removed line",
				FileName:    "data.json",
			},
			deployment: &aws.DeploymentInfo{
				DeploymentNumber:     22,
				ConfigurationVersion: "1",
				State:                "DEPLOYING",
			},
			wantWarning:        true,
			wantWarningContain: "Deployment #22 is currently DEPLOYING",
		},
		{
			name: "has changes with BAKING deployment - should show diff and warning",
			result: &Result{
				HasChanges:  true,
				UnifiedDiff: "+added line\n-removed line",
				FileName:    "data.json",
			},
			deployment: &aws.DeploymentInfo{
				DeploymentNumber:     22,
				ConfigurationVersion: "1",
				State:                "BAKING",
			},
			wantWarning:        true,
			wantWarningContain: "Deployment #22 is currently BAKING",
		},
		{
			name: "has changes with COMPLETE deployment - should show diff only",
			result: &Result{
				HasChanges:  true,
				UnifiedDiff: "+added line\n-removed line",
				FileName:    "data.json",
			},
			deployment: &aws.DeploymentInfo{
				DeploymentNumber:     22,
				ConfigurationVersion: "1",
				State:                "COMPLETE",
			},
			wantWarning: false,
		},
		{
			name: "no changes with DEPLOYING deployment - should show warning only",
			result: &Result{
				HasChanges:  false,
				UnifiedDiff: "",
				FileName:    "data.json",
			},
			deployment: &aws.DeploymentInfo{
				DeploymentNumber:     22,
				ConfigurationVersion: "1",
				State:                "DEPLOYING",
			},
			wantWarning:        true,
			wantWarningContain: "Deployment #22 is currently DEPLOYING",
		},
		{
			name: "no changes with COMPLETE deployment - should show nothing",
			result: &Result{
				HasChanges:  false,
				UnifiedDiff: "",
				FileName:    "data.json",
			},
			deployment: &aws.DeploymentInfo{
				DeploymentNumber:     22,
				ConfigurationVersion: "1",
				State:                "COMPLETE",
			},
			wantWarning: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, stderr := captureOutputAndError(func() {
				displaySilent(tt.result, tt.deployment)
			})

			if tt.wantWarning {
				if !strings.Contains(stderr, tt.wantWarningContain) {
					t.Errorf("displaySilent() stderr missing warning %q\nGot:\n%s", tt.wantWarningContain, stderr)
				}
			} else if strings.Contains(stderr, "is currently") {
				t.Errorf("displaySilent() should not show warning but got:\n%s", stderr)
			}
		})
	}
}

func Test_countChanges(t *testing.T) {
	tests := []struct {
		name        string
		diff        string
		wantAdded   int
		wantRemoved int
	}{
		{
			name:        "empty diff",
			diff:        "",
			wantAdded:   0,
			wantRemoved: 0,
		},
		{
			name:        "additions only",
			diff:        "+added line 1\n+added line 2",
			wantAdded:   2,
			wantRemoved: 0,
		},
		{
			name:        "deletions only",
			diff:        "-removed line 1\n-removed line 2",
			wantAdded:   0,
			wantRemoved: 2,
		},
		{
			name:        "mixed changes",
			diff:        "+added\n-removed\n context",
			wantAdded:   1,
			wantRemoved: 1,
		},
		{
			name:        "ignore file headers",
			diff:        "--- a/file.json\n+++ b/file.json\n+added\n-removed",
			wantAdded:   1,
			wantRemoved: 1,
		},
		{
			name:        "multiple additions and deletions",
			diff:        "+line1\n+line2\n-line3\n-line4\n-line5",
			wantAdded:   2,
			wantRemoved: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			added, removed := countChanges(tt.diff)
			if added != tt.wantAdded {
				t.Errorf("countChanges() added = %v, want %v", added, tt.wantAdded)
			}
			if removed != tt.wantRemoved {
				t.Errorf("countChanges() removed = %v, want %v", removed, tt.wantRemoved)
			}
		})
	}
}

// Helper functions
func captureOutput(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		panic(err)
	}
	return buf.String()
}

func captureOutputAndError(f func()) (stdout, stderr string) {
	oldStdout := os.Stdout
	oldStderr := os.Stderr

	rOut, wOut, _ := os.Pipe()
	rErr, wErr, _ := os.Pipe()

	os.Stdout = wOut
	os.Stderr = wErr

	f()

	wOut.Close()
	wErr.Close()

	os.Stdout = oldStdout
	os.Stderr = oldStderr

	var bufOut, bufErr bytes.Buffer
	if _, err := io.Copy(&bufOut, rOut); err != nil {
		panic(err)
	}
	if _, err := io.Copy(&bufErr, rErr); err != nil {
		panic(err)
	}

	return bufOut.String(), bufErr.String()
}
