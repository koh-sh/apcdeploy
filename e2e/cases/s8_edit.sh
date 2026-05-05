#!/usr/bin/env bash
# S8: $EDITOR-driven edit — happy path, no-op, invalid content rejected.

section "S8" "Edit ($EDITOR workflow)"

step "seed deployment with {\"e\":\"seed\"}"
apc_init json-freeform dev
apc_use_strategy
apc_write_json '{"e":"seed"}'
apc_quiet run --wait-bake --silent

step "edit deploys new content via $EDITOR"
EDITOR="$FAKE_EDITOR" APCDEPLOY_EDIT_CONTENT='{"e":"new"}' \
    apc_quiet edit --region "$REGION" --app "$APP" \
    --profile json-freeform --env dev --wait-bake --silent
get_out=$(apc_stdout get --silent --yes)
assert_jq <(printf '%s' "$get_out") '.e == "new"'

step "edit with identical content reports no changes"
out=$(EDITOR="$FAKE_EDITOR" APCDEPLOY_EDIT_CONTENT='{"e":"new"}' \
    apc_combined edit --region "$REGION" --app "$APP" \
    --profile json-freeform --env dev || true)
assert_contains "$out" "no changes" "edit no-op"

step "edit with invalid JSON is rejected"
expect_fail env EDITOR="$FAKE_EDITOR" APCDEPLOY_EDIT_CONTENT='not-json' \
    "$APCDEPLOY_BIN" edit --region "$REGION" --app "$APP" \
    --profile json-freeform --env dev --silent
