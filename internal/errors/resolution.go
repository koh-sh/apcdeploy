// Package errors maps AWS API error codes to short, actionable user-facing
// hints rendered in the Errors: section of CLI output.
//
// The map is intentionally minimal: only the most frequent failure codes are
// covered (see docs/design/output.md §8.3). New entries are added on demand
// when a real failure shows up — no speculative hints.
package errors

import (
	"errors"

	"github.com/aws/smithy-go"
)

// resolutionHints maps AWS API error codes to user-facing remediation hints.
// Keys are the short ErrorCode() strings returned by smithy APIError values
// (e.g. "ConflictException"), not full type names.
var resolutionHints = map[string]string{
	"ConflictException":         "wait for the current deployment to complete or run 'apcdeploy rollback'.",
	"BadRequestException":       "check your configuration data, JSON/YAML syntax, and any configured validators (JSON Schema / Lambda).",
	"ResourceNotFoundException": "verify the resource names with 'apcdeploy ls-resources' and your AWS region.",
}

// Resolution returns the canonical hint for err's AWS error code, or an empty
// string when err is nil, not an AWS API error, or the code is not in the
// known set.
//
// Callers print "Resolution: <hint>" only when the returned string is
// non-empty (do not invent hints for unknown codes — see output.md §8.3).
func Resolution(err error) string {
	if err == nil {
		return ""
	}
	var apiErr smithy.APIError
	if !errors.As(err, &apiErr) {
		return ""
	}
	return resolutionHints[apiErr.ErrorCode()]
}
