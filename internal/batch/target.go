// Package batch coordinates the multi-config (`-c` repeated) execution
// path used by run / diff / pull. See docs/design/multi-config.md.
//
// The package owns two responsibilities:
//
//   - LoadAll: pre-load and validate every config given on the command
//     line before any AWS work begins, and surface load-time errors as a
//     single batch (multi-config.md §9 / §10.1 "事前エラー").
//
//   - Orchestrator: drive the loaded targets through executor functions,
//     in parallel by default, with optional fail-fast / continue-on-error
//     semantics (multi-config.md §7).
//
// N=1 and N=N flow through the same code path; the orchestrator simply
// runs a single goroutine when only one target is loaded
// (multi-config.md §3 F-10).
package batch

import (
	"github.com/koh-sh/apcdeploy/internal/config"
)

// Target is one loaded `-c` argument: the original path (preserved for
// error messages — multi-config.md §6.3), the loaded Config, and the
// canonical identifier used for duplicate detection and progress
// rendering.
type Target struct {
	// Path is the path string as supplied on the command line. It is
	// kept verbatim so error messages refer back to what the user typed
	// ("./envs/dev.yml") rather than the absolute form used internally
	// for dedup.
	Path string

	// Config is the loaded and validated configuration.
	Config *config.Config

	// Identifier is the "region/app/profile/env" 4-tuple
	// (multi-config.md §6.1). Computed from Config at load time so the
	// orchestrator and Reporter agree on the row label.
	Identifier string
}
