package aws

import (
	"errors"
	"strings"
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
			if !strings.Contains(errMsg, tt.wantContain) {
				t.Errorf("wrapAWSError() error message = %q, want to contain %q", errMsg, tt.wantContain)
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
			err:  &types.BadRequestException{Message: new("Invalid configuration")},
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
			err:  &types.BadRequestException{Message: new("JSON Schema validation failed")},
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
				if !strings.Contains(result, want) {
					t.Errorf("FormatValidationError() = %q, want to contain %q", result, want)
				}
			}
		})
	}
}
