#!/usr/bin/env bash
# Shared environment, output helpers, and ERR/EXIT traps for e2e tests.
# Sourced only by e2e-test.sh — do not `set -e` here; inherits the runner's
# `set -euo pipefail`. Sourced before any case file runs.

# ---- Global env -----------------------------------------------------------

E2E_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
APCDEPLOY_BIN="${APCDEPLOY_BIN:-$E2E_ROOT/apcdeploy}"
FAKE_EDITOR="${FAKE_EDITOR:-$E2E_ROOT/edit-fixture.sh}"
APP="${E2E_APP:-apcdeploy-e2e-test}"
REGION="${E2E_REGION:-ap-northeast-1}"
STRATEGY="${E2E_STRATEGY:-E2E-Test-Strategy}"
SLOW_STRATEGY="${E2E_SLOW_STRATEGY:-E2E-Slow-Strategy}"

export E2E_ROOT APCDEPLOY_BIN FAKE_EDITOR APP REGION STRATEGY SLOW_STRATEGY

# ---- Color (NO_COLOR + TTY aware) ----------------------------------------

# shellcheck disable=SC2034  # palette intentionally complete; C_YELLOW reserved for future Warn-equivalent output
if [[ -t 2 ]] && [[ -z "${NO_COLOR:-}" ]]; then
    C_RESET=$'\033[0m'
    C_RED=$'\033[31m'
    C_GREEN=$'\033[32m'
    C_YELLOW=$'\033[33m'
    C_BLUE=$'\033[34m'
    C_DIM=$'\033[2m'
    C_BOLD=$'\033[1m'
else
    C_RESET="" C_RED="" C_GREEN="" C_YELLOW="" C_BLUE="" C_DIM="" C_BOLD=""
fi

# ---- Section / step state ------------------------------------------------

__SECTION_ID=""
__STEP=""
__START_TIME=$(date +%s)

# Single shared file every apc_* wrapper redirects stderr into. Using a
# fixed path (vs per-call mktemp) is required because wrappers may run in
# $(...) subshells, where any locally-assigned variable is lost when the
# subshell exits — leaving the parent's ERR trap with no way to find the
# capture file. The file is reused across calls; each wrapper truncates it
# on entry so only the most recent command's stderr is shown on failure.
__STDERR_FILE="${TMPDIR:-/tmp}/apcdeploy-e2e-stderr.$$"
: > "$__STDERR_FILE"
export __STDERR_FILE

# Shared step / section counters. Each subshell-per-section finalizes a
# step by appending a line to __STEPS_FILE, and `section()` appends to
# __SECTIONS_FILE; the parent shell's EXIT trap reads the line counts for
# the final summary. Files are needed because per-section subshells can't
# write back into the parent's variables.
__STEPS_FILE="${TMPDIR:-/tmp}/apcdeploy-e2e-steps.$$"
: > "$__STEPS_FILE"
__SECTIONS_FILE="${TMPDIR:-/tmp}/apcdeploy-e2e-sections.$$"
: > "$__SECTIONS_FILE"
export __STEPS_FILE __SECTIONS_FILE

# Print the canonical section header. Closes the previous step (if any).
section() {
    __finalize_step ok
    __SECTION_ID="$1"
    local title="$2"
    echo 1 >> "$__SECTIONS_FILE"
    local rule="══════════════════════════════════════════════════════════════"
    {
        printf '\n%s%s%s\n' "$C_BLUE" "$rule" "$C_RESET"
        printf ' %s%s%s  %s%s%s\n' "$C_BOLD" "$__SECTION_ID" "$C_RESET" "$C_BOLD" "$title" "$C_RESET"
        printf '%s%s%s\n' "$C_BLUE" "$rule" "$C_RESET"
    } >&2
}

# Open a new step. The previous step (if any) is finalized as ✓.
# A failing command between `step` calls trips ERR trap, which finalizes ✗.
step() {
    __finalize_step ok
    __STEP="$1"
}

