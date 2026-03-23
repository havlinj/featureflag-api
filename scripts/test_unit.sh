#!/usr/bin/env bash
# Runs unit tests (excludes integration tests that require -tags=integration).
set -e
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$ROOT_DIR"

echo "== unit tests (./internal/... ./transport/...)"
go test -race ./internal/... ./transport/...
echo "OK"
