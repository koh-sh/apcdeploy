package prompt_test

import (
	"errors"
	"testing"

	promptTesting "github.com/koh-sh/apcdeploy/internal/prompt/testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMockPrompter_Select_ReturnsConfiguredValue(t *testing.T) {
	t.Parallel()
	mock := &promptTesting.MockPrompter{
		SelectFunc: func(message string, options []string) (string, error) {
			return "option2", nil
		},
	}
	result, err := mock.Select("Choose", []string{"option1", "option2"})
	require.NoError(t, err)
	assert.Equal(t, "option2", result)
}

func TestMockPrompter_Select_ReturnsConfiguredError(t *testing.T) {
	t.Parallel()
	expectedErr := errors.New("test error")
	mock := &promptTesting.MockPrompter{
		SelectFunc: func(message string, options []string) (string, error) {
			return "", expectedErr
		},
	}
	result, err := mock.Select("Choose", []string{"option1", "option2"})
	require.Error(t, err)
	assert.Equal(t, expectedErr, err)
	assert.Empty(t, result)
}

func TestMockPrompter_Select_DefaultReturnsEmpty(t *testing.T) {
	t.Parallel()
	mock := &promptTesting.MockPrompter{}
	result, err := mock.Select("Choose", []string{"option1", "option2"})
	require.NoError(t, err)
	assert.Empty(t, result)
}
