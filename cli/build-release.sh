#!/usr/bin/env bash
# Build all Envo CLI binaries for GitHub Release.
# Run from repo root or from cli/
# Output: dist/envo-{os}-{arch}[.exe]

set -e

cd "$(dirname "$0")"
mkdir -p dist

echo "Building Envo CLI for all platforms..."

GOBUILD="go build -ldflags=-s -w -o"

$GOBUILD dist/envo-windows-amd64.exe -buildvcs=false ./cmd/envo
GOOS=darwin GOARCH=amd64 $GOBUILD dist/envo-darwin-amd64 -buildvcs=false ./cmd/envo
GOOS=darwin GOARCH=arm64 $GOBUILD dist/envo-darwin-arm64 -buildvcs=false ./cmd/envo
GOOS=linux GOARCH=amd64 $GOBUILD dist/envo-linux-amd64 -buildvcs=false ./cmd/envo
GOOS=linux GOARCH=arm64 $GOBUILD dist/envo-linux-arm64 -buildvcs=false ./cmd/envo

echo "Done. Binaries in cli/dist/:"
ls -la dist/
