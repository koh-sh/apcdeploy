#!/usr/bin/env bash
# E5: Edge cases — no-deployment notice, exit code 2 sentinel, invalid timeout.

section "E5" "Edge cases"

step "init error-test/staging (assumed to have no prior deployment)"
apc_init error-test staging
apc_use_strategy
# init does not create data.json when there is no prior deployment, so we
# write a placeholder ourselves; diff/status only care about the AWS-side
# absence of a prior deployment, but they still need to be able to load
# the local data file. (In the original e2e-test.sh this was masked by
# stale data.json leaking from earlier sections in the shared cwd.)
apc_write_json '{"e":"1"}'

step "diff reports 'no prior deployment' on stderr"
out=$(apc_combined diff || true)
assert_contains "$out" "no prior deployment" "diff stderr"

step "status reports 'no deployment' on stderr"
out=$(apc_combined status || true)
assert_contains "$out" "no deployment" "status stderr"

step "pull exits with code 2 sentinel when no prior deployment"
expect_exit 2 "$APCDEPLOY_BIN" pull --silent

step "edit exits with code 2 sentinel when no prior deployment"
expect_exit 2 env EDITOR="$FAKE_EDITOR" APCDEPLOY_EDIT_CONTENT='{"e":"x"}' \
    "$APCDEPLOY_BIN" edit --region "$REGION" --app "$APP" \
    --profile error-test --env staging --silent

step "run with negative --timeout fails"
expect_fail "$APCDEPLOY_BIN" run --wait-bake --timeout -1 --silent
