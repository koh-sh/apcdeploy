package diff

import (
	"fmt"
	"os"
	"strings"

	"github.com/koh-sh/apcdeploy/internal/aws"
	"github.com/koh-sh/apcdeploy/internal/config"
)

// displaySilent shows only the diff content in silent mode
func displaySilent(result *Result, deployment *aws.DeploymentInfo) {
	// In silent mode:
	// - No output if there are no changes
	// - Only show the diff content if there are changes
	if result.HasChanges {
		displayColorizedDiff(result.UnifiedDiff)
	}

	// Show deployment warning in silent mode too if deployment is in progress.
	// This is intentional because an ongoing deployment that gets rolled back
	// could change the diff result, and users should be aware of this risk
	// even in silent/automated environments.
	displayDeploymentWarning(deployment)
}

// display shows the diff result in a user-friendly format
func display(result *Result, cfg *config.Config, resources *aws.ResolvedResources, deployment *aws.DeploymentInfo) {
	// Display header to stderr (metadata)
	fmt.Fprintln(os.Stderr, "Configuration Diff")
	fmt.Fprintln(os.Stderr, "==================")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintf(os.Stderr, "Application:   %s\n", cfg.Application)
	fmt.Fprintf(os.Stderr, "Profile:       %s\n", resources.Profile.Name)
	fmt.Fprintf(os.Stderr, "Environment:   %s\n", cfg.Environment)
	fmt.Fprintln(os.Stderr)

	if deployment != nil {
		fmt.Fprintf(os.Stderr, "Remote Version: %s (Deployment #%d)\n", deployment.ConfigurationVersion, deployment.DeploymentNumber)
		if deployment.State != "" {
			fmt.Fprintf(os.Stderr, "Status:         %s\n", deployment.State)
		}
	} else {
		fmt.Fprintln(os.Stderr, "Remote Version: (none)")
	}
	fmt.Fprintf(os.Stderr, "Local File:     %s\n", result.FileName)
	fmt.Fprintln(os.Stderr)

	// Check if there are changes
	if !result.HasChanges {
		fmt.Fprintln(os.Stderr, "✓ No changes detected")
		// Show deployment warning even if no changes
		displayDeploymentWarning(deployment)
		return
	}

	// Display the diff header to stderr
	fmt.Fprintln(os.Stderr, "Changes:")
	fmt.Fprintln(os.Stderr, "--------")
	// Display the actual diff to stdout (machine-readable)
	displayColorizedDiff(result.UnifiedDiff)
	fmt.Fprintln(os.Stderr)

	// Display summary to stderr (metadata)
	addedLines, removedLines := countChanges(result.UnifiedDiff)
	fmt.Fprintf(os.Stderr, "Summary: +%d additions, -%d deletions\n", addedLines, removedLines)

	// Show deployment warning after summary
	displayDeploymentWarning(deployment)
}

// displayDeploymentWarning shows a warning if deployment is in progress
func displayDeploymentWarning(deployment *aws.DeploymentInfo) {
	if deployment != nil && (deployment.State == "DEPLOYING" || deployment.State == "BAKING") {
		fmt.Fprintln(os.Stderr)
		fmt.Fprintf(os.Stderr, "⚠ Deployment #%d is currently %s\n", deployment.DeploymentNumber, deployment.State)
		fmt.Fprintln(os.Stderr, "The diff is calculated against the currently deploying version.")
	}
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
