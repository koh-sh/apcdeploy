package cmd

import (
	"fmt"
	"os"

	"github.com/koh-sh/apcdeploy/internal/apcerrors"
	"github.com/koh-sh/apcdeploy/internal/batch"
	"github.com/koh-sh/apcdeploy/internal/cli"
)

// summaryConfig describes how to render a batch.Summary line for one
// command. The aggregate format is:
//
//	N ok, N <noopVerb>, N failed [(elapsed)]
//
// withElapsed controls whether the trailing "(elapsed)" suffix appears.
// Only wait-style commands (run with --wait-deploy / --wait-bake) include
// it; diff and pull omit it because their elapsed time is dominated by
// network I/O and would be misleading as a metric.
type summaryConfig struct {
	noopVerb    string
	withElapsed bool
}

// renderBatchSummary writes the aggregate summary line plus, when there
// were failures, the per-target Errors: section. The summary is only
// shown when N >= 2 (single-target output already terminates in the
// row's Done/Skip/Fail line; an additional summary would be noisy).
// Both lines go directly to os.Stderr — Reporter has no "plain stderr
// line" primitive, and using Header/Box/Info would over-format the bare
// summary.
//
// silent suppresses both the summary and the Errors: section to match
// the rest of stderr summary output. (Failed targets are already
// surfaced through Targets.Fail before this point and through the
// top-level Error in cmd/root.go, so the Errors: section is redundant
// under --silent.) Callers in cmd/ pass isSilent(); tests pass an
// explicit bool, which removes the package-global mutation that used to
// race under -race.
func renderBatchSummary(summary batch.Summary, n int, cfg summaryConfig, silent bool) {
	if silent {
		return
	}
	if n < 2 {
		renderErrorsSection(summary, silent)
		return
	}
	noOp := summary.NoOp + summary.Skipped
	line := fmt.Sprintf("%d ok, %d %s, %d failed", summary.OK, noOp, cfg.noopVerb, summary.Failed)
	if cfg.withElapsed && summary.Elapsed > 0 {
		line += " (" + cli.FormatElapsed(summary.Elapsed) + ")"
	}
	_, _ = fmt.Fprintln(os.Stderr)
	_, _ = fmt.Fprintln(os.Stderr, line)
	renderErrorsSection(summary, silent)
}

// renderErrorsSection prints the Errors: block beneath the summary when
// any target failed. Resolution hints come from internal/apcerrors so the
// list stays curated; unknown error types omit the Resolution line
// entirely. silent has the same meaning as in renderBatchSummary.
func renderErrorsSection(summary batch.Summary, silent bool) {
	if silent || len(summary.Errors) == 0 {
		return
	}
	_, _ = fmt.Fprintln(os.Stderr)
	_, _ = fmt.Fprintln(os.Stderr, "Errors:")
	for _, e := range summary.Errors {
		_, _ = fmt.Fprintln(os.Stderr, "  "+e.Identifier)
		_, _ = fmt.Fprintln(os.Stderr, "    "+e.Err.Error())
		if hint := apcerrors.Resolution(e.Err); hint != "" {
			_, _ = fmt.Fprintln(os.Stderr, "    Resolution: "+hint)
		}
	}
}
