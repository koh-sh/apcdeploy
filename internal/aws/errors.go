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

// isAccessDeniedError checks if the error is an access denied error
func isAccessDeniedError(err error) bool {
	if err == nil {
		return false
	}

	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		code := apiErr.ErrorCode()
		return code == "AccessDeniedException" ||
			code == "UnauthorizedException" ||
			code == "ForbiddenException"
	}

	return false
}

// isResourceNotFoundError checks if the error is a resource not found error
func isResourceNotFoundError(err error) bool {
	if err == nil {
		return false
	}

	// Check for typed error
	var notFoundErr *types.ResourceNotFoundException
	if errors.As(err, &notFoundErr) {
		return true
	}

	// Check for generic API error
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		return apiErr.ErrorCode() == "ResourceNotFoundException"
	}

	return false
}

// isThrottlingError checks if the error is a throttling error
func isThrottlingError(err error) bool {
	if err == nil {
		return false
	}

	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		code := apiErr.ErrorCode()
		return code == "ThrottlingException" ||
			code == "ThrottledException" ||
			code == "TooManyRequestsException" ||
			code == "RequestLimitExceeded"
	}

	return false
}

// formatAccessDeniedError formats an access denied error with helpful information
func formatAccessDeniedError(operation string) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Access denied for operation: %s\n\n", operation))
	sb.WriteString("Required IAM permissions:\n")
	sb.WriteString(fmt.Sprintf("  - appconfig:%s\n", operation))
	sb.WriteString("\nPlease ensure your IAM user/role has the necessary AppConfig permissions.\n")
	sb.WriteString("For more information, see: https://docs.aws.amazon.com/appconfig/latest/userguide/security-iam.html")

	return sb.String()
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
			sb.WriteString(fmt.Sprintf("  %s\n", *badRequestErr.Message))
		}
	} else {
		sb.WriteString(fmt.Sprintf("  %s\n", err.Error()))
	}

	sb.WriteString("\nPossible causes:\n")
	sb.WriteString("  - JSON Schema validation failed (if JSON Schema validator is configured)\n")
	sb.WriteString("  - Lambda function validation failed (if Lambda validator is configured)\n")
	sb.WriteString("  - Invalid JSON/YAML syntax\n")
	sb.WriteString("  - Configuration does not match the expected schema\n")
	sb.WriteString("\nPlease check your configuration data and validators.")

	return sb.String()
}

// formatUserFriendlyError converts AWS errors into user-friendly messages
func formatUserFriendlyError(err error, operation string) string {
	if err == nil {
		return ""
	}

	// Check for validation errors
	if IsValidationError(err) {
		return FormatValidationError(err)
	}

	// Check for access denied
	if isAccessDeniedError(err) {
		return formatAccessDeniedError(operation)
	}

	// Check for resource not found
	if isResourceNotFoundError(err) {
		return fmt.Sprintf("Resource not found during %s operation. Please verify the resource exists and you have access to it.", operation)
	}

	// Check for throttling
	if isThrottlingError(err) {
		return "Rate limit exceeded. Please wait a moment and try again."
	}

	// Default: return the original error message
	return err.Error()
}
