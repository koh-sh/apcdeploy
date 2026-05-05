#!/usr/bin/env bash
# S3: Multi-config orchestration — `-c` repeated for run/diff/pull.
# Must run before any later section perturbs json-freeform/{dev,staging}.

section "S3" "Multi-config"

mc_dir=multi-config
mkdir -p "$mc_dir/dev" "$mc_dir/stg"

step "init populates per-target apcdeploy.yml under multi-config/{dev,stg}"
( cd "$mc_dir/dev" && apc_quiet init --silent --force \
    --app "$APP" --region "$REGION" --profile json-freeform --env dev )
( cd "$mc_dir/stg" && apc_quiet init --silent --force \
    --app "$APP" --region "$REGION" --profile json-freeform --env staging )

step "tweak strategy + data per target"
for d in "$mc_dir/dev" "$mc_dir/stg"; do
    ( cd "$d" && apc_use_strategy )
done
printf '%s' '{"mc":"dev-1"}' > "$mc_dir/dev/data.json"
printf '%s' '{"mc":"stg-1"}' > "$mc_dir/stg/data.json"

step "multi-config run --wait-bake deploys both targets"
apc_quiet run --wait-bake --silent \
    -c "$mc_dir/dev/apcdeploy.yml" -c "$mc_dir/stg/apcdeploy.yml"

step "get against each target returns its own payload"
dev_out=$(apc_stdout get --silent --yes -c "$mc_dir/dev/apcdeploy.yml")
stg_out=$(apc_stdout get --silent --yes -c "$mc_dir/stg/apcdeploy.yml")
assert_jq <(printf '%s' "$dev_out") '.mc == "dev-1"'
assert_jq <(printf '%s' "$stg_out") '.mc == "stg-1"'

step "multi-config diff reports no changes for both targets"
out=$(apc_combined diff \
    -c "$mc_dir/dev/apcdeploy.yml" -c "$mc_dir/stg/apcdeploy.yml" || true)
assert_match "$out" "no changes" "diff progress"

step "diff body shows === <id> === only for the changed target"
printf '{"mc":"dev-2"}' > "$mc_dir/dev/data.json"
diff_out=$(apc_stdout diff \
    -c "$mc_dir/dev/apcdeploy.yml" -c "$mc_dir/stg/apcdeploy.yml" || true)
assert_contains "$diff_out" "=== ${REGION}/${APP}/json-freeform/dev ===" "diff header"
assert_not_contains "$diff_out" \
    "=== ${REGION}/${APP}/json-freeform/staging ===" "diff header"

step "multi-config pull restores changed target"
apc_quiet pull --silent \
    -c "$mc_dir/dev/apcdeploy.yml" -c "$mc_dir/stg/apcdeploy.yml"
assert_jq "$mc_dir/dev/data.json" '.mc == "dev-1"'
assert_jq "$mc_dir/stg/data.json" '.mc == "stg-1"'

step "path-level dedup: same -c twice is collapsed silently"
apc_quiet diff --silent \
    -c "$mc_dir/dev/apcdeploy.yml" -c "$mc_dir/dev/apcdeploy.yml"

step "identifier-level dedup: distinct paths same 4-tuple is rejected"
mkdir -p "$mc_dir/dup"
cp "$mc_dir/dev/apcdeploy.yml" "$mc_dir/dup/apcdeploy.yml"
cp "$mc_dir/dev/data.json" "$mc_dir/dup/data.json"
expect_fail "$APCDEPLOY_BIN" diff --silent \
    -c "$mc_dir/dev/apcdeploy.yml" -c "$mc_dir/dup/apcdeploy.yml"

step "single-config commands reject multi -c"
expect_fail "$APCDEPLOY_BIN" status --silent \
    -c "$mc_dir/dev/apcdeploy.yml" -c "$mc_dir/stg/apcdeploy.yml"
