package aws

import (
	"fmt"
)

// resolveByName resolves a resource by name using a generic approach
// It returns an error if no matches or multiple matches are found
// The nameGetter and idGetter functions extract the Name and Id fields from each item
func resolveByName[T any](
	items []T,
	name string,
	resourceType string,
	nameGetter func(T) *string,
	idGetter func(T) *string,
) (string, error) {
	var matches []string
	for _, item := range items {
		itemName := nameGetter(item)
		if itemName != nil && *itemName == name {
			itemID := idGetter(item)
			if itemID != nil {
				matches = append(matches, *itemID)
			}
		}
	}

	if len(matches) == 0 {
		return "", fmt.Errorf("%s not found: %s", resourceType, name)
	}

	if len(matches) > 1 {
		return "", fmt.Errorf("multiple %ss found with name: %s", resourceType, name)
	}

	return matches[0], nil
}
