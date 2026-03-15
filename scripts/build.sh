#!/usr/bin/env bash
# Builds the application binary into ./bin/featureflag-api.
set -e
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$ROOT_DIR"

BIN_DIR="$ROOT_DIR/bin"
mkdir -p "$BIN_DIR"
echo "== building ./bin/featureflag-api"
go build -o "$BIN_DIR/featureflag-api" ./cmd
echo "OK: $BIN_DIR/featureflag-api"
