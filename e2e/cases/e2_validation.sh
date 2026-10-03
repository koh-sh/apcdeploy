#!/usr/bin/env bash
# E2: Client-side validation rejects malformed JSON before anything is written.

section "E2" "Validation"

step "init json-freeform/dev"
apc_init json-freeform dev
# Pin the strategy so `run` resolves resources regardless of prior deployments
# (init falls back to AppConfig.AllAtOnce, which MiniStack does not provide).
apc_use_strategy

step "run with broken JSON exits non-zero"
apc_write_json '{"bad": json}'
run_rc=0
run_out=$(apc_combined run --silent) || run_rc=$?
[[ "$run_rc" -ne 0 ]] || fail "run unexpectedly succeeded with broken JSON"
# Assert the reason too, so an unrelated failure cannot pass this step.
assert_contains "$run_out" "invalid JSON syntax" "run error"
