#!/usr/bin/env bash
#
# Build docker images with tag based on git revision or tag
# and push them to the registry. Called from .jenkins.yaml.
#
# When modifying this script run it through shellcheck
# (https://www.shellcheck.net/) before committing.
#

set -e

script_name=$(basename "$0")

echo "running SUNET/eduseal/$script_name"

# We expect Jenkins to have set GIT_COMMIT for us.
if [ "$GIT_COMMIT" = "" ]; then
    echo "$script_name: GIT_COMMIT is not set, exiting"
    exit 1
fi

VERSION=$(git tag --contains "$GIT_COMMIT" | head -1)
if [ "$VERSION" = "" ]; then
    echo "$script_name: did not find a tag related to revision $GIT_COMMIT, using rev as version"
    VERSION=$GIT_COMMIT
fi

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

    # Also tag and push as :test for rolling test deployments
    local test_tag="${tag%:*}:test"
    docker tag "$tag" "$test_tag"
    docker push "$test_tag"

    echo "$script_name: $name done ($tag + $test_tag)"
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

echo "$script_name: all images built and pushed with version $VERSION (also tagged as :test)"
