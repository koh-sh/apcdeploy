#!/usr/bin/env bash
# E2: Client-side validation rejects malformed JSON before any AWS call.

section "E2" "Validation"

step "init json-freeform/dev"
apc_init json-freeform dev

step "run with broken JSON exits non-zero"
apc_write_json '{"bad": json}'
expect_fail "$APCDEPLOY_BIN" run --silent
