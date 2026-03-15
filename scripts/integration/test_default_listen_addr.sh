#!/usr/bin/env bash
# Integration test: when LISTEN_ADDR is unset, server listens on default :8080.
# Starts Postgres and binary, then checks that http://127.0.0.1:8080 responds.
set -e
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"
cd "$ROOT_DIR"

CONTAINER_NAME="featureflag-integration-default-addr"
cleanup() {
  if [[ -n "$BINARY_PID" ]] && kill -0 "$BINARY_PID" 2>/dev/null; then
    kill "$BINARY_PID" 2>/dev/null || true
    wait "$BINARY_PID" 2>/dev/null || true
  fi
  docker rm -f "$CONTAINER_NAME" 2>/dev/null || true
}
trap cleanup EXIT

docker rm -f "$CONTAINER_NAME" 2>/dev/null || true
docker run -d --name "$CONTAINER_NAME" \
  -p 5432:5432 \
  -e POSTGRES_USER=test -e POSTGRES_PASSWORD=test -e POSTGRES_DB=testdb \
  postgres:16-alpine
sleep 2
PG_PORT=$(docker port "$CONTAINER_NAME" 5432 | cut -d: -f2)
export DATABASE_DSN="postgres://test:test@127.0.0.1:$PG_PORT/testdb?sslmode=disable"
export JWT_SECRET="integration-secret"
unset LISTEN_ADDR

"$SCRIPT_DIR/../build.sh" >/dev/null 2>&1
"$ROOT_DIR/bin/featureflag-api" &
BINARY_PID=$!
sleep 2
if ! kill -0 "$BINARY_PID" 2>/dev/null; then
  echo "Binary exited unexpectedly" >&2
  exit 1
fi

CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST http://127.0.0.1:8080/ \
  -H "Content-Type: application/json" \
  -d '{"query":"{ __typename }"}')
if [[ "$CODE" == "000" ]]; then
  echo "Expected server to respond on :8080, got connection failure (code 000)" >&2
  exit 1
fi
echo "OK: server listens on default :8080 (HTTP $CODE)"
