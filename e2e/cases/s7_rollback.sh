#!/usr/bin/env bash
# S7: Rollback — start a slow deploy, stop it, observe ROLLED_BACK.

section "S7" "Rollback"

if e2e_is_local; then
    skip_section "MiniStack completes deployments instantly, so there is no ongoing deployment to roll back"
    return 0
fi

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
