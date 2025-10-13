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

# test title colored with green
function title() {
    echo -e "\e[32m \n##### ${1} #####\n \e[m"
}

cd "$WORKDIR"

echo "Basic workflow: init -> diff -> run -> status -> get -> update -> run"
title "========== S1: Workflow =========="
$APCDEPLOY init --silent --app "$APP" --profile json-freeform --env dev --region "$REGION" --force
use_strategy
echo '{"v":"1"}' > data.json
echo "Test verbose output (without --silent) to verify detailed logging works"
$APCDEPLOY diff 2>&1 | grep -q "Resolving resources"
$APCDEPLOY diff 2>&1 | grep -q "Fetching latest deployment"
$APCDEPLOY diff | grep -q "v"
echo "Test silent mode suppresses verbose output"
if $APCDEPLOY diff --silent 2>&1 | grep -q "Resolving resources"; then
    echo "ERROR: Silent mode should not show progress messages"
    exit 1
fi
echo "Rest of tests use --silent for cleaner output"
$APCDEPLOY run --wait-bake --silent
$APCDEPLOY status --silent | grep -q "COMPLETE"
$APCDEPLOY get --silent | grep -q '"v":"1"'
echo '{"v":"2"}' > data.json
$APCDEPLOY run --wait-bake --silent
$APCDEPLOY get --silent | grep -q '"v":"2"'

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

echo "Constraint errors: concurrent deployment, timeout"
title "========== E3: Constraints =========="
$APCDEPLOY init --silent --app "$APP" --profile error-test --env dev --region "$REGION" --force
use_slow_strategy
echo '{"c":"1"}' > data.json
$APCDEPLOY run --silent >/dev/null 2>&1 &
sleep 2
if $APCDEPLOY run --silent; then exit 1; fi
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

echo "Edge cases: no deployment history, invalid timeout, missing required flags"
title "========== E5: Edge Cases =========="
$APCDEPLOY init --silent --app "$APP" --profile error-test --env staging --region "$REGION" --force
use_strategy
$APCDEPLOY diff --silent 2>&1 | grep -q "No deployment" || echo "⚠️  Deployment may exist"
$APCDEPLOY status --silent 2>&1 | grep -q "No deploy" || echo "⚠️  Deployment may exist"

echo '{"e":"1"}' > data.json
if $APCDEPLOY run --wait-bake --timeout -1 --silent; then exit 1; fi

rm data.txt data.yaml data.json apcdeploy.yml apcdeploy
echo "✅ All tests passed"