__finalize_step() {
    local result="$1"  # ok|fail
    [[ -z "$__STEP" ]] && return 0
    if [[ "$result" == "ok" ]]; then
        printf '  %s✓%s %s\n' "$C_GREEN" "$C_RESET" "$__STEP" >&2
        # Append to shared file so the parent shell's __on_exit can count
        # steps from per-section subshells.
        echo 1 >> "$__STEPS_FILE"
    fi
    __STEP=""
}

# Hard fail with a custom message (used by assertions).
fail() {
    local msg="$1"
    if [[ -n "$__STEP" ]]; then
        printf '  %s✗%s %s\n' "$C_RED" "$C_RESET" "$__STEP" >&2
    fi
    printf '    %s%s%s\n' "$C_RED" "$msg" "$C_RESET" >&2
    __STEP=""  # already reported
    exit 1
}

# ERR trap: print the current step as failed with file:line context.
__on_err() {
    local rc=$?
    local lineno="$1"
    local source="$2"
    if [[ -n "$__STEP" ]]; then
        printf '  %s✗%s %s\n' "$C_RED" "$C_RESET" "$__STEP" >&2
        printf '    %sat %s:%s (exit %d)%s\n' \
            "$C_DIM" "${source#"$E2E_ROOT/"}" "$lineno" "$rc" "$C_RESET" >&2
        # Mark already-reported so the subshell's EXIT trap doesn't
        # re-finalize this step as ✓ (and double-count it in __STEPS_FILE).
        __STEP=""
    fi
    if [[ -n "${__STDERR_FILE:-}" && -s "$__STDERR_FILE" ]]; then
        printf '    %s---- last stderr ----%s\n' "$C_DIM" "$C_RESET" >&2
        tail -n 10 "$__STDERR_FILE" | sed 's/^/    /' >&2
    fi
}

# EXIT trap: emit the final summary on clean exit; cleanup is left to subshells.
__on_exit() {
    local rc=$?
    if [[ $rc -eq 0 ]]; then
        __finalize_step ok
        local total_steps=0
        local total_sections=0
        if [[ -f "$__STEPS_FILE" ]]; then
            total_steps=$(wc -l < "$__STEPS_FILE" | tr -d ' ')
        fi
        if [[ -f "$__SECTIONS_FILE" ]]; then
            total_sections=$(wc -l < "$__SECTIONS_FILE" | tr -d ' ')
        fi
        local elapsed=$(( $(date +%s) - __START_TIME ))
        local mins=$((elapsed / 60))
        local secs=$((elapsed % 60))
        local rule="──────────────────────────────────────────────────────────────"
        {
            printf '\n%s%s%s\n' "$C_DIM" "$rule" "$C_RESET"
            printf ' %s✓%s %d sections, %d steps passed (%dm %ds)\n' \
                "$C_GREEN" "$C_RESET" "$total_sections" "$total_steps" "$mins" "$secs"
            printf '%s%s%s\n' "$C_DIM" "$rule" "$C_RESET"
        } >&2
    fi
    [[ -n "${__STDERR_FILE:-}" ]] && rm -f "$__STDERR_FILE"
    [[ -n "${__STEPS_FILE:-}" ]] && rm -f "$__STEPS_FILE"
    [[ -n "${__SECTIONS_FILE:-}" ]] && rm -f "$__SECTIONS_FILE"
    # Clean up per-section tempdirs registered by e2e-test.sh.
    # `${arr[@]:-default}` is not valid for arrays in bash; gate on
    # existence (`${TMPDIRS+x}`) before reading the length.
    if [[ -z "${E2E_KEEP_TMP:-}" && -n "${TMPDIRS+x}" && ${#TMPDIRS[@]} -gt 0 ]]; then
        rm -rf "${TMPDIRS[@]}"
    fi
}

trap '__on_err $LINENO "${BASH_SOURCE[0]}"' ERR
trap '__on_exit' EXIT
