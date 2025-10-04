package aws

import (
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/appconfig/types"
	"github.com/aws/smithy-go"
)

func TestWrapAWSError(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		operation   string
		wantContain string
	}{
		{
			name:        "wrap generic error",
			err:         errors.New("something went wrong"),
			operation:   "ListApplications",
			wantContain: "ListApplications failed: something went wrong",
		},
		{
			name:        "wrap nil error",
			err:         nil,
			operation:   "GetApplication",
			wantContain: "",
		},
		{
			name: "wrap AWS API error",
			err: &smithy.GenericAPIError{
				Code:    "ResourceNotFoundException",
				Message: "Resource not found",
			},
			operation:   "GetConfigurationProfile",
			wantContain: "GetConfigurationProfile failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wrapped := wrapAWSError(tt.err, tt.operation)

			if tt.err == nil {
				if wrapped != nil {
					t.Errorf("wrapAWSError() with nil error should return nil, got %v", wrapped)
				}
				return
			}

			if wrapped == nil {
				t.Error("wrapAWSError() returned nil for non-nil error")
				return
			}

			errMsg := wrapped.Error()
			if !contains(errMsg, tt.wantContain) {
				t.Errorf("wrapAWSError() error message = %q, want to contain %q", errMsg, tt.wantContain)
			}
		})
	}
}

