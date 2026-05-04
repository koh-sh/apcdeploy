#!/usr/bin/env bash
set -eu

cd "$(dirname "$0")/.."
go build -o e2e/apcdeploy

APCDEPLOY="./apcdeploy"
APP="${E2E_APP:-apcdeploy-e2e-test}"
REGION="${E2E_REGION:-ap-northeast-1}"
STRATEGY="E2E-Test-Strategy"
WORKDIR="./e2e/"

use_strategy() { sed -i '' "s/deployment_strategy:.*/deployment_strategy: ${STRATEGY}/" apcdeploy.yml; }
use_slow_strategy() { sed -i '' "s/deployment_strategy:.*/deployment_strategy: E2E-Slow-Strategy/" apcdeploy.yml; }

# Fake editor used by S7/E3/E5. Reads content from $APCDEPLOY_EDIT_CONTENT
# rather than embedding it, so callers can vary the bytes safely without
# re-writing the fixture or worrying about shell-quote escaping.
FAKE_EDITOR=./edit-fixture.sh

# test title colored with green
function title() {
    echo -e "\e[32m \n##### ${1} #####\n \e[m"
}

cd "$WORKDIR"

echo "Basic workflow: ls-resources -> init -> diff -> run -> status -> get -> pull -> update -> run"
title "========== S1: Workflow =========="
echo "Test ls-resources command"
# `--silent` without `--json` produces no stdout payload; pair with `--json` for grepping/jq.
$APCDEPLOY ls-resources --region "$REGION" --json --silent | jq -e --arg app "$APP" '.applications | map(.name) | index($app) != null' > /dev/null
$APCDEPLOY ls-resources --region "$REGION" --json --silent | jq -e ".region == \"$REGION\"" > /dev/null
$APCDEPLOY ls-resources --region "$REGION" --json --silent | jq -e '.applications | length > 0' > /dev/null
$APCDEPLOY ls-resources --region "$REGION" --show-strategies --json --silent | jq -e '.deployment_strategies | map(.name) | index("AppConfig.AllAtOnce") != null' > /dev/null
$APCDEPLOY init --silent --app "$APP" --profile json-freeform --env dev --region "$REGION" --force
use_strategy
echo '{"v":"1"}' > data.json
echo "Test verbose output (without --silent) to verify detailed logging works"
# diff drives a single Targets row through `comparing` → `✓ <outcome>`.
# Non-TTY mode emits a line per transition; the exact `<outcome>` (`no
# changes`, `diff (N lines changed: ...)`, or `no prior deployment`) depends
# on whether a prior deployment exists and whether local matches it. The
# test only cares that *some* progress reaches stderr, so we match the
# Targets transition vocabulary.
$APCDEPLOY diff 2>&1 | grep -qE "(no changes|no prior deployment|diff \()"
$APCDEPLOY diff | grep -q "v"
echo "Test silent mode suppresses verbose output"
if $APCDEPLOY diff --silent 2>&1 | grep -qE "(no changes|no prior deployment|diff \()"; then
    echo "ERROR: Silent mode should not show progress messages"
    exit 1
fi
echo "Rest of tests use --silent for cleaner output"
$APCDEPLOY run --wait-bake --silent
$APCDEPLOY status --silent | grep -q "COMPLETE"
$APCDEPLOY get --silent --yes | jq -e '.v == "1"' > /dev/null
echo "Test pull command: modify local file, then pull to restore deployed state"
echo '{"v":"modified"}' > data.json
$APCDEPLOY pull --silent
jq -e '.v == "1"' data.json > /dev/null
echo '{"v":"2"}' > data.json
$APCDEPLOY run --wait-bake --silent
$APCDEPLOY get --silent --yes | jq -e '.v == "2"' > /dev/null

echo "Support for different content types: FeatureFlags, YAML, text"
title "========== S2: Content Types =========="
$APCDEPLOY init --silent --app "$APP" --profile json-featureflags --env dev --region "$REGION" --force
use_strategy
echo '{"version":"1","flags":{"test":{"name":"test"}}}' > data.json
$APCDEPLOY run --wait-bake --silent

$APCDEPLOY init --silent --app "$APP" --profile yaml-config --env dev --region "$REGION" --force
use_strategy
sed -i '' 's/data.json/data.yaml/' apcdeploy.yml
echo -e "v: 1\nk: v" > data.yaml
$APCDEPLOY run --wait-bake --silent

