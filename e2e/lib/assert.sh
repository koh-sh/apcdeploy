#!/usr/bin/env bash
# Sourced only — inherits e2e-test.sh's set -euo pipefail.
# Assertions used by case scripts. Each assertion calls `fail` (from common.sh)
# on mismatch, which prints the current step as ✗ with a structured message
# and exits 1 — keeping stack traces consistent across the suite.

# assert_contains <haystack> <needle> [label]
assert_contains() {
    local haystack="$1" needle="$2" label="${3:-output}"
    if [[ "$haystack" != *"$needle"* ]]; then
        fail "$label missing substring <$needle>: <$haystack>"
    fi
}

# assert_not_contains <haystack> <needle> [label]
assert_not_contains() {
    local haystack="$1" needle="$2" label="${3:-output}"
    if [[ "$haystack" == *"$needle"* ]]; then
        fail "$label unexpectedly contains <$needle>: <$haystack>"
    fi
}

# assert_match <haystack> <regex> [label] — extended regex via bash =~
assert_match() {
    local haystack="$1" regex="$2" label="${3:-output}"
    if ! [[ "$haystack" =~ $regex ]]; then
        fail "$label did not match /$regex/: <$haystack>"
    fi
}

# assert_jq <file_or_-> <jq_expression>
# Asserts the expression evaluates truthy. `-` reads stdin; otherwise the
# first arg is treated as a file path (also accepts process substitution
# like `<(printf '%s' "$out")`).
assert_jq() {
    local src="$1" expr="$2"
    if [[ "$src" == "-" ]]; then
        if ! jq -e "$expr" >/dev/null; then
            fail "jq predicate failed: <$expr> (stdin)"
        fi
    else
        if ! jq -e "$expr" "$src" >/dev/null; then
            fail "jq predicate failed: <$expr> ($src)"
        fi
    fi
}

# expect_fail <command...>
# Inverts the exit code: passes only when the command fails (any non-zero).
# Suppresses ERR trap because the failure is expected.
expect_fail() {
    if "$@" >/dev/null 2>&1; then
        fail "command unexpectedly succeeded: $*"
    fi
}

# expect_exit <expected_code> <command...>
# Asserts the command exits with exactly <expected_code>.
expect_exit() {
    local expected="$1"; shift
    local rc=0
    "$@" >/dev/null 2>&1 || rc=$?
    if [[ "$rc" -ne "$expected" ]]; then
        fail "expected exit $expected, got $rc: $*"
    fi
}

# assert_grep <file> <pattern>
assert_grep() {
    local file="$1" pattern="$2"
    if ! grep -qE "$pattern" "$file"; then
        fail "pattern /$pattern/ not found in $file"
    fi
}

# refute_line_match <haystack> <extended_regex> [label]
# Asserts that no line of <haystack> matches the extended regex. Used for
# anchored patterns like "^Description" where assert_not_contains is too
# loose (would miss a match at the very first line).
refute_line_match() {
    local haystack="$1" regex="$2" label="${3:-output}"
    if printf '%s' "$haystack" | grep -qE "$regex"; then
        fail "$label unexpectedly matched /$regex/"
    fi
}
