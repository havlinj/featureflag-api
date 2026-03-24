#!/usr/bin/env bash
# Integration test: binary must exit non-zero when TLS cert/key files are invalid.
set -e
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"
cd "$ROOT_DIR"

if [[ ! -x "$ROOT_DIR/bin/featureflag-api" ]]; then
  "$SCRIPT_DIR/../build.sh" >/dev/null 2>&1
fi

export JWT_SECRET="integration-secret-at-least-32-bytes"
export DATABASE_DSN="postgres://localhost:5432/db?sslmode=disable"
export TLS_CERT_FILE="/tmp/does-not-exist-cert.pem"
export TLS_KEY_FILE="/tmp/does-not-exist-key.pem"
if "$ROOT_DIR/bin/featureflag-api" 2>/dev/null; then
  echo "Expected binary to exit non-zero when TLS files are invalid" >&2
  exit 1
fi
echo "OK: binary exits non-zero when TLS files are invalid"
