#!/usr/bin/env bash
#
# Build docker images with tag based on git revision or tag
# and push them to the registry. Called from .jenkins.yaml.
#
# When modifying this script run it through shellcheck
# (https://www.shellcheck.net/) before committing.
#

set -euo pipefail

script_name=$(basename "$0")

echo "running SUNET/eduseal/$script_name"
echo "$script_name: CI context GITHUB_REF='${GITHUB_REF:-}' TAG_NAME='${TAG_NAME:-}' GIT_BRANCH='${GIT_BRANCH:-}' BRANCH_NAME='${BRANCH_NAME:-}' GIT_COMMIT='${GIT_COMMIT:-}'"

# Jenkins environments differ across jobs/plugins; derive commit robustly.
if [ "${GIT_COMMIT:-}" = "" ]; then
    GIT_COMMIT=$(git rev-parse HEAD)
    echo "$script_name: GIT_COMMIT not set, falling back to HEAD ($GIT_COMMIT)"
fi

echo "$script_name: resolving release version from refs/tags or commit-pointed tags"

# Prefer explicit tag ref from CI environment for release builds.
VERSION=""
if [ "${GITHUB_REF:-}" != "" ] && [[ "$GITHUB_REF" == refs/tags/* ]]; then
    VERSION="${GITHUB_REF#refs/tags/}"
elif [ "${TAG_NAME:-}" != "" ]; then
    VERSION="$TAG_NAME"
elif [ "${GIT_BRANCH:-}" != "" ] && [[ "$GIT_BRANCH" == refs/tags/* ]]; then
    VERSION="${GIT_BRANCH#refs/tags/}"
elif [ "${GIT_BRANCH:-}" != "" ] && [[ "$GIT_BRANCH" == origin/tags/* ]]; then
    VERSION="${GIT_BRANCH#origin/tags/}"
elif [ "${BRANCH_NAME:-}" != "" ] && [[ "$BRANCH_NAME" == tags/* ]]; then
    VERSION="${BRANCH_NAME#tags/}"
else
    VERSION=$(git tag --points-at "$GIT_COMMIT" | grep -E '^(v[0-9]+\.[0-9]+\.[0-9]+|prod-v[0-9]+\.[0-9]+\.[0-9]+)$' | head -1 || true)
fi

if [ "$VERSION" = "" ]; then
    echo "$script_name: no release tag detected for $GIT_COMMIT, skipping docker build"
    exit 0
fi

echo "$script_name: release tag detected, VERSION=$VERSION"

REGISTRY="docker.sunet.se/eduseal"

# --- Build all images in parallel ---

build_and_push() {
    local name="$1"
    local tag="$2"
    local dockerfile="$3"
    shift 3
    local build_args=("$@")

    echo "$script_name: building $tag"
    docker build "${build_args[@]}" --tag "$tag" --file "$dockerfile" .
    docker push "$tag"

    # Also tag and push as :testing for rolling test deployments
    local testing_tag="${tag%:*}:testing"
    docker tag "$tag" "$testing_tag"
    docker push "$testing_tag"

    echo "$script_name: $name done ($tag + $testing_tag)"
}

pids=()

# Build apigw
DOCKER_TAG_APIGW="$REGISTRY/apigw:$VERSION"
build_and_push "apigw" "$DOCKER_TAG_APIGW" "docker/apigw/Dockerfile" \
    --build-arg "SERVICE_NAME=apigw" --build-arg "VERSION=$VERSION" &
pids+=($!)

# Build sealer (lunahsm)
DOCKER_TAG_SEALER="$REGISTRY/sealer_lunahsm:$VERSION"
build_and_push "sealer_lunahsm" "$DOCKER_TAG_SEALER" "docker/sealer/lunahsm/Dockerfile" &
pids+=($!)

# Build validator
DOCKER_TAG_VALIDATOR="$REGISTRY/validator:$VERSION"
build_and_push "validator" "$DOCKER_TAG_VALIDATOR" "docker/validator/Dockerfile" &
pids+=($!)

# Wait for all builds, fail if any fail
failed=0
for pid in "${pids[@]}"; do
    if ! wait "$pid"; then
        failed=1
    fi
done

if [ "$failed" -ne 0 ]; then
    echo "$script_name: one or more builds failed"
    exit 1
fi

echo "$script_name: all images built and pushed with version $VERSION (also tagged as :testing)"
