package aws

import (
	"context"
	"os"
	"testing"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name        string
		region      string
		setupEnv    map[string]string
		wantErr     bool
		errContains string
	}{
		{
			name:    "successful client creation with region",
			region:  "us-east-1",
			wantErr: false,
		},
		{
			name:   "successful client creation with AWS_REGION env",
			region: "",
			setupEnv: map[string]string{
				"AWS_REGION": "ap-northeast-1",
			},
			wantErr: false,
		},
		{
			name:   "successful client creation with AWS_DEFAULT_REGION env",
			region: "",
			setupEnv: map[string]string{
				"AWS_DEFAULT_REGION": "eu-west-1",
			},
			wantErr: false,
		},
		{
			name:    "error when no region specified",
			region:  "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup environment variables
			if tt.setupEnv != nil {
				for k, v := range tt.setupEnv {
					os.Setenv(k, v)
					defer os.Unsetenv(k)
				}
			} else {
				// Clear region env vars for clean test
				os.Unsetenv("AWS_REGION")
				os.Unsetenv("AWS_DEFAULT_REGION")
			}

			ctx := context.Background()
			client, err := NewClient(ctx, tt.region)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				if tt.errContains != "" && err != nil {
					// Check error message contains expected string
					if !contains(err.Error(), tt.errContains) {
						t.Errorf("error = %v, want to contain %v", err, tt.errContains)
					}
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if client == nil {
				t.Error("expected client, got nil")
			}
		})
	}
}

func TestNewClientWithProfile(t *testing.T) {
	tests := []struct {
		name     string
		region   string
		setupEnv map[string]string
		wantErr  bool
	}{
		{
			name:   "client creation with non-existent AWS_PROFILE env should error",
			region: "us-east-1",
			setupEnv: map[string]string{
				"AWS_PROFILE": "test-profile",
			},
			wantErr: true, // Profile doesn't exist, so it should error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup environment variables
			if tt.setupEnv != nil {
				for k, v := range tt.setupEnv {
					os.Setenv(k, v)
					defer os.Unsetenv(k)
				}
			}

			ctx := context.Background()
			_, err := NewClient(ctx, tt.region)

			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}

			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}
