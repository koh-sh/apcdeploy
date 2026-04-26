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

func Test_checkDataFileSize(t *testing.T) {
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
			err := checkDataFileSize(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("checkDataFileSize() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
