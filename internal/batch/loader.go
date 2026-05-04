package batch

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/koh-sh/apcdeploy/internal/config"
)

// ErrDuplicateTarget is returned when two distinct config files resolve to
// the same `region/app/profile/env` 4-tuple. It is exposed as a sentinel so
// command-layer code can react with a tailored exit message.
var ErrDuplicateTarget = errors.New("duplicate target")

// LoadAll reads, parses, and validates every config path supplied on the
// command line, returning one Target per *unique* path in argument order.
//
// The function is the single chokepoint for pre-flight load errors: any
// load failure aborts the batch before the orchestrator runs a single
// AWS call. The first error encountered is returned with the offending
// path included so the user can act on it.
//
// Path-level dedup silently collapses duplicates to one Target.
// Identifier-level duplicates are an explicit error and surface both
// source paths.
func LoadAll(paths []string) ([]*Target, error) {
	if len(paths) == 0 {
		return nil, errors.New("at least one --config path is required")
	}

	targets := make([]*Target, 0, len(paths))
	// seenAbs collapses syntactically distinct paths that point at the
	// same file. Storing the absolute form is enough; we map back to the
	// original path string from the Target itself for messaging.
	seenAbs := make(map[string]struct{}, len(paths))
	// byID lets us pinpoint the *other* path involved in a duplicate
	// without scanning targets twice.
	byID := make(map[string]*Target, len(paths))

	for _, p := range paths {
		abs, err := filepath.Abs(p)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", p, err)
		}
		if _, dup := seenAbs[abs]; dup {
			continue
		}
		seenAbs[abs] = struct{}{}

		cfg, err := config.LoadConfig(p)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", p, err)
		}

		// At load time the only region we know is whatever the user put
		// in the yml. The CLI default-region fallback only applies once
		// an AWS client is built, by which point dedup has already run.
		// Configs that omit `region:` therefore collide on identifier —
		// that's the documented behavior: specify region when using
		// multi-config.
		id := config.Identifier("", cfg)

		if prev, dup := byID[id]; dup {
			return nil, fmt.Errorf(
				"%w: %q is also defined in %q (identifier: %s)",
				ErrDuplicateTarget, p, prev.Path, id,
			)
		}

		t := &Target{Path: p, Config: cfg, Identifier: id}
		byID[id] = t
		targets = append(targets, t)
	}

	return targets, nil
}
