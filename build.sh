#!/bin/sh
VERSION=${1:-"dev"}
COMMIT=$(git rev-parse HEAD 2>/dev/null || echo "unknown")
BUILD_DATE=$(date -u +%Y-%m-%dT%H:%M:%SZ)

echo "Building goddns ${VERSION} (${COMMIT})"

go build -ldflags "-X main.version=${VERSION} -X main.commit=${COMMIT} -X main.buildDate=${BUILD_DATE}" -o goddns ./cmd/goddns

echo "Build completed. Run './goddns -v' to check version."
