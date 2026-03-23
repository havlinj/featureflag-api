#!/usr/bin/env bash
# Integration test: when TLS_CERT_FILE and TLS_KEY_FILE are set, server serves HTTPS.
# Proves LoadTLSConfig is used in main.go.
set -e
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"
cd "$ROOT_DIR"

CONTAINER_NAME="featureflag-integration-tls"
CERT_DIR=$(mktemp -d)
cleanup() {
  if [[ -n "$BINARY_PID" ]] && kill -0 "$BINARY_PID" 2>/dev/null; then
    kill "$BINARY_PID" 2>/dev/null || true
    wait "$BINARY_PID" 2>/dev/null || true
  fi
  docker rm -f "$CONTAINER_NAME" 2>/dev/null || true
  rm -rf "$CERT_DIR"
}
trap cleanup EXIT

openssl req -x509 -newkey rsa:2048 -keyout "$CERT_DIR/key.pem" -out "$CERT_DIR/cert.pem" \
  -days 1 -nodes -subj "/CN=localhost" 2>/dev/null

docker rm -f "$CONTAINER_NAME" 2>/dev/null || true
docker run -d --name "$CONTAINER_NAME" \
  -p 5432:5432 \
  -e POSTGRES_USER=test -e POSTGRES_PASSWORD=test -e POSTGRES_DB=testdb \
  postgres:16-alpine
for _ in {1..60}; do
  if docker exec "$CONTAINER_NAME" pg_isready -U test -d testdb >/dev/null 2>&1; then
    break
  fi
  sleep 0.2
done
PG_PORT=$(docker port "$CONTAINER_NAME" 5432 | cut -d: -f2)
export DATABASE_DSN="postgres://test:test@127.0.0.1:$PG_PORT/testdb?sslmode=disable"
export JWT_SECRET="integration-tls-secret-at-least-32"
export LISTEN_ADDR="127.0.0.1:18443"
export TLS_CERT_FILE="$CERT_DIR/cert.pem"
export TLS_KEY_FILE="$CERT_DIR/key.pem"

"$SCRIPT_DIR/../build.sh" >/dev/null 2>&1
"$ROOT_DIR/bin/featureflag-api" &
BINARY_PID=$!
for _ in {1..60}; do
  if ! kill -0 "$BINARY_PID" 2>/dev/null; then
    echo "Binary exited unexpectedly" >&2
    exit 1
  fi
  if curl -sk -o /dev/null -X POST "https://127.0.0.1:18443/" \
    -H "Content-Type: application/json" \
    -d '{"query":"{ __typename }"}'; then
    break
  fi
  sleep 0.2
done
if ! kill -0 "$BINARY_PID" 2>/dev/null; then
  echo "Binary exited unexpectedly" >&2
  exit 1
fi

CODE=$(curl -sk -o /dev/null -w "%{http_code}" -X POST "https://127.0.0.1:18443/" \
  -H "Content-Type: application/json" \
  -d '{"query":"{ __typename }"}')
if [[ "$CODE" == "000" ]]; then
  echo "Expected HTTPS server to respond on :18443, got connection failure (code 000)" >&2
  exit 1
fi
echo "OK: server serves HTTPS when TLS_CERT_FILE and TLS_KEY_FILE are set"
