package diff

import (
	"fmt"
	"strings"

	"github.com/koh-sh/apcdeploy/internal/aws"
	"github.com/koh-sh/apcdeploy/internal/config"
)

// display shows the diff result in a user-friendly format
func display(result *Result, cfg *config.Config, resources *aws.ResolvedResources, deployment *aws.DeploymentInfo) {
	// Display header
	fmt.Println("Configuration Diff")
	fmt.Println("==================")
	fmt.Println()
	fmt.Printf("Application:   %s\n", cfg.Application)
	fmt.Printf("Profile:       %s\n", resources.Profile.Name)
	fmt.Printf("Environment:   %s\n", cfg.Environment)
	fmt.Println()

	if deployment != nil {
		fmt.Printf("Remote Version: %s (Deployment #%d)\n", deployment.ConfigurationVersion, deployment.DeploymentNumber)
		if deployment.State != "" {
			fmt.Printf("Status:         %s\n", deployment.State)
		}
	} else {
		fmt.Println("Remote Version: (none)")
	}
	fmt.Printf("Local File:     %s\n", result.FileName)
	fmt.Println()

	// Check if there are changes
	if !result.HasChanges {
		fmt.Println("✓ No changes detected")
		return
	}

	// Display the diff
	fmt.Println("Changes:")
	fmt.Println("--------")
	displayColorizedDiff(result.UnifiedDiff)
	fmt.Println()

	// Display summary
	addedLines, removedLines := countChanges(result.UnifiedDiff)
	fmt.Printf("Summary: +%d additions, -%d deletions\n", addedLines, removedLines)
}

// displayColorizedDiff displays the diff with colors
func displayColorizedDiff(diff string) {
	lines := strings.SplitSeq(diff, "\n")
	for line := range lines {
		if len(line) == 0 {
			continue
		}

		switch {
		case strings.HasPrefix(line, "+"):
			// Green for additions
			fmt.Printf("\033[32m%s\033[0m\n", line)
		case strings.HasPrefix(line, "-"):
			// Red for deletions
			fmt.Printf("\033[31m%s\033[0m\n", line)
		case strings.HasPrefix(line, "@"):
			// Cyan for diff headers
			fmt.Printf("\033[36m%s\033[0m\n", line)
		default:
			// Normal for context lines
			fmt.Println(line)
		}
	}
}

// countChanges counts the number of added and removed lines
func countChanges(diff string) (added int, removed int) {
	lines := strings.SplitSeq(diff, "\n")
	for line := range lines {
		switch {
		case strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++"):
			added++
		case strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---"):
			removed++
		}
	}
	return added, removed
}
