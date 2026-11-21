package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
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
		return fmt.Errorf("%s contains characters that cannot be written to apcdeploy.yml safely", unsafeConfigField(cfg))
	}

	// Write to file
	if err := os.WriteFile(outputPath, data, 0o644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// unsafeConfigField returns the YAML key of the first field whose value does
// not survive a YAML round trip on its own, so the GenerateConfigFile error
// can name it.
func unsafeConfigField(cfg Config) string {
	for _, f := range []struct{ key, value string }{
		{"application", cfg.Application},
		{"configuration_profile", cfg.ConfigurationProfile},
		{"environment", cfg.Environment},
		{"deployment_strategy", cfg.DeploymentStrategy},
		{"data_file", cfg.DataFile},
		{"region", cfg.Region},
	} {
		data, err := yaml.Marshal(map[string]string{f.key: f.value})
		var back map[string]string
		if err != nil || yaml.Unmarshal(data, &back) != nil || back[f.key] != f.value {
			return f.key
		}
	}
	return "a resource name"
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

	// A new file is created 0600 because the content comes from AWS and may
	// hold secrets; an existing file (or a symlink's target) keeps its mode,
	// and a read-only one is not overwritten.
	if err := os.WriteFile(outputPath, dataToWrite, 0o600); err != nil {
		return fmt.Errorf("failed to write data file: %w", err)
	}

	return nil
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
