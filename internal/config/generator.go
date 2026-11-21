package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/goccy/go-yaml"
)

// GenerateConfigFile generates an apcdeploy.yml file with the given parameters
func GenerateConfigFile(app, profile, env, dataFile, region, deploymentStrategy, outputPath string, force bool) error {
	// Check if file already exists
	if _, err := os.Stat(outputPath); err == nil && !force {
		return fmt.Errorf("config file already exists at %s (use --force to overwrite)", outputPath)
	}

	// Use provided deployment strategy, or default to AllAtOnce if empty
	strategy := deploymentStrategy
	if strategy == "" {
		strategy = DefaultDeploymentStrategy
	}

	// Create config structure
	cfg := Config{
		Application:          app,
		ConfigurationProfile: profile,
		Environment:          env,
		DataFile:             dataFile,
		DeploymentStrategy:   strategy,
		Region:               region,
	}

	// Marshal to YAML
	data, err := yaml.Marshal(&cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// The names come from AWS and may contain characters (e.g. a lone CR)
	// that the encoder writes in a form the decoder reads differently, which
	// would let a remote name inject keys such as data_file. Refuse to write
	// anything that does not read back exactly as given.
	if decoded, err := decodeConfig(data); err != nil || decoded != cfg {
		return errors.New("resource names contain characters that cannot be written to apcdeploy.yml safely")
	}

	// Write to file
	if err := os.WriteFile(outputPath, data, 0o644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// DetermineDataFileName determines the appropriate data file name based on content type
func DetermineDataFileName(contentType string) string {
	return "data" + ExtensionForContentType(contentType)
}

// ExtensionForContentType returns the file extension (including the leading dot)
// for the given AppConfig content type. Unknown types fall back to ".json".
func ExtensionForContentType(contentType string) string {
	ct := strings.ToLower(strings.TrimSpace(contentType))
	if idx := strings.Index(ct, ";"); idx != -1 {
		ct = strings.TrimSpace(ct[:idx])
	}

	switch ct {
	case ContentTypeJSON:
		return ".json"
	case ContentTypeYAML, "application/yaml":
		return ".yaml"
	case ContentTypeText:
		return ".txt"
	default:
		return ".json"
	}
}

// WriteDataFile writes configuration data to a file with appropriate formatting
// For FeatureFlags profile type, it removes _updatedAt and _createdAt fields
func WriteDataFile(content []byte, contentType, outputPath, profileType string, force bool) error {
	// Check if file already exists
	if _, err := os.Stat(outputPath); err == nil && !force {
		return fmt.Errorf("data file already exists at %s (use --force to overwrite)", outputPath)
	}

	// Normalize content type
	ct := strings.ToLower(strings.TrimSpace(contentType))
	if idx := strings.Index(ct, ";"); idx != -1 {
		ct = strings.TrimSpace(ct[:idx])
	}

	var dataToWrite []byte
	var err error

	// Format based on content type
	switch ct {
	case ContentTypeJSON:
		// Format JSON with indentation
		dataToWrite, err = formatJSON(content, profileType)
		if err != nil {
			return fmt.Errorf("failed to format JSON: %w", err)
		}
	default:
		// Write as-is for YAML and text
		dataToWrite = content
	}

	if err := writeFileAtomic(outputPath, dataToWrite); err != nil {
		return fmt.Errorf("failed to write data file: %w", err)
	}

	return nil
}

// writeFileAtomic replaces path with data via a temporary file in the same
// directory and a rename, so an interrupted write never leaves a truncated
// data file. A symlink is followed (the link itself is kept), an existing
// file keeps its permissions, and a new file is created 0600 because the
// content comes from AWS and may hold secrets.
func writeFileAtomic(path string, data []byte) error {
	perm := os.FileMode(0o600)
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		path = resolved
		if info, err := os.Stat(path); err == nil {
			perm = info.Mode().Perm()
		}
	}

	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer func() { _ = os.Remove(tmpPath) }() // no-op once renamed

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Chmod(perm); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}

// formatJSON formats JSON data with proper indentation.
// For FeatureFlags profile type, it removes _updatedAt and _createdAt fields recursively.
// UseNumber is used during decoding so that large integers (> 2^53) are preserved
// exactly instead of being converted to float64, which would cause precision loss.
func formatJSON(data []byte, profileType string) ([]byte, error) {
	obj, err := decodeJSONUseNumber(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	// For FeatureFlags, remove _updatedAt and _createdAt fields recursively
	if profileType == ProfileTypeFeatureFlags {
		obj = RemoveTimestampFieldsRecursive(obj)
	}

	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(obj); err != nil {
		return nil, fmt.Errorf("failed to encode JSON: %w", err)
	}

	return buf.Bytes(), nil
}
