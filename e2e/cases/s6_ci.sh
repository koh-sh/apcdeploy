#!/usr/bin/env bash
# S6: CI mode — diff --exit-nonzero detects changes vs deployed state.

section "S6" "CI diff --exit-nonzero"

step "init text profile + write fresh content"
apc_init text-config dev
apc_use_strategy
apc_write_text "$(date)"

step "diff --exit-nonzero exits 1 when local diverges from deployed"
expect_exit 1 "$APCDEPLOY_BIN" diff --silent --exit-nonzero

step "after run, diff --exit-nonzero exits 0"
apc_quiet run --wait-bake --timeout 300 --silent
apc_quiet diff --silent --exit-nonzero
