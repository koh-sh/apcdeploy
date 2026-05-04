package cli

import (
	"fmt"
	"time"
)

// FormatDeploymentSummary builds the post-icon Targets summary line for a
// run / edit deployment:
//
//	<verb> [(<elapsed>)] [— v<N>[, <Strategy>][, <addendum>]]
//
// elapsed is the duration to render. Pass 0 to omit it entirely (used for
// the "started" verb, which has no wait phase to time). The caller is
// responsible for choosing the right source of elapsed — wall-clock for
// --wait-deploy, AWS-reported `CompletedAt - StartedAt` for --wait-bake
// (so the displayed time isn't inflated by the polling tick lag).
//
// addendum is appended after the strategy when non-empty (e.g.
// "baking started", "deployment #42").
//
// Centralised here so run and edit cannot drift — both packages were carrying
// identical implementations before.
func FormatDeploymentSummary(verb string, elapsed time.Duration, version int32, strategy, addendum string) string {
	out := verb
	if elapsed > 0 && verb != "started" {
		out += " (" + FormatElapsed(elapsed) + ")"
	}
	if version > 0 {
		out += fmt.Sprintf(" — v%d", version)
	}
	if strategy != "" {
		if version > 0 {
			out += ", " + strategy
		} else {
			out += " — " + strategy
		}
	}
	if addendum != "" {
		out += ", " + addendum
	}
	return out
}

// FormatElapsed renders a duration as compact "Ns" or "Nm Ns" (or "Nm"
// when the seconds part is zero). Used by FormatDeploymentSummary and
// by the multi-config aggregate summary.
//
// Sub-second precision is dropped via Round (nearest-second), so the
// displayed value is the closest integer second to the actual
// duration — consistent with AppConfig's microsecond-precision
// timestamps shown in the AWS console / event log. Truncate would
// always under-report by up to 1s and bias the display.
func FormatElapsed(d time.Duration) string {
	d = d.Round(time.Second)
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	m := int(d.Minutes())
	s := int(d.Seconds()) - m*60
	if s == 0 {
		return fmt.Sprintf("%dm", m)
	}
	return fmt.Sprintf("%dm %ds", m, s)
}
