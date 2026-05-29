package deploywait

import "time"

// RemainingDuration returns the time until deadline, clamped at 1s to
// avoid passing 0/negative values to AWS polling helpers. The actual wait
// is bounded by the shared waitCtx deadline regardless, so the floor only
// matters when this helper is called after the budget is already exhausted.
func RemainingDuration(deadline time.Time) time.Duration {
	remaining := time.Until(deadline)
	if remaining < time.Second {
		return time.Second
	}
	return remaining
}
