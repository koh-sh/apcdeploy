package lsresources

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// FormatJSON formats the resources tree as JSON
func FormatJSON(tree *ResourcesTree, w io.Writer, showStrategies bool) error {
	if !showStrategies {
		treeCopy := *tree
		treeCopy.DeploymentStrategies = nil
		tree = &treeCopy
	}
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(tree)
}

// FormatHumanReadable formats the resources tree in a human-readable format
func FormatHumanReadable(tree *ResourcesTree, w io.Writer, showStrategies bool) error {
	var sb strings.Builder

	// Header
	sb.WriteString(fmt.Sprintf("Region: %s\n\n", tree.Region))

	// Deployment Strategies section
	if showStrategies {
		sb.WriteString("Deployment Strategies:\n")
		if len(tree.DeploymentStrategies) == 0 {
			sb.WriteString("  No deployment strategies found.\n")
		} else {
			for _, strategy := range tree.DeploymentStrategies {
				sb.WriteString(fmt.Sprintf("  - %s (ID: %s)\n", strategy.Name, strategy.ID))
				if strategy.Description != "" {
					sb.WriteString(fmt.Sprintf("    Description: %s\n", strategy.Description))
				}
				sb.WriteString(fmt.Sprintf("    Deployment Duration: %d minutes\n", strategy.DeploymentDurationInMinutes))
				sb.WriteString(fmt.Sprintf("    Final Bake Time: %d minutes\n", strategy.FinalBakeTimeInMinutes))
				sb.WriteString(fmt.Sprintf("    Growth Factor: %.1f%%\n", strategy.GrowthFactor))
				if strategy.GrowthType != "" {
					sb.WriteString(fmt.Sprintf("    Growth Type: %s\n", strategy.GrowthType))
				}
			}
		}
		sb.WriteString("\n")
	}

	// Check if there are any applications
	if len(tree.Applications) == 0 {
		sb.WriteString("Applications:\n")
		sb.WriteString("  No applications found.\n")
		_, err := w.Write([]byte(sb.String()))
		return err
	}

	// Applications section
	sb.WriteString("Applications:\n")
	for i, app := range tree.Applications {
		// Application header
		sb.WriteString(fmt.Sprintf("  [%d] %s (ID: %s)\n", i+1, app.Name, app.ID))

		// Configuration Profiles
		sb.WriteString("      Configuration Profiles:\n")
		if len(app.Profiles) == 0 {
			sb.WriteString("        - No configuration profiles\n")
		} else {
			for _, profile := range app.Profiles {
				sb.WriteString(fmt.Sprintf("        - %s (ID: %s)\n", profile.Name, profile.ID))
			}
		}

		// Environments
		sb.WriteString("      Environments:\n")
		if len(app.Environments) == 0 {
			sb.WriteString("        - No environments\n")
		} else {
			for _, env := range app.Environments {
				sb.WriteString(fmt.Sprintf("        - %s (ID: %s)\n", env.Name, env.ID))
			}
		}

		// Add spacing between applications
		if i < len(tree.Applications)-1 {
			sb.WriteString("\n")
		}
	}

	_, err := w.Write([]byte(sb.String()))
	return err
}
