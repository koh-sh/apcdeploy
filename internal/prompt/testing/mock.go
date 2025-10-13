package testing

import "github.com/koh-sh/apcdeploy/internal/prompt"

// MockPrompter is a test implementation of Prompter
type MockPrompter struct {
	SelectFunc func(message string, options []string) (string, error)
	InputFunc  func(message string, placeholder string) (string, error)
}

// Ensure MockPrompter implements the interface
var _ prompt.Prompter = (*MockPrompter)(nil)

func (m *MockPrompter) Select(message string, options []string) (string, error) {
	if m.SelectFunc != nil {
		return m.SelectFunc(message, options)
	}
	return "", nil
}

func (m *MockPrompter) Input(message string, placeholder string) (string, error) {
	if m.InputFunc != nil {
		return m.InputFunc(message, placeholder)
	}
	return "", nil
}
