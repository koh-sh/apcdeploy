#!/usr/bin/env bash
# S9: --description — default marker, explicit, opt-out, length cap, edit.

section "S9" "Description"

step "default --description shows 'Deployed by apcdeploy'"
apc_init json-freeform dev
apc_use_strategy
apc_write_json '{"d":"1"}'
apc_quiet run --wait-bake --silent
out=$(apc_combined status)
assert_contains "$out" "Deployed by apcdeploy" "status description"

step "explicit --description is recorded"
apc_write_json '{"d":"2"}'
apc_quiet run --wait-bake --silent --description "explicit run desc"
out=$(apc_combined status)
assert_contains "$out" "explicit run desc" "status description"

step "empty --description opts out: no Description row"
apc_write_json '{"d":"3"}'
apc_quiet run --wait-bake --silent --description ""
out=$(apc_combined status)
refute_line_match "$out" "^Description" "status output"

step "--description >1024 runes is rejected client-side"
LONG_DESC=$(printf 'a%.0s' {1..1025})
expect_fail "$APCDEPLOY_BIN" run --silent --description "$LONG_DESC"

step "edit honors --description"
EDITOR="$FAKE_EDITOR" APCDEPLOY_EDIT_CONTENT='{"d":"edited"}' \
    apc_quiet edit --region "$REGION" --app "$APP" \
    --profile json-freeform --env dev --wait-bake --silent \
    --description "explicit edit desc"
out=$(apc_combined status)
assert_contains "$out" "explicit edit desc" "status description"
