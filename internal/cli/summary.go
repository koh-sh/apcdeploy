package cli

import (
	"fmt"
	"time"
)

// FormatDeploymentSummary builds the post-icon Targets summary line for a
// run / edit deployment per docs/design/output.md §3.3.2:
//
//	<verb> [(<elapsed>)] [— v<N>[, <Strategy>][, <addendum>]]
//
// elapsed is omitted when start is the zero value, or when verb is "started"
// (no wait flag was set, so there is no meaningful deploy duration to quote).
// addendum is appended after the strategy when non-empty (e.g.
// "baking started", "deployment #42").
//
// Centralised here so run and edit cannot drift — both packages were carrying
// identical implementations before.
func FormatDeploymentSummary(verb string, start time.Time, version int32, strategy, addendum string) string {
	out := verb
	if !start.IsZero() && verb != "started" {
		out += " (" + FormatElapsed(time.Since(start)) + ")"
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

// FormatElapsed renders a duration as compact "Ns" or "Nm Ns" (or "Nm" when
// the seconds part is zero). Used by FormatDeploymentSummary; exported so
// callers that build their own summary strings can stay consistent.
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