func TestIsAccessDeniedError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "access denied error",
			err: &smithy.GenericAPIError{
				Code:    "AccessDeniedException",
				Message: "User is not authorized",
			},
			want: true,
		},
		{
			name: "unauthorized error",
			err: &smithy.GenericAPIError{
				Code:    "UnauthorizedException",
				Message: "Not authorized",
			},
			want: true,
		},
		{
			name: "forbidden error",
			err: &smithy.GenericAPIError{
				Code:    "ForbiddenException",
				Message: "Forbidden",
			},
			want: true,
		},
		{
			name: "resource not found error",
			err: &smithy.GenericAPIError{
				Code:    "ResourceNotFoundException",
				Message: "Resource not found",
			},
			want: false,
		},
		{
			name: "generic error",
			err:  errors.New("some error"),
			want: false,
		},
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isAccessDeniedError(tt.err); got != tt.want {
				t.Errorf("isAccessDeniedError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsResourceNotFoundError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "resource not found error",
			err: &smithy.GenericAPIError{
				Code:    "ResourceNotFoundException",
				Message: "Resource not found",
			},
			want: true,
		},
		{
			name: "typed resource not found error",
			err:  &types.ResourceNotFoundException{Message: stringPtr("not found")},
			want: true,
		},
		{
			name: "access denied error",
			err: &smithy.GenericAPIError{
				Code:    "AccessDeniedException",
				Message: "Access denied",
			},
			want: false,
		},
		{
			name: "generic error",
			err:  errors.New("some error"),
			want: false,
		},
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isResourceNotFoundError(tt.err); got != tt.want {
				t.Errorf("isResourceNotFoundError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFormatAccessDeniedError(t *testing.T) {
	tests := []struct {
		name        string
		operation   string
		wantContain []string
	}{
		{
			name:      "list applications operation",
			operation: "ListApplications",
			wantContain: []string{
				"Access denied",
				"ListApplications",
				"appconfig:ListApplications",
			},
		},
		{
			name:      "get configuration profile operation",
			operation: "GetConfigurationProfile",
			wantContain: []string{
				"Access denied",
				"GetConfigurationProfile",
				"appconfig:GetConfigurationProfile",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatAccessDeniedError(tt.operation)

			for _, want := range tt.wantContain {
				if !contains(result, want) {
					t.Errorf("formatAccessDeniedError() = %q, want to contain %q", result, want)
				}
			}
		})
	}
}

func TestIsThrottlingError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "throttling exception",
			err: &smithy.GenericAPIError{
				Code:    "ThrottlingException",
				Message: "Rate exceeded",
			},
			want: true,
		},
		{
			name: "throttled exception",
			err: &smithy.GenericAPIError{
				Code:    "ThrottledException",
				Message: "Throttled",
			},
			want: true,
		},
		{
			name: "too many requests",
			err: &smithy.GenericAPIError{
				Code:    "TooManyRequestsException",
				Message: "Too many requests",
			},
			want: true,
		},
		{
			name: "request limit exceeded",
			err: &smithy.GenericAPIError{
				Code:    "RequestLimitExceeded",
				Message: "Request limit exceeded",
			},
			want: true,
		},
		{
			name: "resource not found error",
			err: &smithy.GenericAPIError{
				Code:    "ResourceNotFoundException",
				Message: "Resource not found",
			},
			want: false,
		},
		{
			name: "generic error",
			err:  errors.New("some error"),
			want: false,
		},
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isThrottlingError(tt.err); got != tt.want {
				t.Errorf("isThrottlingError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsValidationError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "bad request exception via API error",
			err: &smithy.GenericAPIError{
				Code:    "BadRequestException",
				Message: "Validation failed",
			},
			want: true,
		},
		{
			name: "typed bad request exception",
			err:  &types.BadRequestException{Message: stringPtr("Invalid configuration")},
			want: true,
		},
		{
			name: "resource not found error",
			err: &smithy.GenericAPIError{
				Code:    "ResourceNotFoundException",
				Message: "Resource not found",
			},
			want: false,
		},
		{
			name: "generic error",
			err:  errors.New("some error"),
			want: false,
		},
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidationError(tt.err); got != tt.want {
				t.Errorf("IsValidationError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFormatValidationError(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		wantContain []string
	}{
		{
			name: "typed bad request exception with message",
			err:  &types.BadRequestException{Message: stringPtr("JSON Schema validation failed")},
			wantContain: []string{
				"Configuration validation failed",
				"JSON Schema validation failed",
				"Possible causes",
				"JSON Schema validator",
				"Lambda validator",
			},
		},
		{
			name: "generic API error",
			err: &smithy.GenericAPIError{
				Code:    "BadRequestException",
				Message: "Lambda function returned error",
			},
			wantContain: []string{
				"Configuration validation failed",
				"Lambda function returned error",
			},
		},
		{
			name: "generic error",
			err:  errors.New("validation error"),
			wantContain: []string{
				"Configuration validation failed",
				"validation error",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatValidationError(tt.err)

			for _, want := range tt.wantContain {
				if !contains(result, want) {
					t.Errorf("FormatValidationError() = %q, want to contain %q", result, want)
				}
			}
		})
	}
}

func Test_formatUserFriendlyError(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		operation   string
		wantContain string
	}{
		{
			name:        "validation error",
			err:         &types.BadRequestException{Message: stringPtr("Schema validation failed")},
			operation:   "CreateHostedConfigurationVersion",
			wantContain: "Configuration validation failed",
		},
		{
			name: "access denied error",
			err: &smithy.GenericAPIError{
				Code:    "AccessDeniedException",
				Message: "User is not authorized",
			},
			operation:   "ListApplications",
			wantContain: "Access denied",
		},
		{
			name: "resource not found error",
			err: &smithy.GenericAPIError{
				Code:    "ResourceNotFoundException",
				Message: "Resource not found",
			},
			operation:   "GetApplication",
			wantContain: "Resource not found",
		},
		{
			name: "throttling error",
			err: &smithy.GenericAPIError{
				Code:    "ThrottlingException",
				Message: "Rate exceeded",
			},
			operation:   "ListDeployments",
			wantContain: "Rate limit exceeded",
		},
		{
			name:        "generic error",
			err:         errors.New("connection timeout"),
			operation:   "StartDeployment",
			wantContain: "connection timeout",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatUserFriendlyError(tt.err, tt.operation)

			if !contains(result, tt.wantContain) {
				t.Errorf("formatUserFriendlyError() = %q, want to contain %q", result, tt.wantContain)
			}
		})
	}
}

// Helper function
func stringPtr(s string) *string {
	return &s
}
