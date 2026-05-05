#!/usr/bin/env bash
# Sourced only — inherits e2e-test.sh's set -euo pipefail.
# apcdeploy CLI wrappers + fixture helpers.
# All wrappers redirect the binary's stderr into $__STDERR_FILE (defined in
# common.sh) so the ERR trap can show the last 10 lines on failure. Each
# wrapper truncates that file on entry so the displayed context is from the
# most recent call only — important because `$(apc_quiet …)` runs in a
# subshell whose local variables are invisible to the parent's trap.

# Run apcdeploy with stderr captured (suppressed unless the call fails).
# Use both as a bare command (`apc_quiet run …`) and inside command
# substitution (`out=$(apc_quiet get …)`).
apc_quiet() {
    : > "$__STDERR_FILE"
    "$APCDEPLOY_BIN" "$@" 2>"$__STDERR_FILE"
}

# Alias retained for readability at call sites that want to signal
# "I'm capturing stdout here" — implementation is intentionally identical
# to apc_quiet.
apc_stdout() {
    apc_quiet "$@"
}

# Merged stdout+stderr — used for assertions on combined output (e.g.
# status's tab-separated rows live on stderr but we want one capture).
# NOTE: stderr is merged into stdout, so $__STDERR_FILE is NOT updated on
# failure here; the ERR trap's "last stderr" section will show data from
# an earlier apc_quiet call instead. Use apc_quiet when you need both
# stdout capture and useful failure diagnostics.
apc_combined() {
    "$APCDEPLOY_BIN" "$@" 2>&1
}

# ---- High-level fixture helpers ------------------------------------------

# apc_init <profile> <env> [extra args...]
# Runs `apcdeploy init --silent --force` against the test app/region.
apc_init() {
    local profile="$1" env="$2"; shift 2
    apc_quiet init --silent --force \
        --app "$APP" --region "$REGION" \
        --profile "$profile" --env "$env" "$@"
}

# apc_use_strategy [strategy_name]
# Patches deployment_strategy in apcdeploy.yml. Defaults to $STRATEGY.
# Uses portable sed (the BSD/GNU difference is hidden behind a temp file move).
apc_use_strategy() {
    local strategy="${1:-$STRATEGY}"
    local tmp; tmp="$(mktemp)"
    sed "s/deployment_strategy:.*/deployment_strategy: ${strategy}/" \
        apcdeploy.yml > "$tmp"
    mv "$tmp" apcdeploy.yml
}

# Internal: patch data_file in apcdeploy.yml (used when changing extension).
__apc_set_data_file() {
    local file="$1"
    local tmp; tmp="$(mktemp)"
    sed "s|data_file:.*|data_file: ${file}|" apcdeploy.yml > "$tmp"
    mv "$tmp" apcdeploy.yml
}

# apc_remove_field <field>
# Strips a top-level field from apcdeploy.yml (used for E4 invalid-config tests).
apc_remove_field() {
    local field="$1"
    local tmp; tmp="$(mktemp)"
    sed "/^${field}:/d" apcdeploy.yml > "$tmp"
    mv "$tmp" apcdeploy.yml
}

# apc_write_data <content> [filename]
# Writes the data file. Default filename is data.json (matches `init` output).
apc_write_data() {
    local content="$1" file="${2:-data.json}"
    printf '%s' "$content" > "$file"
}

# Convenience: write JSON / YAML / text + sync data_file in apcdeploy.yml.
apc_write_json() {
    local content="$1"
    apc_write_data "$content" data.json
}

apc_write_yaml() {
    local content="$1"
    __apc_set_data_file data.yaml
    apc_write_data "$content" data.yaml
}

apc_write_text() {
    local content="$1"
    __apc_set_data_file data.txt
    apc_write_data "$content" data.txt
}
