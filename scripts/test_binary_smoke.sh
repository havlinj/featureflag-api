#!/usr/bin/env bash
# Smoke test against the real application binary:
# build binary, start Postgres (Docker), start app, run one scenario (login → create flag → evaluate), then tear down.
# Requires: go, docker, curl.
set -e
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$ROOT_DIR"

LISTEN_ADDR="127.0.0.1:18080"
BASE_URL="http://$LISTEN_ADDR"
JWT_SECRET="smoke-test-secret-at-least-32-bytes"
ADMIN_EMAIL="admin@smoke.test"
ADMIN_PASSWORD="adminpass"
CONTAINER_NAME="featureflag-smoke-db"

cleanup() {
  if [[ -n "$BINARY_PID" ]] && kill -0 "$BINARY_PID" 2>/dev/null; then
    kill "$BINARY_PID" 2>/dev/null || true
    wait "$BINARY_PID" 2>/dev/null || true
  fi
  if docker ps -q -f "name=$CONTAINER_NAME" 2>/dev/null | grep -q .; then
    docker rm -f "$CONTAINER_NAME" 2>/dev/null || true
  fi
}
trap cleanup EXIT

echo "== 1/6 starting Postgres container"
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

echo "== 2/6 building binary"
"$SCRIPT_DIR/build.sh" >/dev/null

echo "== 3/6 starting application"
export JWT_SECRET
export LISTEN_ADDR
"$ROOT_DIR/bin/featureflag-api" &
BINARY_PID=$!
for _ in {1..60}; do
  if ! kill -0 "$BINARY_PID" 2>/dev/null; then
    echo "Binary exited unexpectedly" >&2
    exit 1
  fi
  if curl -s -o /dev/null -X POST "$BASE_URL/" \
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

echo "== 4/6 seeding admin user"
PASSHASH=$(go run ./cmd/seedpass "$ADMIN_PASSWORD" | tr -d '\n')
docker exec "$CONTAINER_NAME" psql -U test -d testdb -c \
  "INSERT INTO users (email, role, password_hash) VALUES ('$ADMIN_EMAIL', 'admin', '$PASSHASH') ON CONFLICT (email) DO NOTHING"

echo "== 5/6 running smoke scenario (login → createFlag → evaluateFlag)"
LOGIN_RESP=$(curl -s -X POST "$BASE_URL/" \
  -H "Content-Type: application/json" \
  -d "{\"query\":\"mutation Login(\$input: LoginInput!) { login(input: \$input) { token } }\",\"variables\":{\"input\":{\"email\":\"$ADMIN_EMAIL\",\"password\":\"$ADMIN_PASSWORD\"}}}")
TOKEN=$(echo "$LOGIN_RESP" | grep -o '"token":"[^"]*"' | cut -d'"' -f4)
if [[ -z "$TOKEN" ]]; then
  echo "Login failed. Response: $LOGIN_RESP" >&2
  exit 1
fi

CREATE_RESP=$(curl -s -X POST "$BASE_URL/" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"query":"mutation CreateFlag($input: CreateFlagInput!) { createFlag(input: $input) { id key enabled environment } }","variables":{"input":{"key":"smoke-flag","description":"Smoke test","environment":"dev"}}}')
if echo "$CREATE_RESP" | grep -q '"errors"'; then
  echo "createFlag failed. Response: $CREATE_RESP" >&2
  exit 1
fi
if ! echo "$CREATE_RESP" | grep -q '"smoke-flag"'; then
  echo "createFlag: expected key smoke-flag. Response: $CREATE_RESP" >&2
  exit 1
fi

UPDATE_RESP=$(curl -s -X POST "$BASE_URL/" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"query":"mutation UpdateFlag($input: UpdateFlagInput!) { updateFlag(input: $input) { key enabled } }","variables":{"input":{"key":"smoke-flag","enabled":true}}}')
if echo "$UPDATE_RESP" | grep -q '"errors"'; then
  echo "updateFlag failed. Response: $UPDATE_RESP" >&2
  exit 1
fi

EVAL_RESP=$(curl -s -X POST "$BASE_URL/" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"query":"query EvaluateFlag($key: String!, $evaluationContext: EvaluationContextInput!) { evaluateFlag(key: $key, evaluationContext: $evaluationContext) }","variables":{"key":"smoke-flag","evaluationContext":{"userId":"user-1"}}}')
if echo "$EVAL_RESP" | grep -q '"errors"'; then
  echo "evaluateFlag failed. Response: $EVAL_RESP" >&2
  exit 1
fi
if ! echo "$EVAL_RESP" | grep -q '"evaluateFlag":true'; then
  echo "evaluateFlag: expected true. Response: $EVAL_RESP" >&2
  exit 1
fi

echo "== 6/6 tear down (trap cleans binary and container)"
echo "OK: binary smoke test passed."
