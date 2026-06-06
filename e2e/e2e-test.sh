#!/usr/bin/env bash
# Entry point for the apcdeploy E2E suite.
#
# Usage:
#   ./e2e-test.sh              # run all sections in order
#   ./e2e-test.sh S1 S3        # run only the listed sections
#   E2E_KEEP_TMP=1 ./e2e-test.sh  # don't clean per-section tempdirs (debug)
#
# Each section runs in its own tempdir to prevent local-state leakage between
# sections. AWS-side state coupling (shared profile/env in Terraform) is NOT
# fixed here — that requires resource-level isolation in terraform/main.tf.

set -euo pipefail

E2E_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$E2E_DIR/.." && pwd)"

# Build the binary once at the canonical location used by lib/common.sh.
cd "$REPO_ROOT"
go build -o "$E2E_DIR/apcdeploy"
cd "$E2E_DIR"

# shellcheck source=lib/common.sh
source "$E2E_DIR/lib/common.sh"
# shellcheck source=lib/assert.sh
source "$E2E_DIR/lib/assert.sh"
# shellcheck source=lib/apc.sh
source "$E2E_DIR/lib/apc.sh"

# Section registry — order is significant (AWS state coupling).
# S3 must run before S7 / E3 perturb json-freeform/{dev,staging}.
# Format: <ID>:<case-file-relative-to-cases/>
SECTIONS=(
    "S1:s1_workflow.sh"
    "S2:s2_content_types.sh"
    "S3:s3_multiconfig.sh"
    "S4:s4_deployment_control.sh"
    "S5:s5_config.sh"
    "S6:s6_ci.sh"
    "S7:s7_rollback.sh"
    "S8:s8_edit.sh"
    "S9:s9_description.sh"
    "S10:s10_validate.sh"
    "E1:e1_resource_errors.sh"
    "E2:e2_validation.sh"
    "E3:e3_constraints.sh"
    "E4:e4_file_errors.sh"
    "E5:e5_edge_cases.sh"
)

# Filter to the IDs passed on the CLI, if any.
filter_sections() {
    if [[ $# -eq 0 ]]; then
        printf '%s\n' "${SECTIONS[@]}"
        return
    fi
    local want=" $* "
    for entry in "${SECTIONS[@]}"; do
        local id="${entry%%:*}"
        if [[ "$want" == *" $id "* ]]; then
            printf '%s\n' "$entry"
        fi
    done
}

mapfile -t SELECTED < <(filter_sections "$@")
if [[ ${#SELECTED[@]} -eq 0 ]]; then
    printf 'No matching sections for: %s\n' "$*" >&2
    printf 'Available: %s\n' "${SECTIONS[*]%%:*}" >&2
    exit 1
fi

# Run each case in its own tempdir + subshell so failures don't leak state
# into the next section. The subshell also re-exports the ERR/EXIT traps
# (bash inherits ERR via `set -E`, EXIT is per-shell).
#
# Per-section tempdirs are tracked in TMPDIRS and cleaned up by common.sh's
# __on_exit — registering a `trap "rm -rf …" EXIT` per iteration would
# overwrite (not chain) the parent EXIT trap and disable the final summary.
set -E
# Visible to common.sh's __on_exit (same shell) without export — bash does
# not export arrays through the environment anyway.
TMPDIRS=()
for entry in "${SELECTED[@]}"; do
    id="${entry%%:*}"
    file="${entry#*:}"
    case_path="$E2E_DIR/cases/$file"
    if [[ ! -f "$case_path" ]]; then
        printf 'Missing case file: %s\n' "$case_path" >&2
        exit 1
    fi
    tmpdir="$(mktemp -d "${TMPDIR:-/tmp}/apcdeploy-e2e-${id}.XXXXXX")"
    TMPDIRS+=("$tmpdir")
    (
        cd "$tmpdir"
        # Finalize the last step on subshell exit so its ✓ is printed
        # (subshell variables don't propagate, so the parent's __on_exit
        # cannot see the section's trailing step).
        trap '__finalize_step ok' EXIT
        # shellcheck source=/dev/null
        source "$case_path"
    )
done
