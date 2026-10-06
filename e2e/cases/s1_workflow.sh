#!/usr/bin/env bash
# S1: Basic workflow — ls-resources → init → diff → run → status → get → pull.
# Sourced by e2e-test.sh inside a per-section tempdir.

section "S1" "Basic workflow"

step "ls-resources lists the test app + region"
ls_json=$(apc_stdout ls-resources --region "$REGION" --json --silent)
assert_jq <(printf '%s' "$ls_json") ".region == \"$REGION\""
assert_jq <(printf '%s' "$ls_json") '.applications | length > 0'
assert_jq <(printf '%s' "$ls_json") \
    "[.applications[].name] | index(\"$APP\") != null"

step "ls-resources --show-strategies includes AppConfig.AllAtOnce"
ls_strat=$(apc_stdout ls-resources --region "$REGION" \
    --show-strategies --json --silent)
assert_jq <(printf '%s' "$ls_strat") \
    '[.deployment_strategies[].name] | index("AppConfig.AllAtOnce") != null'

step "init creates apcdeploy.yml + data.json"
apc_init json-freeform dev
apc_use_strategy
apc_write_json '{"v":"1"}'

step "diff (non-silent) emits a Targets transition on stderr"
diff_out=$(apc_combined diff || true)
assert_match "$diff_out" \
    '(no changes|no prior deployment|diff \()' "diff progress"

step "diff stdout contains the data payload"
diff_stdout=$(apc_stdout diff || true)
assert_contains "$diff_stdout" 'v' "diff body"

step "diff --silent suppresses progress on stderr"
silent_err=$(apc_combined diff --silent || true)
assert_not_contains "$silent_err" "no changes" "silent stderr"
assert_not_contains "$silent_err" "diff (" "silent stderr"

step "run --wait-bake completes deployment"
apc_quiet run --wait-bake --silent

step "status reports COMPLETE"
status_out=$(apc_stdout status --silent)
assert_contains "$status_out" "COMPLETE" "status"

step "get returns deployed payload v=1"
get_out=$(apc_stdout get --silent --yes)
assert_jq <(printf '%s' "$get_out") '.v == "1"'

step "pull restores deployed state after local modification"
apc_write_json '{"v":"modified"}'
apc_quiet pull --silent
assert_jq data.json '.v == "1"'

step "round-trip: deploy v=2, get returns v=2"
apc_write_json '{"v":"2"}'
apc_quiet run --wait-bake --silent
get_out=$(apc_stdout get --silent --yes)
assert_jq <(printf '%s' "$get_out") '.v == "2"'
