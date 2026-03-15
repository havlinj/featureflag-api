#!/usr/bin/env bash
# Runs integration tests. Requires Docker (testcontainers start Postgres automatically).
set -e
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$ROOT_DIR"

echo "== integration tests (go test -tags=integration ./test/integration/...)"
go test -tags=integration -v ./test/integration/...
echo "OK"
