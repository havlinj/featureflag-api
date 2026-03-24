#!/usr/bin/env bash
# Integration test: binary must exit non-zero when JWT_SECRET is too short.
set -e
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"
cd "$ROOT_DIR"

if [[ ! -x "$ROOT_DIR/bin/featureflag-api" ]]; then
  "$SCRIPT_DIR/../build.sh" >/dev/null 2>&1
fi

export JWT_SECRET="short-secret"
export DATABASE_DSN="postgres://localhost:5432/db?sslmode=disable"
if "$ROOT_DIR/bin/featureflag-api" 2>/dev/null; then
  echo "Expected binary to exit non-zero when JWT_SECRET is too short" >&2
  exit 1
fi
echo "OK: binary exits non-zero when JWT_SECRET is too short"
