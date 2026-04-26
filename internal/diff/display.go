package diff

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/koh-sh/apcdeploy/internal/aws"
	"github.com/koh-sh/apcdeploy/internal/cli"
	"github.com/koh-sh/apcdeploy/internal/config"
	"github.com/koh-sh/apcdeploy/internal/reporter"
)

// inProgressWarningSink is the writer used by the "deployment in progress"
// notice. It is a package-level variable so tests can intercept it; in
// production it is always os.Stderr.
var inProgressWarningSink io.Writer = os.Stderr

// display renders the diff result through the Reporter.
//
// The unified diff itself goes to stdout via Reporter.Diff (always shown,
// even under --silent). The header / metadata / summary lines go through
// Reporter primitives so silent mode suppresses them automatically — callers
// MUST NOT branch on opts.Silent.
func display(r reporter.Reporter, result *Result, cfg *config.Config, resources *aws.ResolvedResources, deployment *aws.DeploymentInfo) {
	r.Header("Configuration Diff")

	metaRows := [][]string{
		{"Application", cfg.Application},
		{"Profile", resources.Profile.Name},
		{"Environment", cfg.Environment},
	}
	if deployment != nil {
		metaRows = append(metaRows, []string{
			"Remote Version", fmt.Sprintf("%s (Deployment #%d)", deployment.ConfigurationVersion, deployment.DeploymentNumber),
		})
		if deployment.State != "" {
			metaRows = append(metaRows, []string{"Status", cli.StateBadge(string(deployment.State))})
		}
	} else {
		metaRows = append(metaRows, []string{"Remote Version", "(none)"})
	}
	metaRows = append(metaRows, []string{"Local File", result.FileName})
	r.Table([]string{"Field", "Value"}, metaRows)

	if !result.HasChanges {
		r.Success("No changes detected")
		displayDeploymentWarning(deployment)
		return
	}

	// Diff payload to stdout (machine-readable; always shown).
	r.Diff([]byte(ensureTrailingNewline(result.UnifiedDiff)))

	added, removed := countChanges(result.UnifiedDiff)
	r.Info(fmt.Sprintf("Summary: +%d additions, -%d deletions", added, removed))

	displayDeploymentWarning(deployment)
}

// displayDeploymentWarning surfaces a notice when the latest deployment is
// still in progress, since the diff is taken against an in-flight version.
//
// CONTRACT EXCEPTION (see .claude/rules/output-contract.md "diff in-progress
// warning"): this writes directly to stderr instead of going through
// Reporter.Warn so the notice still reaches scripts under --silent. An
// in-flight deployment can be rolled back mid-rollout and change what the
// diff is taken against, so users in automated pipelines must still see this
// risk.
func displayDeploymentWarning(deployment *aws.DeploymentInfo) {
	if deployment == nil {
		return
	}
	state := string(deployment.State)
	if state != "DEPLOYING" && state != "BAKING" {
		return
	}
	fmt.Fprintln(inProgressWarningSink)
	fmt.Fprintf(inProgressWarningSink, "⚠ Deployment #%d is currently %s\n", deployment.DeploymentNumber, state)
	fmt.Fprintln(inProgressWarningSink, "The diff is calculated against the currently deploying version.")
}

// ensureTrailingNewline guarantees the diff payload ends with a newline so
// piped consumers see clean line breaks.
func ensureTrailingNewline(s string) string {
	if s == "" || strings.HasSuffix(s, "\n") {
		return s
	}
	return s + "\n"
}

// countChanges counts the number of added and removed lines in a unified diff.
func countChanges(diff string) (added int, removed int) {
	for line := range strings.SplitSeq(diff, "\n") {
		switch {
		case strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++"):
			added++
		case strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---"):
			removed++
		}
	}
	return added, removed
}
