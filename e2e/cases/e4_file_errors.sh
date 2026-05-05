#!/usr/bin/env bash
# E4: File-shape errors — missing config, invalid yml, init refuses overwrite.

section "E4" "File errors"

step "run with non-existent --config fails"
expect_fail "$APCDEPLOY_BIN" run --config xxx.yml --silent

step "run with apcdeploy.yml missing 'application:' fails"
apc_init json-freeform dev
apc_remove_field application
expect_fail "$APCDEPLOY_BIN" run --silent

step "init refuses to overwrite without --force"
apc_init json-freeform dev
expect_fail "$APCDEPLOY_BIN" init --silent --region "$REGION" \
    --app "$APP" --profile json-freeform --env dev

step "init --force overwrites successfully"
apc_init json-freeform dev
