#!/bin/bash -e
# Builds and stages the platform

SERVICE_NAME=platform
cd $(dirname $0)/..

echo "Build and stage"
(
    set -o xtrace

    rm -rf ./build || true
    mkdir -p build/scripts

    # Write the version.json file (this is then embedded during the build).
    HASH=$(git rev-parse HEAD)
    HASH_SHORT=${HASH:0:7}
    BUILD_NUMBER=$(git rev-list --all --count $HASH)
    cat <<EOF >version.json
{
    "build": {
        "buildNumber": $BUILD_NUMBER,
        "hash": "$HASH",
        "shortHash": "$HASH_SHORT"
    }
}
EOF

    # Build the binary
    env GOOS=linux GOARCH=amd64 go build

    # Copy deployables
    cp scripts/deploy.sh build/scripts/
    cp $SERVICE_NAME build/
    cp config-production.json build/
)

exit 0
