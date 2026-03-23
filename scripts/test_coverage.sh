#!/usr/bin/env bash
# Runs coverage for production packages and enforces a minimum threshold.
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$ROOT_DIR"

MIN_COVERAGE=80

# Note: in Go, "./internal/..." includes nested subpackages such as
# "./internal/*/mock" and "./internal/testutil". We keep an explicit include
# list to measure coverage of production packages only.
COVERAGE_PKGS=(
  ./internal/audit
  ./internal/auth
  ./internal/config
  ./internal/db
  ./internal/experiments
  ./internal/flags
  ./internal/users
  ./transport/graphql
  ./transport/graphql/middleware
)

echo "== coverage (${COVERAGE_PKGS[*]})"
go test -covermode=atomic -coverprofile=coverage.out "${COVERAGE_PKGS[@]}"

COVERAGE_TOTAL=$(go tool cover -func=coverage.out | awk '/^total:/ {gsub("%","",$3); print $3}')
if awk "BEGIN {exit !($COVERAGE_TOTAL >= $MIN_COVERAGE)}"; then
  echo "Coverage ${COVERAGE_TOTAL}% >= ${MIN_COVERAGE}%"
else
  echo "Coverage ${COVERAGE_TOTAL}% < ${MIN_COVERAGE}%"
  exit 1
fi
