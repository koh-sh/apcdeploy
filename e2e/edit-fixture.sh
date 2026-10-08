#!/bin/sh
# Fake $EDITOR used by the edit steps in e2e/cases (S8, S9, E3, E5). The
# content to write is supplied via $APCDEPLOY_EDIT_CONTENT so callers can
# vary it per scenario without rewriting this script — keeping the fixture
# stable and check-in-able.
printf '%s' "$APCDEPLOY_EDIT_CONTENT" > "$1"
