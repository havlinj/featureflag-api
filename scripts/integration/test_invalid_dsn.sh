#!/usr/bin/env bash
# Integration test: binary must exit with non-zero when database is unreachable (invalid DSN).
set -e
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"
cd "$ROOT_DIR"

"$SCRIPT_DIR/../build.sh" >/dev/null 2>&1

export JWT_SECRET="test-secret"
export DATABASE_DSN="postgres://nonexistenthost:5432/nonexistent?sslmode=disable"
if "$ROOT_DIR/bin/featureflag-api" 2>/dev/null; then
  echo "Expected binary to exit non-zero when database is unreachable" >&2
  exit 1
fi
echo "OK: binary exits non-zero when DSN is invalid/unreachable"
