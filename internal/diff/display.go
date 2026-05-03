package diff

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/koh-sh/apcdeploy/internal/aws"
	"github.com/koh-sh/apcdeploy/internal/reporter"
)

// inProgressWarningSink is the writer used by the "deployment in progress"
// notice. It is a package-level variable so tests can intercept it; in
// production it is always os.Stderr.
var inProgressWarningSink io.Writer = os.Stderr

// display finalises the Targets row for id with either "diff (N lines
// changed)" or "no changes", emits the unified diff to stdout when changes
// exist, and surfaces the in-progress warning when the latest deployment is
// still rolling out.
//
// For N=1 (single -c) callers, the unified diff body is emitted without a
// `=== <id> ===` header so it can be piped straight into patch/git apply
// (output.md §7.2 stdout header rules).
func display(r reporter.Reporter, tg reporter.Targets, id string, result *Result, deployment *aws.DeploymentInfo) {
	if !result.HasChanges {
		tg.Done(id, "no changes")
		displayDeploymentWarning(deployment)
		return
	}

	r.Diff([]byte(ensureTrailingNewline(result.UnifiedDiff)))
	added, removed := countChanges(result.UnifiedDiff)
	tg.Done(id, formatDiffSummary(added, removed))
	displayDeploymentWarning(deployment)
}

// formatDiffSummary renders the post-icon Targets summary for a diff with
// changes. The wording matches output.md §7.2 (a) — "diff (N lines changed)"
// — but augments it with the +/- breakdown so users get the deletion/addition
// split without scrolling through the patch.
func formatDiffSummary(added, removed int) string {
	total := added + removed
	noun := "lines"
	if total == 1 {
		noun = "line"
	}
	return fmt.Sprintf("diff (%d %s changed: +%d -%d)", total, noun, added, removed)
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
