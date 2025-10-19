package prompt_test

import (
	"errors"
	"testing"

	promptTesting "github.com/koh-sh/apcdeploy/internal/prompt/testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMockPrompter_Select(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		selectFunc     func(message string, options []string) (string, error)
		message        string
		options        []string
		expectedResult string
		expectedError  error
		checkError     func(*testing.T, error)
	}{
		{
			name: "returns configured value",
			selectFunc: func(message string, options []string) (string, error) {
				return "option2", nil
			},
			message:        "Choose",
			options:        []string{"option1", "option2"},
			expectedResult: "option2",
			expectedError:  nil,
			checkError: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "returns configured error",
			selectFunc: func(message string, options []string) (string, error) {
				return "", errors.New("test error")
			},
			message:        "Choose",
			options:        []string{"option1", "option2"},
			expectedResult: "",
			checkError: func(t *testing.T, err error) {
				require.Error(t, err)
				assert.EqualError(t, err, "test error")
			},
		},
		{
			name:           "default returns empty",
			selectFunc:     nil,
			message:        "Choose",
			options:        []string{"option1", "option2"},
			expectedResult: "",
			checkError: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "returns first option",
			selectFunc: func(message string, options []string) (string, error) {
				return "option1", nil
			},
			message:        "Select one",
			options:        []string{"option1", "option2", "option3"},
			expectedResult: "option1",
			checkError: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "handles empty options",
			selectFunc: func(message string, options []string) (string, error) {
				return "", nil
			},
			message:        "Choose",
			options:        []string{},
			expectedResult: "",
			checkError: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mock := &promptTesting.MockPrompter{
				SelectFunc: tt.selectFunc,
			}

			result, err := mock.Select(tt.message, tt.options)

			tt.checkError(t, err)
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}

func TestMockPrompter_Input(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		inputFunc      func(message string, placeholder string) (string, error)
		message        string
		placeholder    string
		expectedResult string
		checkError     func(*testing.T, error)
	}{
		{
			name: "returns configured value",
			inputFunc: func(message string, placeholder string) (string, error) {
				return "user input", nil
			},
			message:        "Enter value",
			placeholder:    "default",
			expectedResult: "user input",
			checkError: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "returns configured error",
			inputFunc: func(message string, placeholder string) (string, error) {
				return "", errors.New("input error")
			},
			message:        "Enter value",
			placeholder:    "default",
			expectedResult: "",
			checkError: func(t *testing.T, err error) {
				require.Error(t, err)
				assert.EqualError(t, err, "input error")
			},
		},
		{
			name:           "default returns empty",
			inputFunc:      nil,
			message:        "Enter value",
			placeholder:    "default",
			expectedResult: "",
			checkError: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mock := &promptTesting.MockPrompter{
				InputFunc: tt.inputFunc,
			}

			result, err := mock.Input(tt.message, tt.placeholder)

			tt.checkError(t, err)
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}

func TestMockPrompter_CheckTTY(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		checkTTYFunc func() error
		checkError   func(*testing.T, error)
	}{
		{
			name: "returns no error",
			checkTTYFunc: func() error {
				return nil
			},
			checkError: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name: "returns configured error",
			checkTTYFunc: func() error {
				return errors.New("no TTY")
			},
			checkError: func(t *testing.T, err error) {
				require.Error(t, err)
				assert.EqualError(t, err, "no TTY")
			},
		},
		{
			name:         "default returns no error",
			checkTTYFunc: nil,
			checkError: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mock := &promptTesting.MockPrompter{
				CheckTTYFunc: tt.checkTTYFunc,
			}

			err := mock.CheckTTY()

			tt.checkError(t, err)
		})
	}
}
