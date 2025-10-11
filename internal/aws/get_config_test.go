package aws

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/appconfigdata"
	"github.com/koh-sh/apcdeploy/internal/aws/mock"
)

func TestGetConfiguration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		applicationID     string
		environmentID     string
		profileID         string
		sessionMock       func(ctx context.Context, params *appconfigdata.StartConfigurationSessionInput, optFns ...func(*appconfigdata.Options)) (*appconfigdata.StartConfigurationSessionOutput, error)
		configMock        func(ctx context.Context, params *appconfigdata.GetLatestConfigurationInput, optFns ...func(*appconfigdata.Options)) (*appconfigdata.GetLatestConfigurationOutput, error)
		expectedContent   []byte
		expectError       bool
		expectedErrorText string
	}{
		{
			name:          "successful get configuration",
			applicationID: "app-123",
			environmentID: "env-123",
			profileID:     "profile-123",
			sessionMock: func(ctx context.Context, params *appconfigdata.StartConfigurationSessionInput, optFns ...func(*appconfigdata.Options)) (*appconfigdata.StartConfigurationSessionOutput, error) {
				return &appconfigdata.StartConfigurationSessionOutput{
					InitialConfigurationToken: aws.String("initial-token"),
				}, nil
			},
			configMock: func(ctx context.Context, params *appconfigdata.GetLatestConfigurationInput, optFns ...func(*appconfigdata.Options)) (*appconfigdata.GetLatestConfigurationOutput, error) {
				return &appconfigdata.GetLatestConfigurationOutput{
					Configuration: []byte(`{"key": "value"}`),
				}, nil
			},
			expectedContent: []byte(`{"key": "value"}`),
			expectError:     false,
		},
		{
			name:          "start session error",
			applicationID: "app-123",
			environmentID: "env-123",
			profileID:     "profile-123",
			sessionMock: func(ctx context.Context, params *appconfigdata.StartConfigurationSessionInput, optFns ...func(*appconfigdata.Options)) (*appconfigdata.StartConfigurationSessionOutput, error) {
				return nil, errors.New("session error")
			},
			configMock:        nil,
			expectError:       true,
			expectedErrorText: "failed to start configuration session",
		},
		{
			name:          "get configuration error",
			applicationID: "app-123",
			environmentID: "env-123",
			profileID:     "profile-123",
			sessionMock: func(ctx context.Context, params *appconfigdata.StartConfigurationSessionInput, optFns ...func(*appconfigdata.Options)) (*appconfigdata.StartConfigurationSessionOutput, error) {
				return &appconfigdata.StartConfigurationSessionOutput{
					InitialConfigurationToken: aws.String("initial-token"),
				}, nil
			},
			configMock: func(ctx context.Context, params *appconfigdata.GetLatestConfigurationInput, optFns ...func(*appconfigdata.Options)) (*appconfigdata.GetLatestConfigurationOutput, error) {
				return nil, errors.New("config error")
			},
			expectError:       true,
			expectedErrorText: "failed to get latest configuration",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockAppConfigData := &mock.MockAppConfigDataClient{
				StartConfigurationSessionFunc: tt.sessionMock,
				GetLatestConfigurationFunc:    tt.configMock,
			}

			client := &Client{
				AppConfigData: mockAppConfigData,
			}

			content, err := client.GetConfiguration(context.Background(), tt.applicationID, tt.environmentID, tt.profileID)

			if tt.expectError {
				if err == nil {
					t.Fatal("expected error but got nil")
				}
				if tt.expectedErrorText != "" && !strings.Contains(err.Error(), tt.expectedErrorText) {
					t.Errorf("expected error to contain %q, got: %v", tt.expectedErrorText, err)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if string(content) != string(tt.expectedContent) {
					t.Errorf("expected content %q, got %q", tt.expectedContent, content)
				}
			}
		})
	}
}
