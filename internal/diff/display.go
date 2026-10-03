package diff

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/koh-sh/apcdeploy/internal/aws"
	"github.com/koh-sh/apcdeploy/internal/batch"
)

// inProgressWarningSink returns the writer used by the "deployment in progress"
// notice. It is a package-level variable so tests can intercept it; in
// production it is always os.Stderr. It resolves os.Stderr at call time
// (rather than capturing it at init) so cmd-level tests that swap
// os.Stderr also capture the warning.
var inProgressWarningSink = func() io.Writer { return os.Stderr }

// formatDiffSummary renders the post-icon Targets summary for a diff with
// changes. The wording is "diff (N lines changed)" augmented with the
// +/- breakdown so users get the deletion/addition split without
// scrolling through the patch.
func formatDiffSummary(added, removed int) string {
	total := added + removed
	noun := "lines"
	if total == 1 {
		noun = "line"
	}
	return fmt.Sprintf("diff (%d %s changed: +%d -%d)", total, noun, added, removed)
}

// deploymentWarning returns the in-progress notice for the latest
// deployment, or "" when the deployment is absent or no longer in flight.
// RunOnTarget only computes the text; it is written later by
// WriteDeploymentWarnings once the Targets block is closed.
func deploymentWarning(deployment *aws.DeploymentInfo) string {
	if deployment == nil {
		return ""
	}
	state := string(deployment.State)
	if state != "DEPLOYING" && state != "BAKING" {
		return ""
	}
	return fmt.Sprintf("Deployment #%d is currently %s", deployment.DeploymentNumber, state)
}

// WriteDeploymentWarnings writes the in-progress notices collected by
// RunOnTarget, one block per target in argument order. warnings is
// aligned with targets (batch.PayloadCollector.Warnings); empty slots are
// skipped. Each block is prefixed with the target identifier because it is
// no longer printed next to its Targets row.
//
// It MUST be called only after batch.Orchestrator.Run has returned: the
// TTY Targets renderer assumes the cursor sits directly below its block, so
// any raw write while the block is open gets overwritten by the next redraw
// (issue #151). Calling it from a single goroutine after the run also keeps
// warnings from different targets from interleaving line by line.
//
// CONTRACT EXCEPTION (see .claude/rules/output-contract.md "diff in-progress
// warning"): this writes directly to stderr instead of going through
// Reporter.Warn so the notice still reaches scripts under --silent. An
// in-flight deployment can be rolled back mid-rollout and change what the
// diff is taken against, so users in automated pipelines must still see this
// risk.
func WriteDeploymentWarnings(targets []*batch.Target, warnings []string) {
	var b strings.Builder
	for i, t := range targets {
		if i >= len(warnings) || warnings[i] == "" {
			continue
		}
		fmt.Fprintf(&b, "\n⚠ %s: %s\n", t.Identifier, warnings[i])
		b.WriteString("The diff is calculated against the currently deploying version.\n")
	}
	if b.Len() == 0 {
		return
	}
	_, _ = io.WriteString(inProgressWarningSink(), b.String())
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