$APCDEPLOY init --silent --app "$APP" --profile text-config --env dev --region "$REGION" --force
use_strategy
sed -i '' 's/data.json/data.txt/' apcdeploy.yml
echo "text" > data.txt
$APCDEPLOY run --wait-bake --silent

echo "Deployment control: skip unchanged, force deploy, async run"
title "========== S3: Deployment Control =========="
$APCDEPLOY init --silent --app "$APP" --profile json-freeform --env staging --region "$REGION" --force
use_strategy
echo '{"t":"1"}' > data.json
$APCDEPLOY run --wait-bake --silent
$APCDEPLOY run --silent
$APCDEPLOY run --force --wait-bake --silent
echo '{"t":"2"}' > data.json
$APCDEPLOY run --silent

echo "Config file generation and deployment strategy verification"
title "========== S4: Config =========="
$APCDEPLOY init --silent --app "$APP" --profile yaml-config --env dev --region "$REGION" --force
grep -q "region: $REGION" apcdeploy.yml
use_strategy
sed -i '' 's/data.json/data.yaml/' apcdeploy.yml
echo "t: 1" > data.yaml
$APCDEPLOY run --wait-bake --silent
$APCDEPLOY status --silent | grep -q "COMPLETE"

echo "CI mode: diff --exit-nonzero for detecting changes"
title "========== S5: CI =========="
$APCDEPLOY init --silent --app "$APP" --profile text-config --env dev --region "$REGION" --force
use_strategy
sed -i '' 's/data.json/data.txt/' apcdeploy.yml
date > data.txt
cat data.txt
if $APCDEPLOY diff --silent --exit-nonzero; then exit 1; fi
$APCDEPLOY run --wait-bake --timeout 300 --silent
$APCDEPLOY diff --silent --exit-nonzero

echo "Rollback: stop ongoing deployment with slow strategy"
title "========== S6: Rollback =========="
$APCDEPLOY init --silent --app "$APP" --profile json-freeform --env dev --region "$REGION" --force
use_slow_strategy
echo '{"r":"1"}' > data.json
$APCDEPLOY run --silent
$APCDEPLOY rollback --silent --yes
$APCDEPLOY status --silent | grep -q "ROLLED_BACK"
if $APCDEPLOY rollback --silent --yes; then exit 1; fi

echo "Edit: $EDITOR-driven workflow (happy, no-op, invalid)"
title "========== S7: Edit =========="
$APCDEPLOY init --silent --app "$APP" --profile json-freeform --env dev --region "$REGION" --force
use_strategy
echo '{"e":"seed"}' > data.json
$APCDEPLOY run --wait-bake --silent

EDITOR=$FAKE_EDITOR APCDEPLOY_EDIT_CONTENT='{"e":"new"}' $APCDEPLOY edit --region "$REGION" --app "$APP" --profile json-freeform --env dev --wait-bake --silent
$APCDEPLOY get --silent --yes | jq -e '.e == "new"' > /dev/null
EDITOR=$FAKE_EDITOR APCDEPLOY_EDIT_CONTENT='{"e":"new"}' $APCDEPLOY edit --region "$REGION" --app "$APP" --profile json-freeform --env dev 2>&1 | grep -q "no changes"
if EDITOR=$FAKE_EDITOR APCDEPLOY_EDIT_CONTENT='not-json' $APCDEPLOY edit --region "$REGION" --app "$APP" --profile json-freeform --env dev --silent; then exit 1; fi

echo "Description: default marker, explicit value, opt-out, length cap, edit"
title "========== S8: Description =========="
$APCDEPLOY init --silent --app "$APP" --profile json-freeform --env dev --region "$REGION" --force
use_strategy
echo '{"d":"1"}' > data.json
$APCDEPLOY run --wait-bake --silent
# status renders Description on stderr via Reporter.Table (tab-separated in
# non-TTY mode) — grep stderr to verify the value is what we expect.
$APCDEPLOY status 2>&1 | grep -qF "Deployed by apcdeploy"
echo '{"d":"2"}' > data.json
$APCDEPLOY run --wait-bake --silent --description "explicit run desc"
$APCDEPLOY status 2>&1 | grep -qF "explicit run desc"
echo '{"d":"3"}' > data.json
$APCDEPLOY run --wait-bake --silent --description ""
# Empty --description opts out of the default; no Description row should appear.
if $APCDEPLOY status 2>&1 | grep -q "^Description"; then exit 1; fi
# >1024 runes is rejected client-side before any AWS round-trip.
LONG_DESC=$(printf 'a%.0s' {1..1025})
if $APCDEPLOY run --silent --description "$LONG_DESC"; then exit 1; fi
EDITOR=$FAKE_EDITOR APCDEPLOY_EDIT_CONTENT='{"d":"edited"}' $APCDEPLOY edit --region "$REGION" --app "$APP" --profile json-freeform --env dev --wait-bake --silent --description "explicit edit desc"
$APCDEPLOY status 2>&1 | grep -qF "explicit edit desc"

