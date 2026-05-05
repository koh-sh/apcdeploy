#!/usr/bin/env bash
# S4: Deployment control — skip-unchanged, --force, async run.

section "S4" "Deployment control"

step "init + first deploy of staging seed"
apc_init json-freeform staging
apc_use_strategy
apc_write_json '{"t":"1"}'
apc_quiet run --wait-bake --silent

step "second run with no changes is a no-op"
# Without --silent so the "skipped (no changes)" Targets line is visible.
out=$(apc_combined run)
assert_contains "$out" "no changes" "skip-unchanged"

step "--force re-deploys identical content"
# --force should bypass the no-change short-circuit; verify by absence of
# the skip marker in the rendered Targets row.
out=$(apc_combined run --force --wait-bake)
assert_not_contains "$out" "no changes" "force ignores no-change"

step "async run (no --wait-*) returns immediately on changed content"
apc_write_json '{"t":"2"}'
apc_quiet run --silent
