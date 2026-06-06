package config

import (
	"path/filepath"
	"strings"
)

// DetermineContentType returns the AppConfig content type for the given profile
// type and data file path.
//
// FeatureFlags configurations always use JSON. Freeform configurations are
// inferred from the file extension, defaulting to text/plain for unknown
// extensions.
func DetermineContentType(profileType, dataPath string) string {
	if profileType == ProfileTypeFeatureFlags {
		return ContentTypeJSON
	}

	switch strings.ToLower(filepath.Ext(dataPath)) {
	case ".json":
		return ContentTypeJSON
	case ".yaml", ".yml":
		return ContentTypeYAML
	default:
		// .txt and any unknown extension fall back to text/plain.
		return ContentTypeText
	}
}