echo "Error handling: non-existent resources (app/profile/env)"
title "========== E1: Resource Errors =========="
if $APCDEPLOY init --silent --app xxx --profile test --env dev --region "$REGION"; then exit 1; fi
if $APCDEPLOY init --silent --app "$APP" --profile xxx --env dev --region "$REGION"; then exit 1; fi
if $APCDEPLOY init --silent --app "$APP" --profile json-freeform --env xxx --region "$REGION"; then exit 1; fi

echo "Validation errors: invalid JSON syntax"
title "========== E2: Validation =========="
$APCDEPLOY init --silent --app "$APP" --profile json-freeform --env dev --region "$REGION" --force
echo '{"bad": json}' > data.json
if $APCDEPLOY run --silent; then exit 1; fi

echo "Constraint errors: concurrent deployment, timeout, edit while ongoing"
title "========== E3: Constraints =========="
$APCDEPLOY init --silent --app "$APP" --profile error-test --env dev --region "$REGION" --force
use_slow_strategy
echo '{"c":"1"}' > data.json
$APCDEPLOY run --silent >/dev/null 2>&1 &
sleep 2
if $APCDEPLOY run --silent; then exit 1; fi
if EDITOR=$FAKE_EDITOR APCDEPLOY_EDIT_CONTENT='{"c":"x"}' $APCDEPLOY edit --region "$REGION" --app "$APP" --profile error-test --env dev --silent; then exit 1; fi
wait || true

echo '{"c":"2"}' > data.json
if $APCDEPLOY run --wait-bake --timeout 5 --silent; then exit 1; fi

echo "File errors: missing config, invalid config, file exists"
title "========== E4: File Errors =========="
if $APCDEPLOY run --config xxx.yml --silent; then exit 1; fi

$APCDEPLOY init --silent --app "$APP" --profile json-freeform --env dev --region "$REGION" --force
sed -i '' '/application:/d' apcdeploy.yml
if $APCDEPLOY run --silent; then exit 1; fi

$APCDEPLOY init --silent --app "$APP" --profile json-freeform --env dev --region "$REGION" --force
if $APCDEPLOY init --silent --app "$APP" --profile json-freeform --env dev --region "$REGION"; then exit 1; fi
$APCDEPLOY init --silent --app "$APP" --profile json-freeform --env dev --region "$REGION" --force

echo "Edge cases: no deployment history, invalid timeout, missing required flags, exit-code 2 sentinel"
title "========== E5: Edge Cases =========="
$APCDEPLOY init --silent --app "$APP" --profile error-test --env staging --region "$REGION" --force
use_strategy
# Run without `--silent` because the "no deployment" notice is now part of
# the Targets row's terminal state ("✓ no prior deployment" for diff,
# "→ no deployment" for status) — silent mode suppresses Targets entirely
# (only Error / Data / Diff survive).
$APCDEPLOY diff 2>&1 | grep -q "no prior deployment" || echo "⚠️  Deployment may exist"
$APCDEPLOY status 2>&1 | grep -q "no deployment" || echo "⚠️  Deployment may exist"

# pull and edit must exit 2 (not 1) so scripts can distinguish "no prior deployment" from real errors.
rc=0; $APCDEPLOY pull --silent || rc=$?; [ "$rc" -eq 2 ]
rc=0; EDITOR=$FAKE_EDITOR APCDEPLOY_EDIT_CONTENT='{"e":"x"}' $APCDEPLOY edit --region "$REGION" --app "$APP" --profile error-test --env staging --silent || rc=$?; [ "$rc" -eq 2 ]

echo '{"e":"1"}' > data.json
if $APCDEPLOY run --wait-bake --timeout -1 --silent; then exit 1; fi

# Reset working state before multi-config: the previous sections leave
# whatever apcdeploy.yml the last init produced; the multi-config block
# below builds its own fixtures from scratch.
rm -f data.txt data.yaml data.json apcdeploy.yml

