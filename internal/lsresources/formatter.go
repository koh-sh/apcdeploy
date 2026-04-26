package lsresources

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/koh-sh/apcdeploy/internal/cli"
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

// FormatHumanReadable formats the resources tree in a human-readable format.
// When tty is true, lipgloss styles are applied to highlight names and dim
// secondary metadata (IDs); when false, the output is plain text suitable for
// piped/CI consumption.
func FormatHumanReadable(tree *ResourcesTree, w io.Writer, showStrategies, tty bool) error {
	heading := func(s string) string { return s }
	subtle := func(s string) string { return s }
	if tty {
		heading = cli.HeadingText
		subtle = cli.SubtleText
	}

	var sb strings.Builder

	fmt.Fprintf(&sb, "Region: %s\n\n", heading(tree.Region))

	if showStrategies {
		sb.WriteString(heading("Deployment Strategies:") + "\n")
		if len(tree.DeploymentStrategies) == 0 {
			sb.WriteString("  No deployment strategies found.\n")
		} else {
			for _, strategy := range tree.DeploymentStrategies {
				fmt.Fprintf(&sb, "  - %s %s\n", heading(strategy.Name), subtle("(ID: "+strategy.ID+")"))
				if strategy.Description != "" {
					fmt.Fprintf(&sb, "    %s %s\n", subtle("Description:"), strategy.Description)
				}
				fmt.Fprintf(&sb, "    %s %d minutes\n", subtle("Deployment Duration:"), strategy.DeploymentDurationInMinutes)
				fmt.Fprintf(&sb, "    %s %d minutes\n", subtle("Final Bake Time:"), strategy.FinalBakeTimeInMinutes)
				fmt.Fprintf(&sb, "    %s %.1f%%\n", subtle("Growth Factor:"), strategy.GrowthFactor)
				if strategy.GrowthType != "" {
					fmt.Fprintf(&sb, "    %s %s\n", subtle("Growth Type:"), strategy.GrowthType)
				}
			}
		}
		sb.WriteString("\n")
	}

	if len(tree.Applications) == 0 {
		sb.WriteString(heading("Applications:") + "\n")
		sb.WriteString("  No applications found.\n")
		_, err := w.Write([]byte(sb.String()))
		return err
	}

	sb.WriteString(heading("Applications:") + "\n")
	for i, app := range tree.Applications {
		fmt.Fprintf(&sb, "  [%d] %s %s\n", i+1, heading(app.Name), subtle("(ID: "+app.ID+")"))

		fmt.Fprintf(&sb, "      %s\n", subtle("Configuration Profiles:"))
		if len(app.Profiles) == 0 {
			sb.WriteString("        - No configuration profiles\n")
		} else {
			for _, profile := range app.Profiles {
				fmt.Fprintf(&sb, "        - %s %s\n", profile.Name, subtle("(ID: "+profile.ID+")"))
			}
		}

		fmt.Fprintf(&sb, "      %s\n", subtle("Environments:"))
		if len(app.Environments) == 0 {
			sb.WriteString("        - No environments\n")
		} else {
			for _, env := range app.Environments {
				fmt.Fprintf(&sb, "        - %s %s\n", env.Name, subtle("(ID: "+env.ID+")"))
			}
		}

		if i < len(tree.Applications)-1 {
			sb.WriteString("\n")
		}
	}

	_, err := w.Write([]byte(sb.String()))
	return err
}
