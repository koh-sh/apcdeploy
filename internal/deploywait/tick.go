package deploywait

import (
	"fmt"
	"math"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/appconfig/types"
	"github.com/koh-sh/apcdeploy/internal/aws"
	"github.com/koh-sh/apcdeploy/internal/reporter"
)

// MakeTargetsDeployTick returns an aws.DeploymentTickFunc that drives a
// row's deploying sub-phase via SetProgress. Once BAKING (or COMPLETE)
// is observed the percent pins at 1.0 and the eta is cleared so callers
// can swap the row to a "baking" sub-phase via SetPhase.
//
// The "(~N min left)" countdown is derived from wall-clock elapsed time
// (waitStart) minus the strategy's totalDuration so non-linear strategies
// (EXPONENTIAL) report honest remaining time.
func MakeTargetsDeployTick(tr reporter.TargetReporter) aws.DeploymentTickFunc {
	waitStart := time.Now()
	return func(state types.DeploymentState, percent float64, totalDuration time.Duration) {
		if state == types.DeploymentStateBaking || state == types.DeploymentStateComplete {
			tr.SetProgress(1.0, 0)
			return
		}
		eta := max(totalDuration-time.Since(waitStart), 0)
		tr.SetProgress(percent/100.0, eta)
	}
}

// MakeTargetsBakeTick returns an aws.BakeTickFunc that updates the row's
// baking sub-phase detail with the current "(~N min left)" countdown.
// The row is expected to already be in the baking sub-phase (the caller
// invokes SetPhase("baking", "") before starting the bake wait).
func MakeTargetsBakeTick(tr reporter.TargetReporter) aws.BakeTickFunc {
	return func(elapsed, total time.Duration) {
		tr.SetPhase("baking", remainingFromElapsedSuffix(elapsed, total))
	}
}

// remainingFromElapsedSuffix renders a "(~N min left)" suffix from total
// minus locally observed elapsed time. Falls back to "(<1 min left)" when
// total is zero (e.g. AppConfig.AllAtOnce), when elapsed has already run
// past total, or when the remaining is below one minute. The function
// always returns a non-empty string so the bar always carries a time hint,
// and the threshold is on the duration itself (not math.Ceil) so 30 s and
// 59 s render honestly as "<1 min left" instead of being rounded up to
// "~1 min left".
func remainingFromElapsedSuffix(elapsed, total time.Duration) string {
	remaining := total - elapsed
	if total <= 0 || remaining < time.Minute {
		return " (<1 min left)"
	}
	return fmt.Sprintf(" (~%d min left)", int(math.Ceil(remaining.Minutes())))
}
