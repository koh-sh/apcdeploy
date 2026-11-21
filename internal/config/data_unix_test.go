//go:build unix

package config

import (
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

// TestLoadDataFileRejectsSpecialFiles covers a data_file that is not a
// regular file: its size reads as 0, so it used to pass the size check and
// then block (FIFO) or stream without limit (device) in the read.
func TestLoadDataFileRejectsSpecialFiles(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		path func(t *testing.T) string
	}{
		{
			name: "named pipe",
			path: func(t *testing.T) string {
				p := filepath.Join(t.TempDir(), "data.json")
				if err := syscall.Mkfifo(p, 0o600); err != nil {
					t.Fatalf("mkfifo: %v", err)
				}
				return p
			},
		},
		{name: "character device", path: func(*testing.T) string { return "/dev/zero" }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			path := tt.path(t)
			done := make(chan error, 1)
			go func() {
				_, err := LoadDataFile(path)
				done <- err
			}()
			select {
			case err := <-done:
				if err == nil {
					t.Fatal("expected an error, got nil")
				}
			case <-time.After(5 * time.Second):
				t.Fatal("LoadDataFile blocked")
			}
		})
	}
}
