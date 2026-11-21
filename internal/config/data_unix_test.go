//go:build unix

package config

import (
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

// TestLoadDataFileRejectsFIFO covers a data_file that is a named pipe: its
// size reads as 0, so it used to pass the size check and then block (or
// stream without limit) in the read.
func TestLoadDataFileRejectsFIFO(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "data.json")
	if err := syscall.Mkfifo(path, 0o600); err != nil {
		t.Fatalf("mkfifo: %v", err)
	}

	done := make(chan error, 1)
	go func() {
		_, err := LoadDataFile(path)
		done <- err
	}()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("expected an error for a FIFO data file, got nil")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("LoadDataFile blocked on a FIFO")
	}
}
