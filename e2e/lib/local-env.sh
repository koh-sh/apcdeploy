#!/usr/bin/env bash
# Point every AWS SDK / Terraform call at the local MiniStack emulator.
# Sourced by lib/common.sh (E2E_TARGET=local) and the e2e-local-setup mise
# task, so the e2e suite and Terraform share one definition.

E2E_LOCAL_ENDPOINT="http://localhost:4566"

# Drop everything that could redirect calls elsewhere or supply real
# credentials. Service-specific AWS_ENDPOINT_URL_* variables take precedence
# over AWS_ENDPOINT_URL in the SDKs, so they are removed as well.
# shellcheck disable=SC2046  # word splitting of variable names is intended
unset AWS_PROFILE AWS_DEFAULT_PROFILE AWS_SESSION_TOKEN AWS_SECURITY_TOKEN \
    AWS_ROLE_ARN AWS_WEB_IDENTITY_TOKEN_FILE \
    AWS_CONTAINER_CREDENTIALS_FULL_URI AWS_CONTAINER_CREDENTIALS_RELATIVE_URI \
    AWS_IGNORE_CONFIGURED_ENDPOINT_URLS \
    $(compgen -e | grep '^AWS_ENDPOINT_URL_' || true)

export AWS_ENDPOINT_URL="$E2E_LOCAL_ENDPOINT"
export AWS_ACCESS_KEY_ID=test
export AWS_SECRET_ACCESS_KEY=test
export AWS_CONFIG_FILE=/dev/null
export AWS_SHARED_CREDENTIALS_FILE=/dev/null
