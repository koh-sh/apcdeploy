#!/usr/bin/env bash
# E2: Client-side validation rejects malformed JSON before anything is written.

section "E2" "Validation"

step "init json-freeform/dev"
apc_init json-freeform dev
# Pin the strategy so `run` resolves the same resources regardless of which
# strategy init inherited from a prior deployment.
apc_use_strategy

step "run with broken JSON exits non-zero"
apc_write_json '{"bad": json}'
expect_fail_with "invalid JSON syntax" "$APCDEPLOY_BIN" run --silent
