#!/usr/bin/env bash
# S5: Generated config + deployment-strategy verification (yaml profile).

section "S5" "Config generation"

step "init writes region into apcdeploy.yml"
apc_init yaml-config dev
assert_grep apcdeploy.yml "region: $REGION"

step "deploy yaml content and verify status"
apc_use_strategy
apc_write_yaml 't: 1'
apc_quiet run --wait-bake --silent
status_out=$(apc_stdout status --silent)
assert_contains "$status_out" "COMPLETE" "status"
