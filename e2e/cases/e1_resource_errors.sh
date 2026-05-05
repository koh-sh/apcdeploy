#!/usr/bin/env bash
# E1: Non-existent application / profile / environment all fail init.

section "E1" "Resource errors"

step "init with unknown application fails"
expect_fail "$APCDEPLOY_BIN" init --silent --region "$REGION" \
    --app xxx --profile test --env dev

step "init with unknown profile fails"
expect_fail "$APCDEPLOY_BIN" init --silent --region "$REGION" \
    --app "$APP" --profile xxx --env dev

step "init with unknown environment fails"
expect_fail "$APCDEPLOY_BIN" init --silent --region "$REGION" \
    --app "$APP" --profile json-freeform --env xxx
