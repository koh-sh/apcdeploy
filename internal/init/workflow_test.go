package init

import (
	"context"
	"testing"

	"github.com/koh-sh/apcdeploy/internal/prompt"
	promptTesting "github.com/koh-sh/apcdeploy/internal/prompt/testing"
	reporterTesting "github.com/koh-sh/apcdeploy/internal/reporter/testing"
)

func TestNewInitWorkflow(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		opts         *Options
		setupMock    func() *promptTesting.MockPrompter
		expectError  bool
		validateFunc func(*testing.T, *InitWorkflow, error)
	}{
		{
			name: "with all flags provided",
			opts: &Options{
				Application: "test-app",
				Profile:     "test-profile",
				Environment: "test-env",
				Region:      "us-east-1",
				ConfigFile:  "apcdeploy.yml",
				OutputData:  "data.json",
				Force:       false,
			},
			setupMock: func() *promptTesting.MockPrompter {
				return &promptTesting.MockPrompter{}
			},
			expectError: false,
			validateFunc: func(t *testing.T, workflow *InitWorkflow, err error) {
				if workflow == nil {
					t.Error("expected non-nil workflow")
				}
				if workflow != nil && workflow.awsClient == nil {
					t.Error("expected awsClient to be initialized")
				}
			},
		},
		{
			name: "without region triggers interactive selection",
			opts: &Options{
				Application: "test-app",
				Profile:     "test-profile",
				Environment: "test-env",
				Region:      "",
				ConfigFile:  "apcdeploy.yml",
				OutputData:  "data.json",
				Force:       false,
			},
			setupMock: func() *promptTesting.MockPrompter {
				return &promptTesting.MockPrompter{
					SelectFunc: func(message string, options []string) (string, error) {
						return "us-east-1", nil
					},
				}
			},
			expectError: false,
			validateFunc: func(t *testing.T, workflow *InitWorkflow, err error) {
				// Either success or error is acceptable depending on environment
				if err == nil && workflow == nil {
					t.Error("expected either error or non-nil workflow")
				}
			},
		},
		{
			name: "without region and TTY check fails",
			opts: &Options{
				Application: "test-app",
				Profile:     "test-profile",
				Environment: "test-env",
				Region:      "",
				ConfigFile:  "apcdeploy.yml",
				OutputData:  "data.json",
				Force:       false,
			},
			setupMock: func() *promptTesting.MockPrompter {
				return &promptTesting.MockPrompter{
					CheckTTYFunc: func() error {
						return prompt.ErrNoTTY
					},
				}
			},
			expectError: true,
			validateFunc: func(t *testing.T, workflow *InitWorkflow, err error) {
				if err == nil {
					t.Fatal("expected error when TTY check fails")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			mockReporter := &reporterTesting.MockReporter{}
			mockPrompter := tt.setupMock()

			workflow, err := NewInitWorkflow(ctx, tt.opts, mockPrompter, mockReporter)

			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectError && err != nil && tt.validateFunc != nil {
				// Some tests allow errors depending on environment
				tt.validateFunc(t, workflow, err)
				return
			}

			if tt.validateFunc != nil {
				tt.validateFunc(t, workflow, err)
			}
		})
	}
}

func TestNewInitWorkflowWithClient(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		validateFunc func(*testing.T, *InitWorkflow)
	}{
		{
			name: "creates workflow with provided client",
			validateFunc: func(t *testing.T, workflow *InitWorkflow) {
				if workflow == nil {
					t.Fatal("expected non-nil workflow")
				}
				if workflow.selector == nil {
					t.Error("expected selector to be initialized")
				}
				if workflow.initializer == nil {
					t.Error("expected initializer to be initialized")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			mockReporter := &reporterTesting.MockReporter{}
			mockPrompter := &promptTesting.MockPrompter{}

			workflow := NewInitWorkflowWithClient(nil, mockPrompter, mockReporter)

			if workflow.reporter != mockReporter {
				t.Error("expected reporter to be set")
			}
			if workflow.prompter != mockPrompter {
				t.Error("expected prompter to be set")
			}

			if tt.validateFunc != nil {
				tt.validateFunc(t, workflow)
			}
		})
	}
}
