#!/bin/sh
# Fake $EDITOR used by S7/E3/E5 in e2e-test.sh. The content to write is
# supplied via $APCDEPLOY_EDIT_CONTENT so callers can vary it per scenario
# without rewriting this script — keeping the fixture stable and check-in-able.
printf '%s' "$APCDEPLOY_EDIT_CONTENT" > "$1"
