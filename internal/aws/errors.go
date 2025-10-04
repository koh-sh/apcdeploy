package aws

import (
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/appconfig/types"
	"github.com/aws/smithy-go"
)

// WrapAWSError wraps an AWS API error with additional context
func WrapAWSError(err error, operation string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s failed: %w", operation, err)
}

// IsAccessDeniedError checks if the error is an access denied error
func IsAccessDeniedError(err error) bool {
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

// IsResourceNotFoundError checks if the error is a resource not found error
func IsResourceNotFoundError(err error) bool {
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

// IsThrottlingError checks if the error is a throttling error
func IsThrottlingError(err error) bool {
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

// FormatAccessDeniedError formats an access denied error with helpful information
func FormatAccessDeniedError(operation string) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Access denied for operation: %s\n\n", operation))
	sb.WriteString("Required IAM permissions:\n")
	sb.WriteString(fmt.Sprintf("  - appconfig:%s\n", operation))
	sb.WriteString("\nPlease ensure your IAM user/role has the necessary AppConfig permissions.\n")
	sb.WriteString("For more information, see: https://docs.aws.amazon.com/appconfig/latest/userguide/security-iam.html")

	return sb.String()
}

// FormatUserFriendlyError converts AWS errors into user-friendly messages
func FormatUserFriendlyError(err error, operation string) string {
	if err == nil {
		return ""
	}

	// Check for access denied
	if IsAccessDeniedError(err) {
		return FormatAccessDeniedError(operation)
	}

	// Check for resource not found
	if IsResourceNotFoundError(err) {
		return fmt.Sprintf("Resource not found during %s operation. Please verify the resource exists and you have access to it.", operation)
	}

	// Check for throttling
	if IsThrottlingError(err) {
		return "Rate limit exceeded. Please wait a moment and try again."
	}

	// Default: return the original error message
	return err.Error()
}
