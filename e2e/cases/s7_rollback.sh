#!/usr/bin/env bash
# S7: Rollback — start a slow deploy, stop it, observe ROLLED_BACK.

section "S7" "Rollback"

step "start slow deploy on json-freeform/dev"
apc_init json-freeform dev
apc_use_strategy "$SLOW_STRATEGY"
apc_write_json '{"r":"1"}'
apc_quiet run --silent

step "rollback --yes stops the ongoing deployment"
apc_quiet rollback --silent --yes

step "status reports ROLLED_BACK"
status_out=$(apc_stdout status --silent)
assert_contains "$status_out" "ROLLED_BACK" "status"

step "second rollback with no ongoing deployment fails"
expect_fail "$APCDEPLOY_BIN" rollback --silent --yes
