#!/usr/bin/env bash
set -eu

REGION="${1:-ap-northeast-1}"
APP_NAME="${2:-apcdeploy-e2e-test}"

echo "Cleaning up hosted configuration versions for $APP_NAME..."

# Get application ID by tag
APP_ID=$(aws appconfig list-applications --region "$REGION" --query "Items[?Name=='$APP_NAME'].Id" --output text)

if [ -z "$APP_ID" ]; then
    echo "Application $APP_NAME not found"
    exit 0
fi

echo "Application ID: $APP_ID"

# Profile names created by Terraform
PROFILE_NAMES=(
    "json-freeform"
    "json-featureflags"
    "yaml-config"
    "text-config"
    "error-test"
)

for PROFILE_NAME in "${PROFILE_NAMES[@]}"; do
    PROFILE_ID=$(aws appconfig list-configuration-profiles \
        --application-id "$APP_ID" \
        --region "$REGION" \
        --query "Items[?Name=='$PROFILE_NAME'].Id" \
        --output text)

    if [ -z "$PROFILE_ID" ]; then
        continue
    fi

    echo "Cleaning profile $PROFILE_NAME ($PROFILE_ID)..."

    # Get all hosted configuration versions
    VERSIONS=$(aws appconfig list-hosted-configuration-versions \
        --application-id "$APP_ID" \
        --configuration-profile-id "$PROFILE_ID" \
        --region "$REGION" \
        --query "Items[].VersionNumber" \
        --output text 2>/dev/null || echo "")

    for VERSION in $VERSIONS; do
        echo "  Deleting version $VERSION..."
        aws appconfig delete-hosted-configuration-version \
            --application-id "$APP_ID" \
            --configuration-profile-id "$PROFILE_ID" \
            --version-number "$VERSION" \
            --region "$REGION" || true
    done
done

echo "Cleanup complete"
