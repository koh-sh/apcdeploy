// Package testing provides test utilities for the reporter package.
package testing

import (
	"strings"

	"github.com/koh-sh/apcdeploy/internal/reporter"
)

// MockReporter is a test implementation of ProgressReporter
type MockReporter struct {
	Messages []string
}

// Ensure MockReporter implements the interface
var _ reporter.ProgressReporter = (*MockReporter)(nil)

func (m *MockReporter) Progress(message string) {
	m.Messages = append(m.Messages, "progress: "+message)
}

func (m *MockReporter) Success(message string) {
	m.Messages = append(m.Messages, "success: "+message)
}

func (m *MockReporter) Warning(message string) {
	m.Messages = append(m.Messages, "warning: "+message)
}

// HasMessage checks if the reporter received a message containing the given text
func (m *MockReporter) HasMessage(text string) bool {
	for _, msg := range m.Messages {
		if strings.Contains(msg, text) {
			return true
		}
	}
	return false
}

// Clear clears all messages
func (m *MockReporter) Clear() {
	m.Messages = nil
}
