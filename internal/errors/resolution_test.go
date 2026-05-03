package errors

import (
	"errors"
	"fmt"
	"testing"

	"github.com/aws/smithy-go"
)

func TestResolution(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "nil error returns empty hint",
			err:  nil,
			want: "",
		},
		{
			name: "non-AWS error returns empty hint",
			err:  errors.New("plain error"),
			want: "",
		},
		{
			name: "ConflictException maps to wait/rollback hint",
			err: &smithy.GenericAPIError{
				Code:    "ConflictException",
				Message: "There is already a deployment in progress",
			},
			want: "wait for the current deployment to complete or run 'apcdeploy rollback'.",
		},
		{
			name: "BadRequestException maps to validator hint",
			err: &smithy.GenericAPIError{
				Code:    "BadRequestException",
				Message: "validation failed",
			},
			want: "check your configuration data, JSON/YAML syntax, and any configured validators (JSON Schema / Lambda).",
		},
		{
			name: "ResourceNotFoundException maps to ls-resources hint",
			err: &smithy.GenericAPIError{
				Code:    "ResourceNotFoundException",
				Message: "application not found",
			},
			want: "verify the resource names with 'apcdeploy ls-resources' and your AWS region.",
		},
		{
			name: "unknown AWS error code returns empty hint",
			err: &smithy.GenericAPIError{
				Code:    "ThrottlingException",
				Message: "rate exceeded",
			},
			want: "",
		},
		{
			name: "wrapped ConflictException is still resolved",
			err:  fmt.Errorf("StartDeployment failed: %w", &smithy.GenericAPIError{Code: "ConflictException"}),
			want: "wait for the current deployment to complete or run 'apcdeploy rollback'.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := Resolution(tt.err); got != tt.want {
				t.Errorf("Resolution() = %q, want %q", got, tt.want)
			}
		})
	}
}
