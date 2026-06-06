package aws

import (
	"testing"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/appconfig/types"
)

func TestProfileInfoJSONSchemaContent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		validators  []types.Validator
		wantContent string
		wantOK      bool
	}{
		{
			name: "json schema validator present",
			validators: []types.Validator{
				{Type: types.ValidatorTypeJsonSchema, Content: awssdk.String(`{"type":"object"}`)},
			},
			wantContent: `{"type":"object"}`,
			wantOK:      true,
		},
		{
			name: "lambda validator ignored",
			validators: []types.Validator{
				{Type: types.ValidatorTypeLambda, Content: awssdk.String("arn:aws:lambda:...")},
			},
			wantOK: false,
		},
		{
			name: "json schema among lambda",
			validators: []types.Validator{
				{Type: types.ValidatorTypeLambda, Content: awssdk.String("arn:...")},
				{Type: types.ValidatorTypeJsonSchema, Content: awssdk.String(`{"a":1}`)},
			},
			wantContent: `{"a":1}`,
			wantOK:      true,
		},
		{
			name:       "no validators",
			validators: nil,
			wantOK:     false,
		},
		{
			name: "json schema with nil content",
			validators: []types.Validator{
				{Type: types.ValidatorTypeJsonSchema, Content: nil},
			},
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			p := &ProfileInfo{Validators: tt.validators}
			content, ok := p.JSONSchemaContent()
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}
			if content != tt.wantContent {
				t.Fatalf("content = %q, want %q", content, tt.wantContent)
			}
		})
	}
}

func TestProfileInfoHasLambdaValidator(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		validators []types.Validator
		want       bool
	}{
		{
			name:       "lambda present",
			validators: []types.Validator{{Type: types.ValidatorTypeLambda, Content: awssdk.String("arn:aws:lambda:...")}},
			want:       true,
		},
		{
			name:       "json schema only",
			validators: []types.Validator{{Type: types.ValidatorTypeJsonSchema, Content: awssdk.String("{}")}},
			want:       false,
		},
		{
			name: "both present",
			validators: []types.Validator{
				{Type: types.ValidatorTypeJsonSchema, Content: awssdk.String("{}")},
				{Type: types.ValidatorTypeLambda, Content: awssdk.String("arn:...")},
			},
			want: true,
		},
		{
			name:       "no validators",
			validators: nil,
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			p := &ProfileInfo{Validators: tt.validators}
			if got := p.HasLambdaValidator(); got != tt.want {
				t.Errorf("HasLambdaValidator() = %v, want %v", got, tt.want)
			}
		})
	}
}
