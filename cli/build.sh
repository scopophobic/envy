#!/usr/bin/env bash
# Build Envo CLI for current platform (macOS/Linux).
# Run from cli/: ./build.sh
# Output: envo in cli/

set -e
cd "$(dirname "$0")"
go build -o envo ./cmd/envo
echo "Built: ./envo"
echo "Run: ./envo whoami"
