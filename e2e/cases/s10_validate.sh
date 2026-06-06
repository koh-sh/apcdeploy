#!/usr/bin/env bash
# S10: validate — read-only schema validation across profile types.
# Uses the isolated json-validated / json-lambda profiles so it does not
# perturb (and is not perturbed by) other sections.

section "S10" "Validate"

# --- FeatureFlags: built-in structure schema + per-value constraints ---
step "FeatureFlags valid data passes validate"
apc_init json-featureflags dev
apc_write_json '{"version":"1","flags":{"f":{"name":"f","attributes":{"c":{"constraints":{"type":"string","enum":["a","b"]}}}}},"values":{"f":{"enabled":true,"c":"a"}}}'
apc_quiet validate --silent

step "FeatureFlags constraint violation fails validate"
apc_write_json '{"version":"1","flags":{"f":{"name":"f","attributes":{"c":{"constraints":{"type":"string","enum":["a","b"]}}}}},"values":{"f":{"enabled":true,"c":"z"}}}'
expect_fail "$APCDEPLOY_BIN" validate --silent

# --- Freeform JSON with a remote JSON_SCHEMA validator ---
step "Freeform remote schema: valid data passes"
apc_init json-validated dev
apc_write_json '{"port":8080}'
apc_quiet validate --silent

step "Freeform remote schema: violation exits 1"
apc_write_json '{"port":0}'
expect_exit 1 "$APCDEPLOY_BIN" validate --silent

# --- Freeform JSON without any validator: syntax only ---
step "Freeform no validator: schema-free JSON passes (syntax only)"
apc_init json-freeform dev
apc_write_json '{"literally":"anything"}'
apc_quiet validate --silent

step "Freeform no validator: broken JSON fails"
apc_write_json '{bad json}'
expect_fail "$APCDEPLOY_BIN" validate --silent

# --- LAMBDA validator is skipped (never invoked), reported as syntax-only ---
step "LAMBDA validator is skipped, not checked"
apc_init json-lambda dev
apc_write_json '{"anything":true}'
# No --silent: the Targets summary line carries the skip notice.
out=$(apc_combined validate)
assert_contains "$out" "lambda validator not checked" "validate summary"

# --- Read-only: validate never creates a configuration version ---
step "validate is read-only (does not deploy)"
apc_init json-validated dev
apc_use_strategy
apc_write_json '{"port":1}'
apc_quiet run --wait-bake --silent
apc_write_json '{"port":2}'
apc_quiet validate --silent
body=$(apc_stdout get --silent --yes)
assert_jq <(printf '%s' "$body") '.port == 1'

# --- Multi-config: one invocation validates every target ---
step "multi-config validate checks every target"
mc_dir=validate-mc
mkdir -p "$mc_dir/v" "$mc_dir/ff"
( cd "$mc_dir/v" && apc_quiet init --silent --force \
    --app "$APP" --region "$REGION" --profile json-validated --env dev )
( cd "$mc_dir/ff" && apc_quiet init --silent --force \
    --app "$APP" --region "$REGION" --profile json-featureflags --env dev )
printf '%s' '{"port":5}' > "$mc_dir/v/data.json"
printf '%s' '{"version":"1","flags":{"f":{"name":"f"}}}' > "$mc_dir/ff/data.json"
apc_quiet validate --silent \
    -c "$mc_dir/v/apcdeploy.yml" -c "$mc_dir/ff/apcdeploy.yml"

step "multi-config validate fails when any target is invalid"
printf '%s' '{"port":0}' > "$mc_dir/v/data.json"
expect_fail "$APCDEPLOY_BIN" validate --silent \
    -c "$mc_dir/v/apcdeploy.yml" -c "$mc_dir/ff/apcdeploy.yml"
