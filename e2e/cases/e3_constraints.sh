#!/usr/bin/env bash
# E3: Concurrency / timeout — second run while first is in flight, edit
# during deploy, and per-call --timeout enforcement.

section "E3" "Constraints"

# Track the slow background deploy so we can stop both the local process
# and the AWS-side rollout if a later step short-circuits via set -e.
__e3_bg_pid=""
__e3_bg_active=0
__e3_cleanup_bg() {
    if [[ -n "$__e3_bg_pid" ]]; then
        kill "$__e3_bg_pid" 2>/dev/null || true
        wait "$__e3_bg_pid" 2>/dev/null || true
    fi
    if (( __e3_bg_active )); then
        # Best-effort: stop the AWS rollout so the next section doesn't
        # see ConflictException on the same profile.
        "$APCDEPLOY_BIN" rollback --yes --silent >/dev/null 2>&1 || true
        __e3_bg_active=0
    fi
}
# This overrides the per-section `trap '__finalize_step ok' EXIT` registered
# by e2e-test.sh, so we re-invoke __finalize_step explicitly to keep the
# trailing ✓ of the last step. Anything else the runner expects on EXIT
# must be added here too.
trap '__e3_cleanup_bg; __finalize_step ok' EXIT

step "start a slow deploy in the background"
apc_init error-test dev
apc_use_strategy "$SLOW_STRATEGY"
apc_write_json '{"c":"1"}'
"$APCDEPLOY_BIN" run --silent >/dev/null 2>&1 &
__e3_bg_pid=$!
__e3_bg_active=1
# Poll status until AWS reports the deploy is in flight (DEPLOYING/BAKING).
# `sleep N` is fragile under CI load — the next step depends on AWS
# actually being mid-rollout, not on wall-clock time.
for _ in $(seq 1 30); do
    state=$(apc_combined status --silent 2>/dev/null || true)
    [[ "$state" == *DEPLOYING* || "$state" == *BAKING* ]] && break
    sleep 1
done
[[ "$state" == *DEPLOYING* || "$state" == *BAKING* ]] \
    || fail "background deploy never reached DEPLOYING/BAKING (state=$state)"

step "second run during ongoing deploy is rejected"
expect_fail "$APCDEPLOY_BIN" run --silent

step "edit during ongoing deploy is rejected"
expect_fail env EDITOR="$FAKE_EDITOR" APCDEPLOY_EDIT_CONTENT='{"c":"x"}' \
    "$APCDEPLOY_BIN" edit --region "$REGION" --app "$APP" \
    --profile error-test --env dev --silent

# Reap the backgrounded run so its rollout doesn't leak into later sections.
wait "$__e3_bg_pid" 2>/dev/null || true
__e3_bg_pid=""
__e3_bg_active=0

step "--wait-bake --timeout 5 fails fast on the slow strategy"
apc_write_json '{"c":"2"}'
expect_fail "$APCDEPLOY_BIN" run --wait-bake --timeout 5 --silent
