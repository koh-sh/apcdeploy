package aws

import (
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/appconfig/types"
	"github.com/aws/smithy-go"
)

// wrapAWSError wraps an AWS API error with additional context
func wrapAWSError(err error, operation string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s failed: %w", operation, err)
}

// IsValidationError checks if the error is a validation error
func IsValidationError(err error) bool {
	if err == nil {
		return false
	}

	// Check for typed error
	var badRequestErr *types.BadRequestException
	if errors.As(err, &badRequestErr) {
		return true
	}

	// Check for generic API error
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		return apiErr.ErrorCode() == "BadRequestException"
	}

	return false
}

// FormatValidationError formats a validation error with detailed information
func FormatValidationError(err error) string {
	var sb strings.Builder

	sb.WriteString("Configuration validation failed:\n\n")

	// Extract detailed error message
	var badRequestErr *types.BadRequestException
	if errors.As(err, &badRequestErr) {
		if badRequestErr.Message != nil {
			fmt.Fprintf(&sb, "  %s\n", *badRequestErr.Message)
		}
	} else {
		fmt.Fprintf(&sb, "  %s\n", err.Error())
	}

	sb.WriteString("\nPossible causes:\n")
	sb.WriteString("  - JSON Schema validation failed (if JSON Schema validator is configured)\n")
	sb.WriteString("  - Lambda function validation failed (if Lambda validator is configured)\n")
	sb.WriteString("  - Invalid JSON/YAML syntax\n")
	sb.WriteString("  - Configuration does not match the expected schema\n")
	sb.WriteString("\nPlease check your configuration data and validators.")

	return sb.String()
}
