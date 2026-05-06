// Package batchtest provides test helpers for assembling batch.Target
// fixtures, used by per-command executor tests that exercise
// RunOnTarget directly. Centralising the assembly here keeps the
// Identifier generation rule and the Targets/TargetReporter wiring in a
// single place — individual executor tests should not duplicate this
// plumbing.
package batchtest

import (
	"testing"

	"github.com/koh-sh/apcdeploy/internal/batch"
	"github.com/koh-sh/apcdeploy/internal/config"
	"github.com/koh-sh/apcdeploy/internal/reporter"
	reportertest "github.com/koh-sh/apcdeploy/internal/reporter/testing"
)

// BuildTarget loads configPath, constructs a batch.Target with the
// canonical identifier, opens a single-row Targets handle on rep, and
// returns the per-row TargetReporter view alongside a cleanup function
// that closes the Targets handle. Callers MUST defer cleanup() to avoid
// leaking the rendering goroutine.
//
// Returning a cleanup func instead of opening Targets implicitly mirrors
// the pattern used by the orchestrator (caller owns the lifetime) and
// keeps the helper agnostic to the test's Reporter mock variant.
func BuildTarget(t *testing.T, rep *reportertest.MockReporter, configPath string) (*batch.Target, reporter.TargetReporter, func()) {
	t.Helper()
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("BuildTarget: failed to load config %q: %v", configPath, err)
	}
	target := &batch.Target{
		Path:       configPath,
		Config:     cfg,
		Identifier: config.Identifier(cfg),
	}
	tg := rep.Targets([]string{target.Identifier})
	tr := batch.NewTargetReporter(tg, target.Identifier)
	return target, tr, func() { tg.Close() }
}