echo "Multi-config: -c repeated for run/diff/pull (multi-config.md)"
title "========== S9: Multi-config =========="
MC_DIR=multi-config
rm -rf "$MC_DIR"
mkdir -p "$MC_DIR/dev" "$MC_DIR/stg"

# Build two independent fixtures via init, one per environment so each
# resolves to a distinct (region/app/profile/env) identifier.
( cd "$MC_DIR/dev" && ../../apcdeploy init --silent --app "$APP" --profile json-freeform --env dev --region "$REGION" --force )
sed -i '' "s/deployment_strategy:.*/deployment_strategy: ${STRATEGY}/" "$MC_DIR/dev/apcdeploy.yml"
echo '{"mc":"dev-1"}' > "$MC_DIR/dev/data.json"

( cd "$MC_DIR/stg" && ../../apcdeploy init --silent --app "$APP" --profile json-freeform --env staging --region "$REGION" --force )
sed -i '' "s/deployment_strategy:.*/deployment_strategy: ${STRATEGY}/" "$MC_DIR/stg/apcdeploy.yml"
echo '{"mc":"stg-1"}' > "$MC_DIR/stg/data.json"

# Multi-config run deploys both targets; --wait-bake ensures we exit
# only after both have COMPLETE.
$APCDEPLOY run -c "$MC_DIR/dev/apcdeploy.yml" -c "$MC_DIR/stg/apcdeploy.yml" --wait-bake --silent
$APCDEPLOY get --silent --yes -c "$MC_DIR/dev/apcdeploy.yml" | jq -e '.mc == "dev-1"' > /dev/null
$APCDEPLOY get --silent --yes -c "$MC_DIR/stg/apcdeploy.yml" | jq -e '.mc == "stg-1"' > /dev/null

# Multi-config diff: no changes for either target.
$APCDEPLOY diff -c "$MC_DIR/dev/apcdeploy.yml" -c "$MC_DIR/stg/apcdeploy.yml" 2>&1 | grep -qE "no changes"

# Modify one target's data; multi-config diff stdout must include the
# `=== <id> ===` header for the changed target only.
echo '{"mc":"dev-2"}' > "$MC_DIR/dev/data.json"
diff_out=$($APCDEPLOY diff -c "$MC_DIR/dev/apcdeploy.yml" -c "$MC_DIR/stg/apcdeploy.yml" 2>/dev/null || true)
echo "$diff_out" | grep -q "=== ${REGION}/${APP}/json-freeform/dev ==="
if echo "$diff_out" | grep -q "=== ${REGION}/${APP}/json-freeform/staging ==="; then
    echo "ERROR: staging had no changes — its === header should not appear"
    exit 1
fi

# Multi-config pull: rewriting the changed local file restores the
# deployed content, then both files report no changes.
$APCDEPLOY pull -c "$MC_DIR/dev/apcdeploy.yml" -c "$MC_DIR/stg/apcdeploy.yml" --silent
jq -e '.mc == "dev-1"' "$MC_DIR/dev/data.json" > /dev/null
jq -e '.mc == "stg-1"' "$MC_DIR/stg/data.json" > /dev/null

# Duplicate-target detection: pointing the same -c twice must be
# silently deduplicated; pointing two configs at the same identifier
# (different paths) must error.
$APCDEPLOY diff -c "$MC_DIR/dev/apcdeploy.yml" -c "$MC_DIR/dev/apcdeploy.yml" --silent
mkdir -p "$MC_DIR/dup"
cp "$MC_DIR/dev/apcdeploy.yml" "$MC_DIR/dup/apcdeploy.yml"
cp "$MC_DIR/dev/data.json" "$MC_DIR/dup/data.json"
if $APCDEPLOY diff -c "$MC_DIR/dev/apcdeploy.yml" -c "$MC_DIR/dup/apcdeploy.yml" --silent; then
    echo "ERROR: duplicate target should have failed"
    exit 1
fi

# Single-config commands reject multi -c with a clear error.
if $APCDEPLOY status -c "$MC_DIR/dev/apcdeploy.yml" -c "$MC_DIR/stg/apcdeploy.yml" --silent 2>/dev/null; then
    echo "ERROR: status must reject multi -c"
    exit 1
fi

rm -rf "$MC_DIR"
rm -f apcdeploy
echo "✅ All tests passed"
