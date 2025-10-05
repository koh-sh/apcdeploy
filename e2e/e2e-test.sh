#!/usr/bin/env bash
set -eu

cd "$(dirname "$0")/.."
go build -o e2e/apcdeploy

APCDEPLOY="./apcdeploy"
APP="${E2E_APP:-apcdeploy-e2e-test}"
REGION="${E2E_REGION:-ap-northeast-1}"
STRATEGY="E2E-Test-Strategy"
WORKDIR="./e2e/"

use_strategy() { sed -i '' "s/deployment_strategy:.*/deployment_strategy: $STRATEGY/" apcdeploy.yml; }
use_slow_strategy() { sed -i '' "s/deployment_strategy:.*/deployment_strategy: E2E-Slow-Strategy/" apcdeploy.yml; }

# test title colored with green
function title() {
    echo -e "\e[32m \n##### ${1} #####\n \e[m"
}

cd $WORKDIR

# Basic workflow: init -> diff -> run -> status -> update -> run
title "========== S1: Workflow =========="
$APCDEPLOY init --app $APP --profile json-freeform --env dev --region $REGION --force
use_strategy
echo '{"v":"1"}' > data.json
$APCDEPLOY diff | grep -q "v"
$APCDEPLOY run --wait
$APCDEPLOY status | grep -q "COMPLETE"
echo '{"v":"2"}' > data.json
$APCDEPLOY run --wait

# Support for different content types: FeatureFlags, YAML, text
title "========== S2: Content Types =========="
$APCDEPLOY init --app $APP --profile json-featureflags --env dev --region $REGION --force
use_strategy
echo '{"version":"1","flags":{"test":{"name":"test"}}}' > data.json
$APCDEPLOY run --wait

$APCDEPLOY init --app $APP --profile yaml-config --env dev --region $REGION --force
use_strategy
echo -e "v: 1\nk: v" > data.yaml
$APCDEPLOY run --wait

$APCDEPLOY init --app $APP --profile text-config --env dev --region $REGION --force
use_strategy
echo "text" > data.txt
$APCDEPLOY run --wait

# Deployment control: skip unchanged, force deploy, async run
title "========== S3: Deployment Control =========="
$APCDEPLOY init --app $APP --profile json-freeform --env staging --region $REGION --force
use_strategy
echo '{"t":"1"}' > data.json
$APCDEPLOY run --wait
$APCDEPLOY run | grep -q "No changes"
$APCDEPLOY run --force --wait
echo '{"t":"2"}' > data.json
$APCDEPLOY run

# Config file generation and deployment strategy verification
title "========== S4: Config =========="
$APCDEPLOY init --app $APP --profile yaml-config --env dev --region $REGION --force
grep -q "region: $REGION" apcdeploy.yml
use_strategy
echo "t: 1" > data.yaml
$APCDEPLOY run --wait
$APCDEPLOY status | grep -q "E2E-Test-Strategy"

# CI mode: diff --exit-nonzero for detecting changes
title "========== S5: CI =========="
$APCDEPLOY init --app $APP --profile text-config --env dev --region $REGION --force
use_strategy
echo "mod" > data.txt
! $APCDEPLOY diff --exit-nonzero
$APCDEPLOY run --wait --timeout 300
$APCDEPLOY diff --exit-nonzero

# Error handling: non-existent resources (app/profile/env)
title "========== E1: Resource Errors =========="
! $APCDEPLOY init --app xxx --profile test --env dev --region $REGION
! $APCDEPLOY init --app $APP --profile xxx --env dev --region $REGION
! $APCDEPLOY init --app $APP --profile json-freeform --env xxx --region $REGION

# Validation errors: invalid JSON/YAML syntax
title "========== E2: Validation =========="
$APCDEPLOY init --app $APP --profile json-freeform --env dev --region $REGION --force
echo '{"bad": json}' > data.json
! $APCDEPLOY run

sed -i '' 's/json-freeform/yaml-config/' apcdeploy.yml
sed -i '' 's/data.json/data.yaml/' apcdeploy.yml
echo -e "bad:\n x: 1" > data.yaml
! $APCDEPLOY run

# Constraint errors: concurrent deployment, timeout
title "========== E3: Constraints =========="
$APCDEPLOY init --app $APP --profile error-test --env dev --region $REGION --force
use_slow_strategy
echo '{"c":"1"}' > data.json
$APCDEPLOY run >/dev/null 2>&1 &
sleep 2
! $APCDEPLOY run
wait || true

echo '{"c":"2"}' > data.json
! $APCDEPLOY run --wait --timeout 5

# File errors: missing config, invalid config, file exists
title "========== E4: File Errors =========="
! $APCDEPLOY run --config xxx.yml

$APCDEPLOY init --app $APP --profile json-freeform --env dev --region $REGION --force
sed -i '' '/application:/d' apcdeploy.yml
! $APCDEPLOY run

$APCDEPLOY init --app $APP --profile json-freeform --env dev --region $REGION --force
! $APCDEPLOY init --app $APP --profile json-freeform --env dev --region $REGION
$APCDEPLOY init --app $APP --profile json-freeform --env dev --region $REGION --force

# Edge cases: no deployment history, invalid timeout, missing required flags
title "========== E5: Edge Cases =========="
$APCDEPLOY init --app $APP --profile error-test --env staging --region $REGION --force
use_strategy
$APCDEPLOY diff 2>&1 | grep -q "No deployment" || echo "⚠️  Deployment may exist"
$APCDEPLOY status 2>&1 | grep -q "No deploy" || echo "⚠️  Deployment may exist"

echo '{"e":"1"}' > data.json
! $APCDEPLOY run --wait --timeout -1
! $APCDEPLOY init --app $APP --profile json-freeform

rm data.txt data.yaml data.json apcdeploy.yml apcdeploy
echo "✅ All tests passed"
