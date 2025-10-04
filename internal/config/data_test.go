package config

import (
	"testing"
)

func TestLoadDataFile(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "valid JSON file",
			path:    "../../testdata/data/valid.json",
			wantErr: false,
		},
		{
			name:    "valid YAML file",
			path:    "../../testdata/data/valid.yaml",
			wantErr: false,
		},
		{
			name:    "valid text file",
			path:    "../../testdata/data/valid.txt",
			wantErr: false,
		},
		{
			name:    "file not found",
			path:    "../../testdata/data/nonexistent.txt",
			wantErr: true,
		},
		{
			name:    "file too large (> 2MB)",
			path:    "../../testdata/data/toolarge.txt",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := LoadDataFile(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadDataFile() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDetermineContentType(t *testing.T) {
	tests := []struct {
		name         string
		path         string
		expectedType string
	}{
		{
			name:         "JSON file",
			path:         "test.json",
			expectedType: "application/json",
		},
		{
			name:         "YAML file with .yaml extension",
			path:         "test.yaml",
			expectedType: "application/x-yaml",
		},
		{
			name:         "YAML file with .yml extension",
			path:         "test.yml",
			expectedType: "application/x-yaml",
		},
		{
			name:         "text file",
			path:         "test.txt",
			expectedType: "text/plain",
		},
		{
			name:         "unknown extension defaults to text",
			path:         "test.conf",
			expectedType: "text/plain",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			contentType := DetermineContentType(tt.path)
			if contentType != tt.expectedType {
				t.Errorf("DetermineContentType() = %s, want %s", contentType, tt.expectedType)
			}
		})
	}
}

func TestValidateDataFile(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		contentType string
		wantErr     bool
	}{
		{
			name:        "valid JSON",
			path:        "../../testdata/data/valid.json",
			contentType: "application/json",
			wantErr:     false,
		},
		{
			name:        "valid YAML",
			path:        "../../testdata/data/valid.yaml",
			contentType: "application/x-yaml",
			wantErr:     false,
		},
		{
			name:        "invalid JSON",
			path:        "../../testdata/data/invalid.json",
			contentType: "application/json",
			wantErr:     true,
		},
		{
			name:        "invalid YAML",
			path:        "../../testdata/data/invalid.yaml",
			contentType: "application/x-yaml",
			wantErr:     true,
		},
		{
			name:        "text file (always valid)",
			path:        "../../testdata/data/valid.txt",
			contentType: "text/plain",
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := LoadDataFile(tt.path)
			if err != nil {
				t.Fatalf("LoadDataFile() error = %v", err)
			}

			err = ValidateDataFile(data, tt.contentType)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateDataFile() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCheckDataFileSize(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "small file (< 2MB)",
			path:    "../../testdata/data/valid.json",
			wantErr: false,
		},
		{
			name:    "large file (> 2MB)",
			path:    "../../testdata/data/toolarge.txt",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckDataFileSize(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("CheckDataFileSize() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
