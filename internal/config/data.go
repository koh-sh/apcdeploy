package config

import (
	"fmt"
	"io"
	"os"
)

// LoadDataFile reads a configuration data file of at most MaxConfigSize bytes.
//
// Only regular files are accepted: a FIFO or device reports size 0, so it
// would pass the size check and then block or stream without limit. The
// check runs before opening (opening a FIFO blocks) and again on the opened
// descriptor, and the read itself is capped so a file that grows after the
// check cannot exceed the limit either.
func LoadDataFile(path string) ([]byte, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}
	if err := checkDataFileInfo(info); err != nil {
		return nil, err
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read data file: %w", err)
	}
	defer f.Close()

	info, err = f.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}
	if err := checkDataFileInfo(info); err != nil {
		return nil, err
	}

	data, err := io.ReadAll(io.LimitReader(f, MaxConfigSize+1))
	if err != nil {
		return nil, fmt.Errorf("failed to read data file: %w", err)
	}
	if len(data) > MaxConfigSize {
		return nil, fmt.Errorf("file size exceeds maximum allowed size (%d bytes)", MaxConfigSize)
	}
	return data, nil
}

// checkDataFileInfo rejects anything but a regular file within the size limit.
func checkDataFileInfo(info os.FileInfo) error {
	if !info.Mode().IsRegular() {
		return fmt.Errorf("data file is not a regular file (mode %s)", info.Mode().Type())
	}
	if info.Size() > MaxConfigSize {
		return fmt.Errorf("file size (%d bytes) exceeds maximum allowed size (%d bytes)", info.Size(), MaxConfigSize)
	}
	return nil
}
