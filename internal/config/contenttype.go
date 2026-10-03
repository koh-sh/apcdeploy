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

// NormalizationExtension returns the extension that selects the normalizer
// (see NormalizeByExtension) for the given profile type and data file path.
//
// It is derived from the resolved content type rather than the raw file
// extension, so FeatureFlags data is always normalized as JSON (and has its
// _updatedAt / _createdAt timestamps stripped) even when the data file does
// not end in ".json". Freeform keeps extension-driven normalization.
func NormalizationExtension(profileType, dataPath string) string {
	return ExtensionForContentType(DetermineContentType(profileType, dataPath))
}
