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

# Build and push apigw
DOCKER_TAG_APIGW="$REGISTRY/apigw:$VERSION"
echo "$script_name: building $DOCKER_TAG_APIGW"
docker build --build-arg "SERVICE_NAME=apigw" --build-arg "VERSION=$VERSION" --tag "$DOCKER_TAG_APIGW" --file docker/apigw/Dockerfile .
docker push "$DOCKER_TAG_APIGW"

# Build and push sealer (lunahsm)
DOCKER_TAG_SEALER="$REGISTRY/sealer_lunahsm:$VERSION"
echo "$script_name: building $DOCKER_TAG_SEALER"
docker build --tag "$DOCKER_TAG_SEALER" --file docker/sealer/lunahsm/Dockerfile .
docker push "$DOCKER_TAG_SEALER"

# Build and push validator
DOCKER_TAG_VALIDATOR="$REGISTRY/validator:$VERSION"
echo "$script_name: building $DOCKER_TAG_VALIDATOR"
docker build --tag "$DOCKER_TAG_VALIDATOR" --file docker/validator/Dockerfile .
docker push "$DOCKER_TAG_VALIDATOR"

echo "$script_name: all images built and pushed with version $VERSION"
