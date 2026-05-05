#!/usr/bin/env bash
# S2: ContentType handling — FeatureFlags JSON, YAML, plain text.
# Each profile is deployed and then round-tripped to confirm the content
# type is honored end to end.
#
# FeatureFlags uses pull rather than get because the AppConfig Data API
# returns evaluated flag values (e.g. `{"test":{}}`) — the raw flags
# definition is only retrievable via the hosted-version endpoint, which
# pull uses.

section "S2" "Content types"

step "FeatureFlags profile deploys and round-trips through pull"
apc_init json-featureflags dev
apc_use_strategy
apc_write_json '{"version":"1","flags":{"test":{"name":"test"}}}'
apc_quiet run --wait-bake --silent
# Mutate the local file so pull has something to overwrite.
apc_write_json '{"version":"1","flags":{"placeholder":{"name":"x"}}}'
apc_quiet pull --silent
assert_jq data.json '.flags.test.name == "test"'

step "YAML profile deploys and round-trips literal content"
apc_init yaml-config dev
apc_use_strategy
apc_write_yaml $'v: 1\nk: v'
apc_quiet run --wait-bake --silent
yaml_out=$(apc_stdout get --silent --yes)
assert_contains "$yaml_out" "v: 1" "yaml body"
assert_contains "$yaml_out" "k: v" "yaml body"

step "Text profile deploys and round-trips literal content"
apc_init text-config dev
apc_use_strategy
apc_write_text 'hello-text-payload'
apc_quiet run --wait-bake --silent
text_out=$(apc_stdout get --silent --yes)
assert_contains "$text_out" "hello-text-payload" "text body"
