package display

import (
	"strings"
	"testing"
)

func TestSuccess(t *testing.T) {
	tests := []struct {
		name     string
		message  string
		contains string
	}{
		{
			name:     "success message",
			message:  "Deployment completed",
			contains: "Deployment completed",
		},
		{
			name:     "success message with checkmark",
			message:  "File created",
			contains: "✓",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := successMsg(tt.message)
			if !strings.Contains(result, tt.contains) {
				t.Errorf("successMsg() = %s, want to contain %s", result, tt.contains)
			}
		})
	}
}

func TestError(t *testing.T) {
	tests := []struct {
		name     string
		message  string
		contains string
	}{
		{
			name:     "error message",
			message:  "Deployment failed",
			contains: "Deployment failed",
		},
		{
			name:     "error message with cross mark",
			message:  "File not found",
			contains: "✗",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := errorMsg(tt.message)
			if !strings.Contains(result, tt.contains) {
				t.Errorf("errorMsg() = %s, want to contain %s", result, tt.contains)
			}
		})
	}
}

func TestWarning(t *testing.T) {
	tests := []struct {
		name     string
		message  string
		contains string
	}{
		{
			name:     "warning message",
			message:  "Deployment in progress",
			contains: "Deployment in progress",
		},
		{
			name:     "warning message with symbol",
			message:  "Resource limit approaching",
			contains: "⚠",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := warningMsg(tt.message)
			if !strings.Contains(result, tt.contains) {
				t.Errorf("warningMsg() = %s, want to contain %s", result, tt.contains)
			}
		})
	}
}

func TestProgress(t *testing.T) {
	tests := []struct {
		name     string
		message  string
		contains string
	}{
		{
			name:     "progress message",
			message:  "Deploying...",
			contains: "Deploying...",
		},
		{
			name:     "progress message with hourglass",
			message:  "Loading configuration",
			contains: "⏳",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := progressMsg(tt.message)
			if !strings.Contains(result, tt.contains) {
				t.Errorf("progressMsg() = %s, want to contain %s", result, tt.contains)
			}
		})
	}
}
