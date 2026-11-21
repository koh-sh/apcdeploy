package batch

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/koh-sh/apcdeploy/internal/config"
)

// ErrDuplicateTarget is returned when two distinct config files resolve to
// the same `region/app/profile/env` 4-tuple. It is exposed as a sentinel so
// command-layer code can react with a tailored exit message.
var ErrDuplicateTarget = errors.New("duplicate target")

// ErrSharedDataFile is returned by RequireDistinctDataFiles when two targets
// resolve to the same data_file.
var ErrSharedDataFile = errors.New("shared data_file")

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

		id := config.Identifier(cfg)

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

// RequireDistinctDataFiles rejects targets whose data_file resolves to the
// same file. Sharing a data file is fine for commands that only read it, but
// a command that writes it (pull) would run the writes concurrently and keep
// whichever target finished last. Paths are compared after cleaning and, for
// files that already exist, with os.SameFile, which also catches symlinked
// directories and case-only differences on case-insensitive filesystems.
func RequireDistinctDataFiles(targets []*Target) error {
	type seen struct {
		target *Target
		path   string
		info   os.FileInfo // nil when the file does not exist yet
	}
	var files []seen
	for _, t := range targets {
		cur := seen{target: t, path: filepath.Clean(t.Config.DataFile)}
		if info, err := os.Stat(cur.path); err == nil {
			cur.info = info
		}
		for _, prev := range files {
			if prev.path == cur.path || (prev.info != nil && cur.info != nil && os.SameFile(prev.info, cur.info)) {
				return fmt.Errorf("%w: %q and %q both write %s", ErrSharedDataFile, prev.target.Path, t.Path, cur.path)
			}
		}
		files = append(files, cur)
	}
	return nil
}
